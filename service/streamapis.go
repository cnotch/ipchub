// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package service

import (
	"net/http"
	"path"

	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/network/websocket"
	"github.com/cnotch/ipchub/provider/auth"
	"github.com/cnotch/ipchub/service/flv"
	"github.com/cnotch/ipchub/service/hls"
	"github.com/cnotch/xlog"

	"github.com/cnotch/apirouter"
	"github.com/cnotch/ipchub/utils/scan"
)

// 初始化流式访问
func (s *Service) initHTTPStreams(mux *http.ServeMux) {
	mux.Handle("/ws/", apirouter.WrapHandler(http.HandlerFunc(s.onWebSocketRequest), apirouter.PreInterceptor(s.streamInterceptor)))
	mux.Handle("/streams/", apirouter.WrapHandler(http.HandlerFunc(s.onStreamsRequest), apirouter.PreInterceptor(s.streamInterceptor)))
}

// websocket 请求处理
func (s *Service) onWebSocketRequest(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get(usernameHeaderKey)
	streamPath, ext := extractStreamPathAndExt(r.URL.Path)
	_ = ext

	if ws, ok := websocket.TryUpgrade(w, r, streamPath, username); ok {

		if ws.Subprotocol() == "rtsp" { // rtsp 直连
			// rtsp接入
			s.rtsp.OnAccept(ws)
			return
		}

		if ext == ".flv" {
			go flv.ConsumeByWebsocket(s.logger, streamPath, r.RemoteAddr, ws)
			return
		}

		s.logger.Warnf("websocket sub-protocol is not supported: %s.", ws.Subprotocol())
		ws.Close()
	}
}

// streams 请求处理(flv,mu38,ts)
func (s *Service) onStreamsRequest(w http.ResponseWriter, r *http.Request) {
	// 获取文件后缀和流路径
	streamPath, ext := extractStreamPathAndExt(r.URL.Path)
	s.logger.Info("http access stream media.",
		xlog.F("path", streamPath),
		xlog.F("ext", ext))

	w.Header().Set("Access-Control-Allow-Origin", "*")
	switch ext {
	case ".flv":
		flv.ConsumeByHTTP(s.logger, streamPath, r.RemoteAddr, w)
	case ".m3u8":
		token := r.URL.Query().Get("token")
		hls.GetM3u8(s.logger, streamPath, token, r.RemoteAddr, w)
	case ".ts":
		hls.GetTS(s.logger, streamPath, r.RemoteAddr, w)
	default:
		s.logger.Warnf("request file ext is not supported: %s.", ext)
		http.NotFound(w, r)
	}
}

func (s *Service) streamInterceptor(w http.ResponseWriter, r *http.Request) bool {
	if path.Base(r.URL.Path) == "crossdomain.xml" {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(crossdomainxml)
		return false
	}

	if !config.Auth() {
		// 不启用媒体流访问验证
		return true
	}

	if s.authInterceptor(w, r) {
		return permissionInterceptor(w, r)
	}

	return false
}

// 验证用户是否有权限播放指定的流
func permissionInterceptor(w http.ResponseWriter, r *http.Request) bool {
	userName := r.Header.Get(usernameHeaderKey)
	u := auth.Get(userName)

	streamPath, _ := extractStreamPathAndExt(r.URL.Path)

	if u == nil || !u.ValidatePermission(streamPath, auth.PullRight) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return false
	}

	return true
}

// 提取请求路径中的流path和格式后缀
func extractStreamPathAndExt(requestPath string) (streamPath, ext string) {
	ext = path.Ext(requestPath)
	_, substr, _ := scan.NewScanner('/', nil).Scan(requestPath[1:])
	streamPath = requestPath[1+len(substr) : len(requestPath)-len(ext)]
	return
}
