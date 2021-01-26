// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package wsp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/ipchub/network/websocket"
	"github.com/cnotch/ipchub/provider/security"
	"github.com/cnotch/ipchub/service/rtsp"
	"github.com/cnotch/ipchub/stats"
	"github.com/cnotch/xlog"
	"github.com/pixelbender/go-sdp/sdp"
)

const (
	statusInit = iota
	statusReady
	statusPlaying
)
const (
	rtspURLPrefix = "rtsp://" // RTSP地址前缀
)

// Session RTSP 会话
type Session struct {
	// 创建时设置
	svr         *Server
	channelID   string
	logger      *xlog.Logger
	closed      bool
	paused      bool
	lsession    string // 本地会话标识
	timeout     time.Duration
	conn        websocket.Conn
	lockW       sync.Mutex
	dataChannel websocket.Conn

	// DESCRIBE 后设置
	url      *url.URL
	path     string
	rawSdp   string
	sdp      *sdp.Session
	aControl string
	vControl string
	aCodec   string
	vCodec   string

	// Setup 后设置
	transport rtsp.RTPTransport

	// 启动流媒体传输后设置
	status int // session状态
	source *media.Stream
	cid    *media.CID
}

func newSession(svr *Server, conn websocket.Conn, channelID string) *Session {

	session := &Session{
		svr:       svr,
		channelID: channelID,
		lsession:  security.NewID().Base64(),
		timeout:   config.NetTimeout() * time.Duration(2),
		conn:      conn,
		transport: rtsp.RTPTransport{
			Mode: rtsp.PlaySession, // 默认为播放
			Type: rtsp.RTPUnknownTrans,
		},
		status: statusInit,
	}

	for i := 0; i < 4; i++ {
		session.transport.Channels[i] = -1
		session.transport.ClientPorts[i] = -1
	}
	session.logger = svr.logger.With(xlog.Fields(
		xlog.F("channel", channelID),
		xlog.F("path", conn.Path())))

	return session
}

// 设置rtp数据通道
func (s *Session) setDataChannel(dc websocket.Conn) {
	s.lockW.Lock()
	s.dataChannel = dc
	s.lockW.Unlock()
}

// Addr Session地址
func (s *Session) Addr() string {
	return s.conn.RemoteAddr().String()
}

// Consume 消费媒体包
func (s *Session) Consume(p Pack) {
	if s.closed || s.paused {
		return
	}

	buf := buffers.Get().(*bytes.Buffer)
	buf.Reset()
	defer buffers.Put(buf)
	p2 := p.(*rtsp.RTPPack)
	p2.Write(buf, s.transport.Channels[:])

	var err error
	s.lockW.Lock()
	if s.dataChannel != nil {
		_, err = s.dataChannel.Write(buf.Bytes())
	}
	s.lockW.Unlock()

	if err != nil {
		s.logger.Errorf("send pack error = %v , close socket", err)
		s.Close()
		return
	}
}

// Close 关闭会话
func (s *Session) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true
	s.paused = false
	s.conn.Close()
	s.lockW.Lock()
	if s.dataChannel != nil {
		s.dataChannel.Close()
	}
	s.lockW.Unlock()
	return nil
}

