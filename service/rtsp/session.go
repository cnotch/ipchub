// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/ipchub/network/socket/buffered"
	"github.com/cnotch/ipchub/network/websocket"
	"github.com/cnotch/ipchub/provider/auth"
	"github.com/cnotch/ipchub/provider/security"
	"github.com/cnotch/ipchub/stats"
	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/xlog"
	"github.com/emitter-io/address"
	"github.com/pixelbender/go-sdp/sdp"
)

const (
	realm = config.Name
)

const (
	statusInit = iota
	statusReady
	statusPlaying
	statusRecording
)

var buffers = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024*2))
	},
}

// Session RTSP 会话
type Session struct {
	// 创建时设置
	svr      *Server
	logger   *xlog.Logger
	closed   bool
	lsession string // 本地会话标识
	timeout  time.Duration
	conn     *buffered.Conn
	lockW    sync.Mutex

	wsconn websocket.Conn

	authMode auth.Mode
	nonce    string
	user     *auth.User

	// DESCRIBE，或 ANNOUNCE 后设置
	url      *url.URL
	path     string
	rawSdp   string
	sdp      *sdp.Session
	aControl string
	vControl string
	aCodec   string
	vCodec   string
	mode     SessionMode

	// Setup 后设置
	transport RTPTransport

	// 启动流媒体传输后设置
	status   int            // session状态
	stream   mediaStream    // 媒体流
	consumer media.Consumer // 消费者
}

func newSession(svr *Server, conn net.Conn) *Session {

	session := &Session{
		svr:      svr,
		lsession: security.NewID().Base64(),
		timeout:  config.NetTimeout(),
		conn: buffered.NewConn(conn,
			buffered.FlushRate(config.NetFlushRate()),
			buffered.BufferSize(config.NetBufferSize())),
		mode: UnknownSession,
		transport: RTPTransport{
			Mode: PlaySession, // 默认为播放
			Type: RTPUnknownTrans,
		},
		authMode: config.RtspAuthMode(),
		nonce:    security.NewID().MD5(),
		status:   statusInit,
		stream:   defaultStream,
		consumer: defaultConsumer,
	}

	if wsc, ok := conn.(websocket.Conn); ok { // 如果是WebSocket，有http进行验证
		session.authMode = auth.NoneAuth
		session.wsconn = wsc
		session.path = wsc.Path()
		session.user = auth.Get(wsc.Username())
	}

	ipaddr, _ := address.Parse(conn.RemoteAddr().String(), 80)
	// 如果是本机IP，不验证；以便ffmpeg本机rtsp->rtmp
	if utils.IsLocalhostIP(ipaddr.IP) {
		session.authMode = auth.NoneAuth
	}

	for i := rtpChannelMin; i < rtpChannelCount; i++ {
		session.transport.Channels[i] = -1
		session.transport.ClientPorts[i] = -1
	}
	session.logger = svr.logger.With(xlog.Fields(
		xlog.F("session", session.lsession)))

	return session
}

// Addr Session地址
func (s *Session) Addr() string {
	return s.conn.RemoteAddr().String()
}

// Consume 消费媒体包
func (s *Session) Consume(p Pack) {
	s.consumer.Consume(p)
}

// Close 关闭会话
func (s *Session) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true
	s.conn.Close()
	return nil
}

func (s *Session) process() {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Errorf("session panic; %v \n %s", r, debug.Stack())
		}

		stats.RtspConns.Release()
		s.Close()
		s.consumer.Close()
		s.stream.Close()

		// 重置到初始状态
		s.conn = nil
		s.status = statusInit
		s.stream = defaultStream
		s.consumer = defaultConsumer
		s.logger.Infof("close rtsp session")
	}()

	s.logger.Infof("open rtsp session")
	stats.RtspConns.Add() // 增加一个 RTSP 连接计数
	reader := s.conn.Reader()

	for !s.closed {
		deadLine := time.Time{}
		if s.timeout > 0 {
			deadLine = time.Now().Add(s.timeout)
		}
		if err := s.conn.SetReadDeadline(deadLine); err != nil {
			s.logger.Error(err.Error())
			break
		}

		err := receive(s.logger, reader, s.transport.Channels[:], s)
		if err != nil {
			if err == io.EOF { // 如果客户端断开提醒
				s.logger.Warn("The client actively disconnects")
			} else if !s.closed { // 如果主动关闭，不提示
				s.logger.Error(err.Error())
			}
			break
		}
	}
}

