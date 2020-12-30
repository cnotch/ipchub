// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/ipchub/network/socket/buffered"
	"github.com/cnotch/ipchub/stats"
	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/xlog"
	"github.com/pixelbender/go-sdp/sdp"
)

const (
	defaultUserAgent = config.Name + "-rstp-client/1.0"
)

// PullClient 负责拉流到服务器
type PullClient struct {
	// 打开前设置
	closed      bool
	url         *url.URL
	userName    string
	password    string
	md5password string
	path        string
	rtpChannels [rtpChannelCount]int
	logger      *xlog.Logger

	// 添加到流媒体中心后设置
	stream *media.Stream

	// 打开连接后设置
	conn     *buffered.Conn
	lockW    sync.Mutex
	realm    string
	nonce    string
	rsession string
	seq      int64

	rawSdp   string
	sdp      *sdp.Session
	aControl string
	vControl string
	aCodec   string
	vCodec   string
}

// NewPullClient 创建拉流客户端
func NewPullClient(localPath, remoteURL string) (*PullClient, error) {
	// 检查远端路径
	url, err := url.Parse(remoteURL)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(url.Scheme) != "rtsp" {
		return nil, fmt.Errorf("RemoteURL '%s' is not RTSP url", remoteURL)
	}
	if strings.ToLower(url.Hostname()) == "" {
		return nil, fmt.Errorf("RemoteURL '%s' is not RTSP url", remoteURL)
	}
	// 如果没有 port，补上默认端口
	port := url.Port()
	if len(port) == 0 {
		url.Host = url.Hostname() + ":554"
	}

	// 提取用户名和密码
	var userName, password string
	if url.User != nil {
		userName = url.User.Username()
		password, _ = url.User.Password()
		url.User = nil
	}

	// 检查发布路径
	path := utils.CanonicalPath(localPath)

	if path == "" {
		path = utils.CanonicalPath(url.Path)
	} else {
		_, err := url.Parse("rtsp://localhost" + path)
		if err != nil {
			return nil, fmt.Errorf("Path '%s' 不合法", localPath)
		}
	}

	client := &PullClient{
		closed:   true,
		url:      url,
		userName: userName,
		password: password,
		path:     path,
	}

	for i := rtpChannelMin; i < rtpChannelCount; i++ {
		client.rtpChannels[i] = int(i)
	}

	client.logger = xlog.L().With(xlog.Fields(
		xlog.F("path", client.path),
		xlog.F("rurl", client.url.String()),
		xlog.F("type", "pull")))

	return client, nil
}

// Ping 测试网络和服务器
func (c *PullClient) Ping() error {
	if !c.closed {
		return nil
	}

	defer func() {
		c.disconnect()
		c.conn = nil
		c.stream = nil
	}()

	err := c.connect()
	if err != nil {
		return err
	}

	// OPTIONS 尝试握手
	err = c.requestHandshake()
	if err != nil {
		return err
	}

	// DESCRIBE 获取 sdp，看是否存在指定媒体
	return c.requestSDP()
}

// Open 打开拉流客户端
// 依次发生请求：OPTIONS、DESCRIBE、SETUP、PLAY
// 全部成功，启动接收 RTP流 go routine
func (c *PullClient) Open() (err error) {
	if !c.closed {
		return nil
	}

	defer func() {
		if err != nil { // 出现任何错误执行断链操作
			c.disconnect()
			c.conn = nil
			c.stream = nil
		}
	}()

	// 连接
	err = c.connect()
	if err != nil {
		return err
	}

	// 请求握手
	err = c.requestHandshake()
	if err != nil {
		return err
	}

	// 获取流信息
	err = c.requestSDP()
	if err != nil {
		return err
	}

	// 设置通讯通道
	err = c.requestSetup()
	if err != nil {
		return err
	}

	// 请求播放
	err = c.requestPlay()
	if err != nil {
		return err
	}

	return err
}

// Close 关闭客户端
func (c *PullClient) Close() error {
	c.disconnect()
	return nil
}

func (c *PullClient) requestHandshake() (err error) {
	// 使用 OPTIONS 尝试握手
	r := c.newRequest(MethodOptions, c.url)
	r.Header.Set(FieldRequire, "implicit-play")
	_, err = c.requestWithResponse(r)
	return err
}