func (s *Session) process() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			s.logger.Errorf("wsp channel panic, %v \n %s", r, debug.Stack())
		}

		if err != nil {
			if err == io.EOF { // 如果客户端断开提醒
				s.logger.Warn("websocket disconnect actively")
			} else if !s.closed { // 如果主动关闭，不提示
				s.logger.Error(err.Error())
			}
		}

		// 删除通道
		s.svr.sessions.Delete(s.channelID)
		// 停止消费
		if s.cid != nil {
			s.source.StopConsume(*s.cid)
			s.cid = nil
			s.source = nil
		}
		// 关闭连接
		s.Close()

		// 重置到初始状态
		s.conn = nil
		s.dataChannel = nil
		s.status = statusInit
		stats.WspConns.Release()
		s.logger.Info("close wsp channel")
	}()

	s.logger.Info("open wsp channel")

	stats.WspConns.Add() // 增加一个 RTSP 连接计数

	for !s.closed {
		deadLine := time.Time{}
		if s.timeout > 0 {
			deadLine = time.Now().Add(s.timeout)
		}
		if err = s.conn.SetReadDeadline(deadLine); err != nil {
			break
		}

		var req *Request
		req, err = DecodeRequest(s.conn, s.logger)
		if err != nil {
			break
		}

		if req.Cmd != CmdWrap {
			s.logger.Error("must is WRAP command request")
			break
		}

		// 从包装命令中提取 rtsp 请求
		var rtspReq *rtsp.Request
		rtspReq, err = rtsp.ReadRequest(bufio.NewReader(bytes.NewBufferString(req.Body)))
		if err != nil {
			break
		}

		// 处理请求，并获得响应
		rtspResp := s.onRequest(rtspReq)

		// 发送响应
		buf := buffers.Get().(*bytes.Buffer)
		buf.Reset()
		defer buffers.Put(buf)
		req.ResponseOK(buf, map[string]string{FieldChannel: s.channelID}, "")
		rtspResp.Write(buf)
		_, err = s.conn.Write(buf.Bytes())
		if err != nil {
			break
		}

		s.logger.Debugf("wsp ===>>>\r\n%s", buf.String())

		// 关闭通道
		if rtspReq.Method == rtsp.MethodTeardown {
			break
		}
	}
}

func (s *Session) onRequest(req *rtsp.Request) *rtsp.Response {
	resp := s.newResponse(rtsp.StatusOK, req)
	// 预处理
	continueProcess := s.onPreprocess(resp, req)
	if !continueProcess {
		return resp
	}

	switch req.Method {
	case rtsp.MethodDescribe:
		s.onDescribe(resp, req)
	case rtsp.MethodSetup:
		s.onSetup(resp, req)
	case rtsp.MethodPlay:
		s.onPlay(resp, req)
	case rtsp.MethodPause:
		s.onPause(resp, req)
	default:
		// 状态不支持的方法
		resp.StatusCode = rtsp.StatusMethodNotValidInThisState
	}
	return resp
}

func (s *Session) onDescribe(resp *rtsp.Response, req *rtsp.Request) {

	// TODO: 检查 accept 中的类型是否包含 sdp
	s.url = req.URL
	s.path = s.conn.Path() // 使用websocket路径
	// s.path = utils.CanonicalPath(req.URL.Path)
	stream := media.GetOrCreate(s.path)
	if stream == nil {
		resp.StatusCode = rtsp.StatusNotFound
		return
	}

	// 从流中取 sdp
	sdpRaw := stream.Sdp()
	if len(sdpRaw) == 0 {
		resp.StatusCode = rtsp.StatusNotFound
		return
	}
	err := s.parseSdp(sdpRaw)
	if err != nil { // TODO：需要更好的处理方式
		resp.StatusCode = rtsp.StatusNotFound
		return
	}

	resp.Header.Set(rtsp.FieldContentType, "application/sdp")
	resp.Body = s.rawSdp
}