// receiveHandler.onPack
func (s *Session) onPack(pack *RTPPack) (err error) {
	return s.stream.WritePacket(pack)
}

// receiveHandler.onResponse
func (s *Session) onResponse(resp *Response) (err error) {
	// 忽略,服务器不会主动发起请求
	return
}

// receiveHandler.onRequest
func (s *Session) onRequest(req *Request) (err error) {
	resp := s.newResponse(StatusOK, req)
	// 预处理
	continueProcess, err := s.onPreprocess(resp, req)
	if !continueProcess {
		return err
	}

	switch req.Method {
	case MethodDescribe:
		s.onDescribe(resp, req)
	case MethodAnnounce:
		s.onAnnounce(resp, req)
	case MethodSetup:
		s.onSetup(resp, req)
	case MethodRecord:
		s.onRecord(resp, req)
	case MethodPlay:
		return s.onPlay(resp, req) // play 发送流媒体不在当前 routine，需要先回复
	default:
		// 状态不支持的方法
		resp.StatusCode = StatusMethodNotValidInThisState
	}

	// 发送响应
	err = s.response(resp)
	return err
}

func (s *Session) onDescribe(resp *Response, req *Request) {

	// TODO: 检查 accept 中的类型是否包含 sdp
	s.url = req.URL
	if s.wsconn == nil { // websocket访问的路径有ws://路径表示
		s.path = utils.CanonicalPath(req.URL.Path)
	}

	stream := media.GetOrCreate(s.path)
	if stream == nil {
		resp.StatusCode = StatusNotFound
		return
	}

	if !s.checkPermission(auth.PullRight) {
		resp.StatusCode = StatusForbidden
		return
	}

	// 从流中取 sdp
	sdpRaw := stream.Attr("sdp")
	if len(sdpRaw) == 0 {
		resp.StatusCode = StatusNotFound
		return
	}
	err := s.parseSdp(sdpRaw)
	if err != nil { // TODO：需要更好的处理方式
		resp.StatusCode = StatusNotFound
		return
	}

	resp.Header.Set(FieldContentType, "application/sdp")
	resp.Body = s.rawSdp
	s.mode = PlaySession // 标记为播放会话
}

func (s *Session) onAnnounce(resp *Response, req *Request) {

	// 检查 Content-Type: application/sdp
	if req.Header.Get(FieldContentType) != "application/sdp" {
		resp.StatusCode = StatusBadRequest // TODO:更合适的代码
		return
	}

	s.url = req.URL
	s.path = utils.CanonicalPath(req.URL.Path)

	if !s.checkPermission(auth.PushRight) {
		resp.StatusCode = StatusForbidden
		return
	}

	// 从流中取 sdp
	err := s.parseSdp(req.Body)
	if err != nil {
		resp.StatusCode = StatusBadRequest
		return
	}

	s.mode = RecordSession // 标记为录像会话
}

