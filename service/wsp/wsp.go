// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package wsp

import (
	"bytes"
	"net"
	"sync"

	"github.com/cnotch/ipchub/network/websocket"
	"github.com/cnotch/ipchub/provider/security"
	"github.com/cnotch/xlog"
	"github.com/kelindar/tcp"
)

// Server https://github.com/Streamedian/html5_rtsp_player 客户端配套的服务器
type Server struct {
	logger *xlog.Logger

	sessions sync.Map
}

// CreateAcceptHandler 创建连接接入处理器
func CreateAcceptHandler() tcp.OnAccept {
	svr := &Server{
		logger: xlog.L(),
	}
	return svr.onAcceptConn
}

// onAcceptConn 当新连接接入时触发
func (svr *Server) onAcceptConn(c net.Conn) {
	wsc := c.(websocket.Conn)
	if wsc.Subprotocol() == "control" {
		go svr.handshakeControlChannel(wsc)
	} else {
		go svr.handshakeDataChannel(wsc)
	}
}

func (svr *Server) handshakeControlChannel(wsc websocket.Conn) {
	svr.logger.Info("wsp control channel handshake.")
	wsc = wsc.TextTransport()
	for {
		req, err := DecodeRequest(wsc, svr.logger)
		if err != nil {
			svr.logger.Error(err.Error())
			wsc.Close()
			break
		}

		if req.Cmd == CmdGetInfo {
			continue
		}

		if req.Cmd != CmdInit {
			svr.logger.Errorf("wsp control channel handshake failed, malformed WSP request command: %s.", req.Cmd)
			wsc.Close()
			break
		}

		// 初始化
		channelID := security.NewID().String()
		buf := buffers.Get().(*bytes.Buffer)
		buf.Reset()
		defer buffers.Put(buf)
		req.ResponseOK(buf, map[string]string{FieldChannel: channelID}, "")
		_, err = wsc.Write(buf.Bytes())
		if err != nil {
			svr.logger.Error(err.Error())
			wsc.Close()
			break
		}
		session := newSession(svr, wsc, channelID)
		svr.sessions.Store(channelID, session)
		svr.logger.Debugf("wsp ===>>> \r\n%s", buf.String())
		go session.process()
		break
	}
}

func (svr *Server) handshakeDataChannel(wsc websocket.Conn) {
	tc := wsc.TextTransport()
	req, err := DecodeRequest(tc, svr.logger)
	if err != nil {
		svr.logger.Error(err.Error())
		tc.Close()
		return
	}

	channelID := req.Header[FieldChannel]
	code := 200
	text := "OK"
	var session *Session
	si, ok := svr.sessions.Load(channelID)
	if ok {
		session = si.(*Session)
	} else {
		code = 404
		text = "NOT FOUND"
	}

	buf := buffers.Get().(*bytes.Buffer)
	buf.Reset()
	defer buffers.Put(buf)
	req.ResponseTo(buf, code, text, map[string]string{}, "")
	_, err = tc.Write(buf.Bytes())
	if err != nil {
		svr.logger.Error(err.Error())
		tc.Close()
		return
	}

	svr.logger.Debugf("wsp ===>>> \r\n%s", buf.String())
	if session == nil {
		tc.Close()
		return
	}

	// 添加到session
	session.setDataChannel(wsc)
}