func (s *Session) onSetup(resp *rtsp.Response, req *rtsp.Request) {
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
	vPath := getControlPath(s.vControl)
	if vPath == "" {
		resp.StatusCode = rtsp.StatusInternalServerError
		resp.Status = "Invalid VControl"
		return
	}

	aPath := getControlPath(s.aControl)

	ts := req.Header.Get(rtsp.FieldTransport)
	resp.Header.Set(rtsp.FieldTransport, ts) // 先回写transport

	// 检查控制路径
	chindex := -1
	if setupPath == aPath || (aPath != "" && strings.LastIndex(setupPath, aPath) == len(setupPath)-len(aPath)) {
		chindex = int(rtsp.ChannelAudio)
	} else if setupPath == vPath || (vPath != "" && strings.LastIndex(setupPath, vPath) == len(setupPath)-len(vPath)) {
		chindex = int(rtsp.ChannelVideo)
	} else { // 找不到被 Setup 的资源
		resp.StatusCode = rtsp.StatusInternalServerError
		resp.Status = fmt.Sprintf("SETUP Unkown control:%s", setupPath)
		return
	}

	err := s.transport.ParseTransport(chindex, ts)
	if err != nil {
		resp.StatusCode = rtsp.StatusInvalidParameter
		resp.Status = err.Error()
		return
	}

	// 检查必须是play模式
	if rtsp.PlaySession != s.transport.Mode {
		resp.StatusCode = rtsp.StatusInvalidParameter
		resp.Status = "can't setup as record"
		return
	}

	if s.transport.Type != rtsp.RTPTCPUnicast { // 需要修改回复的transport
		resp.StatusCode = rtsp.StatusUnsupportedTransport
		resp.Status = "websocket only support tcp unicast"
		return
	}

	if s.status < statusReady { // 初始状态切换到Ready
		s.status = statusReady
	}
}

func (s *Session) onPlay(resp *rtsp.Response, req *rtsp.Request) {
	if s.status == statusPlaying {
		s.paused = false
		return
	}

	stream := media.GetOrCreate(s.path)
	if stream == nil {
		resp.StatusCode = rtsp.StatusNotFound
		return
	}

	resp.Header.Set(rtsp.FieldRange, req.Header.Get(rtsp.FieldRange))
	if s.cid == nil {
		s.source = stream
		cid := stream.StartConsume(s, media.RTPPacket, "wsp")
		// cid := stream.StartConsumeNoGopCache(s, media.RTPPacket, "wsp")
		s.cid = &cid
	}
	s.status = statusPlaying
	s.paused = false
	return
}

func (s *Session) onPause(resp *rtsp.Response, req *rtsp.Request) {
	if s.status == statusPlaying {
		s.paused = true
	}
}

func (s *Session) onPreprocess(resp *rtsp.Response, req *rtsp.Request) (continueProcess bool) {
	// Options
	if req.Method == rtsp.MethodOptions {
		resp.Header.Set(rtsp.FieldPublic, "DESCRIBE, SETUP, TEARDOWN, PLAY, OPTIONS, ANNOUNCE")
		return false
	}

	// 关闭请求
	if req.Method == rtsp.MethodTeardown {
		return false
	}

	// 检查状态下的方法
	switch s.status {
	case statusReady:
		continueProcess = req.Method == rtsp.MethodSetup ||
			req.Method == rtsp.MethodPlay
	case statusPlaying:
		continueProcess = (req.Method == rtsp.MethodPlay ||
			req.Method == rtsp.MethodPause)
	default:
		continueProcess = !(req.Method == rtsp.MethodPlay ||
			req.Method == rtsp.MethodRecord)
	}

	if !continueProcess {
		resp.StatusCode = rtsp.StatusMethodNotValidInThisState
		return false
	}

	return true
}

func (s *Session) newResponse(code int, req *rtsp.Request) *rtsp.Response {
	resp := &rtsp.Response{
		StatusCode: code,
		Header:     make(rtsp.Header),
		Request:    req,
	}

	resp.Header.Set(rtsp.FieldCSeq, req.Header.Get(rtsp.FieldCSeq))
	resp.Header.Set(rtsp.FieldSession, s.lsession)
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

func getControlPath(ctrl string) (path string) {
	if len(ctrl) >= len(rtspURLPrefix) && strings.EqualFold(ctrl[:len(rtspURLPrefix)], rtspURLPrefix) {
		ctrlURL, err := url.Parse(ctrl)
		if err != nil {
			return
		}
		if ctrlURL.Port() == "" {
			ctrlURL.Host = fmt.Sprintf("%s:554", ctrlURL.Hostname())
		}
		return ctrlURL.String()
	}
	return ctrl
}