func (s *Session) onSetup(resp *Response, req *Request) {
	// a=control:streamid=1
	// a=control:rtsp://192.168.1.165/trackID=1
	// a=control:?ctype=video
	setupURL := &url.URL{}
	*setupURL = *req.URL
	if setupURL.Port() == "" {
		setupURL.Host = fmt.Sprintf("%s:554", setupURL.Host)
	}
	setupPath := setupURL.String()

	//setupPath = setupPath[strings.LastIndex(setupPath, "/")+1:]
	vPath, err := getControlPath(s.vControl)
	if err != nil {
		resp.StatusCode = StatusInternalServerError
		resp.Status = "Invalid VControl"
		return
	}

	aPath, err := getControlPath(s.aControl)
	if err != nil {
		resp.StatusCode = StatusInternalServerError
		resp.Status = "Invalid AControl"
		return
	}

	ts := req.Header.Get(FieldTransport)
	resp.Header.Set(FieldTransport, ts) // 先回写transport

	// 检查控制路径
	chindex := -1
	if setupPath == aPath || (aPath != "" && strings.LastIndex(setupPath, aPath) == len(setupPath)-len(aPath)) {
		chindex = int(ChannelAudio)
	} else if setupPath == vPath || (vPath != "" && strings.LastIndex(setupPath, vPath) == len(setupPath)-len(vPath)) {
		chindex = int(ChannelVideo)
	} else { // 找不到被 Setup 的资源
		resp.StatusCode = StatusInternalServerError
		resp.Status = fmt.Sprintf("SETUP Unkown control:%s", setupPath)
		return
	}

	err = s.transport.ParseTransport(chindex, ts)
	if err != nil {
		resp.StatusCode = StatusInvalidParameter
		resp.Status = err.Error()
		return
	}

	// 检查和以前的命令是否一致
	if s.mode == UnknownSession {
		s.mode = s.transport.Mode
	}

	if s.mode != s.transport.Mode {
		resp.StatusCode = StatusInvalidParameter
		if s.mode == PlaySession {
			resp.Status = "Current state can't setup as record"
		} else {
			resp.Status = "Current state can't setup as play"
		}
		return
	}

	// record 只支持 TCP 单播
	if s.mode == RecordSession {
		// 检查用户权限
		if !s.checkPermission(auth.PushRight) {
			resp.StatusCode = StatusForbidden
			return
		}

		if s.transport.Type != RTPTCPUnicast {
			resp.StatusCode = StatusUnsupportedTransport
			resp.Status = "when mode = record，only support tcp unicast"
		} else {
			if s.status < statusReady { // 初始状态切换到Ready
				s.status = statusReady
			}
		}
		return
	}

	// 检查用户权限，播放
	if !s.checkPermission(auth.PullRight) {
		resp.StatusCode = StatusForbidden
		return
	}

	if s.transport.Type == RTPMulticast { // 需要修改回复的transport
		st := media.GetOrCreate(s.path)
		if st == nil { // 没有找到源
			resp.StatusCode = StatusNotFound
			return
		}
		ma := st.Multicastable()
		if ma == nil { // 不支持组播
			resp.StatusCode = StatusUnsupportedTransport
			return
		}

		ts = fmt.Sprintf("%s;destination=%s;port=%d-%d;source=%s;ttl=%d",
			ts, ma.MulticastIP(),
			ma.Port(chindex), ma.Port(chindex+1),
			ma.SourceIP(), ma.TTL())
		resp.Header.Set(FieldTransport, ts)
	}

	if s.status < statusReady { // 初始状态切换到Ready
		s.status = statusReady
	}
}

func (s *Session) onRecord(resp *Response, req *Request) {
	if s.status == statusRecording {
		return
	}

	// 传输模式、会话模式判断
	if s.mode != RecordSession || s.transport.Type != RTPTCPUnicast {
		resp.StatusCode = StatusMethodNotValidInThisState
		return
	}

	if !s.checkPermission(auth.PushRight) {
		resp.StatusCode = StatusForbidden
		return
	}

	s.asTCPPusher()
	s.status = statusRecording
}

func (s *Session) onPlay(resp *Response, req *Request) (err error) {
	if s.status == statusPlaying {
		return
	}

	// 传输模式、会话模式判断
	if s.mode != PlaySession || s.transport.Type == RTPUnknownTrans {
		resp.StatusCode = StatusMethodNotValidInThisState
		return s.response(resp)
	}

	stream := media.GetOrCreate( s.path)
	if stream == nil {
		resp.StatusCode = StatusNotFound
		return s.response(resp)
	}

	if !s.checkPermission(auth.PullRight) {
		resp.StatusCode = StatusForbidden
		return s.response(resp)
	}

	resp.Header.Set(FieldRange, req.Header.Get(FieldRange))
	switch s.transport.Type {
	case RTPTCPUnicast:
		err = s.asTCPConsumer(stream, resp)
	case RTPUDPUnicast:
		err = s.asUDPConsumer(stream, resp)
	default:
		err = s.asMulticastConsumer(stream, resp)
	}

	if err == nil {
		s.status = statusPlaying
	}
	return
}

func (s *Session) checkPermission(right auth.AccessRight) bool {
	if s.authMode == auth.NoneAuth {
		return true
	}

	if s.user == nil {
		return false
	}

	return s.user.ValidatePermission(s.path, right)
}

func (s *Session) checkAuth(r *Request) (user *auth.User, err error) {
	switch s.authMode {
	case auth.BasicAuth:
		username, password, has := r.BasicAuth()
		if !has {
			return nil, errors.New("require legal Authorization field")
		}
		user := auth.Get(username)
		if user == nil {
			return nil, errors.New("user not exist")
		}
		err = user.ValidatePassword(password)
		if err != nil {
			return nil, err
		}
		return user, nil

	case auth.DigestAuth:
		username, response, has := r.DigestAuth()
		if !has {
			return nil, errors.New("require legal Authorization field")
		}
		user := auth.Get(username)
		if user == nil {
			return nil, errors.New("user not exist")
		}
		resp2 := formatDigestAuthResponse(realm, s.nonce, r.Method, r.URL.String(), username, user.Password)
		if resp2 == response {
			return user, nil
		}
		resp2 = formatDigestAuthResponse(realm, s.nonce, r.Method, r.URL.String(), username, user.PasswordMD5())
		if resp2 == response {
			return user, nil
		}
		s.nonce = security.NewID().MD5()
		return nil, errors.New("require legal Authorization field")
	default: // 无需验证
		return nil, nil
	}
}

