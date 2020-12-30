// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"net/http"
	"runtime/debug"

	"github.com/cnotch/ipchub/av/format/flv"
	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/ipchub/stats"
	"github.com/cnotch/xlog"
)

type httpFlvConsumer struct {
	logger  *xlog.Logger
	addr    string
	w       *flv.Writer
	closeCh chan bool
	closed  bool
}

func (c *httpFlvConsumer) Consume(pack Pack) {
	if c.closed {
		return
	}

	err := c.w.WriteTag(pack.(*flv.Tag))

	if err != nil {
		c.logger.Errorf("http-flv: send tag failed; %v", err)
		c.Close()
		return
	}
}

func (c *httpFlvConsumer) Close() (err error) {
	if c.closed {
		return
	}

	c.closed = true
	close(c.closeCh)
	return nil
}

// ConsumeByHTTP 处理 http 方式访问流媒体
func ConsumeByHTTP(logger *xlog.Logger, path string, addr string, w http.ResponseWriter) {
	logger = logger.With(xlog.Fields(
		xlog.F("path", path),
		xlog.F("addr", addr)))

	stream := media.GetOrCreate(path)
	if stream == nil {
		http.Error(w, "404 page not found", http.StatusNotFound)
		logger.Errorf("http-flv: no stream found")
		return
	}

	typeFlags := stream.FlvTypeFlags()
	if typeFlags == 0 {
		http.Error(w, "404 page not found", http.StatusNotFound)
		logger.Errorf("http-flv: stream not support flv")
		return
	}

	var cid media.CID
	defer func() {
		if r := recover(); r != nil {
			xlog.Errorf("http-flv: panic; %v \n %s", r, debug.Stack())
		}
		stream.StopConsume(cid)
		stats.FlvConns.Release()
		logger.Info("http-flv: stop http-flv consume")
	}()

	logger.Info("http-flv: start http-flv consume")
	stats.FlvConns.Add()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "video/x-flv")

	// 启动 pack 消费,必须 StartConsume 前写入 Header
	flvWriter, err := flv.NewWriter(w, typeFlags)
	if err != nil {
		http.Error(w, "send flv header failed", http.StatusInternalServerError)
		logger.Error("http-flv: send flv header failed.")
		return
	}

	c := &httpFlvConsumer{
		logger:  logger,
		w:       flvWriter,
		closeCh: make(chan bool),
	}

	cid = stream.StartConsume(c, media.FLVPacket, "net=http-flv,"+addr)

	// 等待关闭
	select {
	case <-c.closeCh:
	}
}
