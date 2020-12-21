// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"net"
	"sync"

	"github.com/cnotch/ipchub/network/socket/listener"
	"github.com/kelindar/tcp"
	"github.com/cnotch/xlog"
)

// MatchRTSP 仅匹配 RTSP 请求方法
// 注意：由于RTSP 和 HTTP 都有 OPTIONS 方法，因此对 RTSP 的 OPTIONS 做了进一步细化
func MatchRTSP() listener.Matcher {
	return listener.MatchPrefix("OPTIONS * RTSP", "OPTIONS * rtsp",
		"OPTIONS rtsp://", "OPTIONS RTSP://",
		MethodDescribe, MethodAnnounce, MethodSetup,
		MethodPlay, MethodPause, MethodTeardown,
		MethodGetParameter, MethodSetParameter,
		MethodRecord, MethodRedirect)
}

// Server rtsp 服务器
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
	s := newSession(svr, c)
	go s.process()
}