func (s *Session) onPreprocess(resp *Response, req *Request) (continueProcess bool, err error) {
	// Options 方法无需验证，直接回复
	if req.Method == MethodOptions {
		resp.Header.Set(FieldPublic, "DESCRIBE, SETUP, TEARDOWN, PLAY, OPTIONS, ANNOUNCE, RECORD")
		err = s.response(resp)
		return false, err
	}

	// 关闭请求
	if req.Method == MethodTeardown {
		// 发送响应
		err = s.response(resp)
		s.Close()
		return false, err
	}

	// 检查状态下的方法
	switch s.status {
	case statusReady:
		continueProcess = req.Method == MethodSetup ||
			req.Method == MethodPlay || req.Method == MethodRecord
	case statusPlaying:
		continueProcess = req.Method == MethodPlay
	case statusRecording:
		continueProcess = req.Method == MethodRecord
	default:
		continueProcess = !(req.Method == MethodPlay || req.Method == MethodRecord)
	}
	if !continueProcess {
		resp.StatusCode = StatusMethodNotValidInThisState
		err = s.response(resp)
		return false, err
	}

	// 检查认证
	user, err2 := s.checkAuth(req)
	if err2 != nil {
		resp.StatusCode = StatusUnauthorized
		if err2 != nil {
			resp.Status = err2.Error()
		}
		err = s.response(resp)
		return false, err
	}

	s.user = user
	return true, nil
}

func (s *Session) response(resp *Response) error {
	s.lockW.Lock()

	var err error

	if s.wsconn != nil { // websocket 客户端
		buf := buffers.Get().(*bytes.Buffer)
		buf.Reset()
		defer buffers.Put(buf)

		err = resp.Write(buf) // 保证写入包的完整性，简化前端分包
		_, err = s.wsconn.Write(buf.Bytes())
	} else {
		err = resp.Write(s.conn)
		if err == nil {
			_, err = s.conn.Flush()
		}
	}

	s.lockW.Unlock()

	if err != nil {
		s.logger.Errorf("send response error = %v", err)
		return err
	}

	if s.logger.LevelEnabled(xlog.DebugLevel) {
		s.logger.Debugf("===>>>\r\n%s", strings.TrimSpace(resp.String()))
	}

	return nil
}

func (s *Session) newResponse(code int, req *Request) *Response {
	resp := &Response{
		StatusCode: code,
		Header:     make(Header),
		Request:    req,
	}

	resp.Header.Set(FieldCSeq, req.Header.Get(FieldCSeq))
	resp.Header.Set(FieldSession, s.lsession)

	// 根据认证模式增加认证所需的字段
	switch s.authMode {
	case auth.BasicAuth:
		resp.SetBasicAuth(realm)
	case auth.DigestAuth:
		resp.SetDigestAuth(realm, s.nonce)
	}
	return resp
}

func (s *Session) parseSdp(rawSdp string) (err error) {
	// 从流中取 sdp
	s.rawSdp = rawSdp
	// 解析
	s.sdp, err = sdp.ParseString(s.rawSdp)
	if err != nil {
		return
	}

	for _, media := range s.sdp.Media {
		switch media.Type {
		case "video":
			s.vControl = media.Attributes.Get("control")
			s.vCodec = media.Format[0].Name
		case "audio":
			s.aControl = media.Attributes.Get("control")
			s.aCodec = media.Format[0].Name
		}
	}
	return
}

func getControlPath(ctrl string) (path string, err error) {
	if len(ctrl) >= len(rtspURLPrefix) && strings.EqualFold(ctrl[:len(rtspURLPrefix)], rtspURLPrefix) {
		var ctrlURL *url.URL
		ctrlURL, err = url.Parse(ctrl)
		if err != nil {
			return "", err
		}
		if ctrlURL.Port() == "" {
			ctrlURL.Host = fmt.Sprintf("%s:554", ctrlURL.Hostname())
		}
		return ctrlURL.String(), nil
	}
	return ctrl, nil
}