func (c *PullClient) requestSDP() (err error) {
	// DESCRIBE 获取 sdp
	r := c.newRequest(MethodDescribe, c.url)
	r.Header.Set(FieldAccept, "application/sdp")
	resp, err := c.requestWithResponse(r)
	if err != nil {
		return err
	}

	// 解析
	c.rawSdp = resp.Body
	c.sdp, err = sdp.ParseString(c.rawSdp)
	if err != nil {
		return err
	}

	for _, media := range c.sdp.Media {
		switch media.Type {
		case "video":
			c.vControl = media.Attributes.Get("control")
			c.vCodec = media.Format[0].Name

		case "audio":
			c.aControl = media.Attributes.Get("control")
			c.aCodec = media.Format[0].Name
		}
	}
	return err
}

func (c *PullClient) requestSetup() (err error) {
	var respVS, respAS *Response
	// 视频通道设置
	if len(c.vControl) > 0 {
		var setupURL *url.URL
		setupURL, err = c.getSetupURL(c.vControl)

		r := c.newRequest(MethodSetup, setupURL)
		r.Header.Set(FieldTransport,
			fmt.Sprintf("RTP/AVP/TCP;unicast;interleaved=%d-%d", c.rtpChannels[ChannelVideo], c.rtpChannels[ChannelVideoControl]))
		respVS, err = c.requestWithResponse(r)
		if err != nil {
			return err
		}
	}

	// 音频通道设置
	if len(c.aControl) > 0 {
		var setupURL *url.URL
		setupURL, err = c.getSetupURL(c.aControl)

		r := c.newRequest(MethodSetup, setupURL)
		r.Header.Set(FieldTransport,
			fmt.Sprintf("RTP/AVP/TCP;unicast;interleaved=%d-%d", c.rtpChannels[ChannelAudio], c.rtpChannels[ChannelAudioControl]))

		respAS, err = c.requestWithResponse(r)
		if err != nil {
			return err
		}
	}
	_ = respVS
	_ = respAS
	return
}

func (c *PullClient) requestPlay() (err error) {
	r := c.newRequest(MethodPlay, c.url)

	resp, err := c.requestWithResponse(r)
	if err != nil {
		return err
	}
	_ = resp
	mproxy := &multicastProxy{
		path:        c.path,
		bufferSize:  config.NetBufferSize(),
		multicastIP: utils.Multicast.NextIP(), // 设置组播IP
		ttl:         config.MulticastTTL(),
		logger:      c.logger,
	}

	for i := rtpChannelMin; i < rtpChannelCount; i++ {
		mproxy.ports[i] = utils.Multicast.NextPort()
	}

	c.stream = media.NewStream(c.path, c.rawSdp,
		media.Attr("addr", c.url.String()),
		media.Multicast(mproxy))
	go c.playStream()

	return nil
}

func (c *PullClient) playStream() {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Errorf("pull stream panic; %v \n %s", r, debug.Stack())
		}

		stats.RtspConns.Release() // 减少RTSP连接计数
		media.Unregist(c.stream)  // 从媒体中心取消注册
		c.disconnect()            // 确保网络关闭
		c.conn = nil              // 通知GC，尽早释放资源
		c.stream = nil
		c.logger.Infof("close pull stream")
	}()

	c.logger.Infof("open pull stream")
	media.Regist(c.stream) // 向媒体中心注册流
	stats.RtspConns.Add()  // 增加一个 RTSP 连接计数

	lastHeartbeat := time.Now()
	reader := c.conn.Reader()
	heartbeatInterval := config.NetHeartbeatInterval()
	timeout := config.NetTimeout()

	for !c.closed {
		deadLine := time.Time{}
		if timeout > 0 {
			deadLine = time.Now().Add(timeout)
		}
		if err := c.conn.SetReadDeadline(deadLine); err != nil {
			c.logger.Error(err.Error())
			break
		}

		err := receive(c.logger, reader, c.rtpChannels[:], c)
		if err != nil {
			if err == io.EOF { // 如果对方断开
				c.logger.Warn("The remote RTSP server is actively disconnected.")
			} else if !c.closed { // 如果非主动关闭
				c.logger.Error(err.Error())
			}
			break
		}

		if heartbeatInterval > 0 && time.Now().Sub(lastHeartbeat) > heartbeatInterval {
			lastHeartbeat = time.Now()
			// 心跳包
			r := c.newRequest(MethodOptions, c.url)
			err := c.request(r)
			if err != nil {
				c.logger.Error(err.Error())
				break
			}
		}
	}
	reader = nil
}

