// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/ipchub/network/socket/listener"
	"github.com/cnotch/ipchub/provider/auth"
	"github.com/cnotch/ipchub/provider/route"
	"github.com/cnotch/ipchub/service/rtsp"
	"github.com/cnotch/ipchub/service/wsp"
	"github.com/cnotch/scheduler"
	"github.com/cnotch/xlog"
	"github.com/emitter-io/address"
	"github.com/kelindar/tcp"
)

// Service 网络服务对象(服务的入口)
type Service struct {
	context  context.Context
	cancel   context.CancelFunc
	logger   *xlog.Logger
	tlsusing bool
	http     *http.Server
	rtsp     *tcp.Server
	wsp      *tcp.Server
	tokens   *auth.TokenManager
}

// NewService 创建服务
func NewService(ctx context.Context, l *xlog.Logger) (s *Service, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	s = &Service{
		context: ctx,
		cancel:  cancel,
		logger:  l,
		http:    new(http.Server),
		rtsp:    new(tcp.Server),
		wsp:     new(tcp.Server),
		tokens:  new(auth.TokenManager),
	}

	// 设置 http 的Handler
	mux := http.NewServeMux()

	// 管理员控制台
	if consoleAppDir, ok := config.ConsoleAppDir(); ok {
		mux.Handle("/", http.FileServer(http.Dir(consoleAppDir)))
	}

	// Demo应用
	if demosAppDir, ok := config.DemosAppDir(); ok {
		mux.Handle("/demos/", http.StripPrefix("/demos/", http.FileServer(http.Dir(demosAppDir))))
	}

	if config.Profile() {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	s.initApis(mux)
	s.initHTTPStreams(mux)
	s.http.Handler = mux

	// 设置 rtsp AcceptHandler
	s.rtsp.OnAccept = rtsp.CreateAcceptHandler()
	// 设置 wsp AcceptHandler
	s.wsp.OnAccept = wsp.CreateAcceptHandler()
	// 启动定时存储拉流信息
	scheduler.PeriodFunc(time.Minute*5, time.Minute*5, func() {
		route.Flush()
		auth.Flush()
		s.tokens.ExpCheck()
	}, "The task of scheduled storage of routing tables and authorization information tables(5minutes")

	s.logger.Info("service configured")
	return s, nil
}

// Listen starts the service.
func (s *Service) Listen() (err error) {
	defer s.Close()
	s.hookSignals()

	// http rtsp ws
	addr, err := address.Parse(config.Addr(), 554)
	if err != nil {
		s.logger.Panic(err.Error())
	}

	s.listen(addr, nil)

	// https wss
	tlsconf := config.GetTLSConfig()
	if tlsconf != nil {
		tls, err := tlsconf.Load()
		if err == nil {
			if tlsAddr, err := address.Parse(tlsconf.ListenAddr, 443); err == nil {
				s.listen(tlsAddr, tls)
				s.tlsusing = true
			}
		}
	}

	s.logger.Infof("service started(%s).", config.Version)
	s.logger = xlog.L()
	// Block
	select {}
}

// listen configures an main listener on a specified address.
func (s *Service) listen(addr *net.TCPAddr, conf *tls.Config) {
	// Create new listener
	s.logger.Infof("starting the listener, addr = %s.", addr.String())

	l, err := listener.New(addr.String(), conf)
	if err != nil {
		s.logger.Panic(err.Error())
	}

	// Set the read timeout on our mux listener
	timeout := time.Duration(int64(config.NetTimeout()) / 3)
	l.SetReadTimeout(timeout)

	// Set Error handler
	l.HandleError(listener.ErrorHandler(func(err error) bool {
		xlog.Warn(err.Error())
		return true
	}))

	// Configure the matchers
	l.ServeAsync(rtsp.MatchRTSP(), s.rtsp.Serve)
	l.ServeAsync(listener.MatchHTTP(), s.http.Serve)
	go l.Serve()
}

// Close closes gracefully the service.,
func (s *Service) Close() {
	if s.cancel != nil {
		s.cancel()
	}

	// 停止计划任务
	jobs := scheduler.Jobs()
	for _, job := range jobs {
		job.Cancel()
	}

	// 清空注册
	media.UnregistAll()
	// 退出前确保最新数据被存储
	route.Flush()
	auth.Flush()
}

// OnSignal starts the signal processing and makes su
func (s *Service) hookSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range c {
			s.onSignal(sig)
		}
	}()
}

// OnSignal will be called when a OS-level signal is received.
func (s *Service) onSignal(sig os.Signal) {
	switch sig {
	case syscall.SIGTERM:
		fallthrough
	case syscall.SIGINT:
		s.logger.Warn(fmt.Sprintf("received signal %s, exiting...", sig.String()))
		s.Close()
		os.Exit(0)
	}
}