func (c *PullClient) onPack(p *RTPPack) error {
	return c.stream.WriteRtpPacket(p)
}

func (c *PullClient) onRequest(r *Request) (err error) {
	// 只处理 Options 方法
	switch r.Method {
	case MethodOptions:
		resp := &Response{
			StatusCode: 200,
			Header:     r.Header,
		}
		resp.Header.Del(FieldUserAgent)
		resp.Header.Set(FieldPublic, MethodOptions)
		err = c.response(resp)
		if err != nil {
			return err
		}
	default:
		resp := &Response{
			StatusCode: StatusMethodNotAllowed,
			Header:     r.Header,
		}
		resp.Header.Del(FieldUserAgent)
		err = c.response(resp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *PullClient) onResponse(resp *Response) (err error) {
	// 忽略
	return
}

func (c *PullClient) getSetupURL(ctrl string) (setupURL *url.URL, err error) {
	if len(ctrl) >= len(rtspURLPrefix) && strings.EqualFold(ctrl[:len(rtspURLPrefix)], rtspURLPrefix) {
		return url.Parse(ctrl)
	}

	setupURL = new(url.URL)
	*setupURL = *c.url
	if setupURL.Path[len(setupURL.Path)-1] == '/' {
		setupURL.Path = setupURL.Path + ctrl
	} else {
		setupURL.Path = setupURL.Path + "/" + ctrl
	}

	return
}

func (c *PullClient) newRequest(method string, url *url.URL) *Request {
	r := &Request{
		Method: method,
		Header: make(Header),
	}

	r.URL = url
	if url == nil {
		r.URL = c.url
	}

	r.Header.Set(FieldUserAgent, defaultUserAgent)
	r.Header.Set(FieldCSeq, strconv.FormatInt(atomic.AddInt64(&c.seq, 1), 10))
	if len(c.rsession) > 0 {
		r.Header.Set(FieldSession, c.rsession)
	}

	// 和安全相关，已经收到安全作用域信息
	if len(c.realm) > 0 {
		pw := c.password
		if len(c.md5password) > 0 {
			pw = c.md5password
		}

		if len(c.nonce) > 0 {
			// Digest 认证
			r.SetDigestAuth(r.URL, c.realm, c.nonce, c.userName, pw)
		} else {
			// Basic 认证
			r.SetBasicAuth(c.userName, pw)
		}
	}

	return r
}

func (c *PullClient) receiveResponse() (resp *Response, err error) {
	resp, err = ReadResponse(c.conn.Reader())
	if err != nil {
		return nil, err
	}

	if c.logger.LevelEnabled(xlog.DebugLevel) {
		c.logger.Debugf("<<<===\r\n%s", strings.TrimSpace(resp.String()))
	}

	return
}

func (c *PullClient) requestWithResponse(r *Request) (*Response, error) {
	err := c.request(r)
	if err != nil {
		return nil, err
	}

	resp, err := c.receiveResponse()
	if err != nil {
		return nil, err
	}

	// 保存 session
	c.rsession = resp.Header.Get(FieldSession)

	// 如果需要安全信息，增加安全信息并再次请求
	if resp.StatusCode == StatusUnauthorized {

		if len(c.userName) == 0 {
			return resp, errors.New("require username and password")
		}

		pw := c.password
		auth := resp.Header.Get(FieldWWWAuthenticate)
		if len(auth) > len(digestAuthPrefix) && strings.EqualFold(auth[:len(digestAuthPrefix)], digestAuthPrefix) {
			ok := false
			c.realm, c.nonce, ok = resp.DigestAuth()
			if !ok {
				return resp, fmt.Errorf("WWW-Authenticate, %s", auth)
			}

			r.SetDigestAuth(r.URL, c.realm, c.nonce, c.userName, pw)
		} else if len(auth) > len(basicAuthPrefix) && strings.EqualFold(auth[:len(basicAuthPrefix)], basicAuthPrefix) {
			ok := false
			c.realm, ok = resp.BasicAuth()
			if !ok {
				return resp, fmt.Errorf("WWW-Authenticate, %s", auth)
			}
			r.SetBasicAuth(c.userName, pw)
		} else {
			return resp, fmt.Errorf("WWW-Authenticate, %s", auth)
		}

		// 修改请求序号
		r.Header.Set(FieldCSeq, strconv.FormatInt(atomic.AddInt64(&c.seq, 1), 10))

		err := c.request(r)
		if err != nil {
			return nil, err
		}

		resp, err = c.receiveResponse()
		if err != nil {
			return nil, err
		}

		// 保存 session
		c.rsession = resp.Header.Get(FieldSession)

		// TODO: 代码臃肿，需要优化
		// 再试一次 password md5的情况
		if resp.StatusCode == StatusUnauthorized {
			md5Digest := md5.Sum([]byte(c.password))
			c.md5password = hex.EncodeToString(md5Digest[:])

			pw := c.md5password
			auth := resp.Header.Get(FieldWWWAuthenticate)
			if len(auth) > len(digestAuthPrefix) && strings.EqualFold(auth[:len(digestAuthPrefix)], digestAuthPrefix) {
				ok := false
				c.realm, c.nonce, ok = resp.DigestAuth()
				if !ok {
					return resp, fmt.Errorf("WWW-Authenticate, %s", auth)
				}

				r.SetDigestAuth(r.URL, c.realm, c.nonce, c.userName, pw)
			} else if len(auth) > len(basicAuthPrefix) && strings.EqualFold(auth[:len(basicAuthPrefix)], basicAuthPrefix) {
				ok := false
				c.realm, ok = resp.BasicAuth()
				if !ok {
					return resp, fmt.Errorf("WWW-Authenticate, %s", auth)
				}
				r.SetBasicAuth(c.userName, pw)
			} else {
				return resp, fmt.Errorf("WWW-Authenticate, %s", auth)
			}

			// 修改请求序号
			r.Header.Set(FieldCSeq, strconv.FormatInt(atomic.AddInt64(&c.seq, 1), 10))

			err := c.request(r)
			if err != nil {
				return nil, err
			}

			resp, err = c.receiveResponse()
			if err != nil {
				return nil, err
			}

			// 保存 session
			c.rsession = resp.Header.Get(FieldSession)
		}
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 300) {
		return resp, errors.New(resp.Status)
	}

	return resp, nil
}

func (c *PullClient) request(req *Request) error {
	c.lockW.Lock()
	err := req.Write(c.conn)
	if err == nil {
		_, err = c.conn.Flush()
	}
	c.lockW.Unlock()

	if err != nil {
		c.logger.Errorf("send request error = %v", err)
		return err
	}

	if c.logger.LevelEnabled(xlog.DebugLevel) {
		c.logger.Debugf("===>>>\r\n%s", strings.TrimSpace(req.String()))
	}
	return err
}

func (c *PullClient) response(resp *Response) error {
	c.lockW.Lock()
	err := resp.Write(c.conn)
	if err == nil {
		_, err = c.conn.Flush()
	}
	c.lockW.Unlock()

	if err != nil {
		c.logger.Errorf("send response error = %v", err)
		return err
	}

	if c.logger.LevelEnabled(xlog.DebugLevel) {
		c.logger.Debugf("===>>>\r\n%s", strings.TrimSpace(resp.String()))
	}
	return nil
}

func (c *PullClient) connect() error {
	// 连接超时要更短
	timeout := time.Duration(int64(config.NetTimeout()) / 3)
	conn, err := net.DialTimeout("tcp", c.url.Host, timeout)
	if err != nil {
		c.logger.Errorf("connet remote server fail,err = %v", err)
		return err
	}

	c.closed = false // 已经连接
	c.conn = buffered.NewConn(conn,
		buffered.FlushRate(config.NetFlushRate()),
		buffered.BufferSize(config.NetBufferSize()))

	c.logger.Infof("connect remote server success")
	return nil
}

func (c *PullClient) disconnect() {
	if c.closed {
		return
	}

	c.closed = true

	c.logger.Info("disconnec from remote server")
	if c.conn != nil {
		c.conn.Close()
	}

	c.rsession = ""
	atomic.StoreInt64(&c.seq, 0)
	c.realm = ""
	c.sdp = nil
	c.aControl = ""
	c.vControl = ""
	c.aCodec = ""
	c.vCodec = ""
}
