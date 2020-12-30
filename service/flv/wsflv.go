// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"runtime/debug"
	"time"

	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/ipchub/network/websocket"
	"github.com/cnotch/ipchub/av/format/flv"
	"github.com/cnotch/ipchub/stats"
	"github.com/cnotch/xlog"
)

type wsFlvConsumer struct {
	logger *xlog.Logger
	w      *flv.Writer
	conn   websocket.Conn
	closed bool
}

func (c *wsFlvConsumer) Consume(pack Pack) {
	if c.closed {
		return
	}

	err := c.w.WriteTag(pack.(*flv.Tag))

	if err != nil {
		c.logger.Errorf("ws-flv: send tag failed; %v", err)
		c.Close()
		return
	}
}

func (c *wsFlvConsumer) Close() (err error) {
	if c.closed {
		return
	}

	c.closed = true
	c.conn.Close()
	return nil
}

func (c *wsFlvConsumer) Type() string {
	return "websocket-flv"
}

// ConsumeByWebsocket 处理 websocket 方式访问流媒体
func ConsumeByWebsocket(logger *xlog.Logger, path string, addr string, conn websocket.Conn) {
	logger = logger.With(xlog.Fields(
		xlog.F("path", path),
		xlog.F("addr", addr)))

	stream := media.GetOrCreate(path)
	if stream == nil {
		conn.Close()
		logger.Errorf("ws-flv: no stream found")
		return
	}

	typeFlags := stream.FlvTypeFlags()
	if typeFlags == 0 {
		conn.Close()
		logger.Errorf("ws-flv: stream not support flv")
		return
	}

	var cid media.CID

	defer func() {
		if r := recover(); r != nil {
			xlog.Errorf("ws-flv: panic; %v \n %s", r, debug.Stack())
		}
		stream.StopConsume(cid)
		conn.Close()
		stats.FlvConns.Release()
		logger.Info("stop websocket-flv consume")
	}()

	logger.Info("start websocket-flv consume")
	stats.FlvConns.Add()

	// 启动 pack 消费,必须 StartConsume 前写入 Header
	flvWriter, err := flv.NewWriter(conn, typeFlags)
	if err != nil {
		logger.Error("ws-flv: send flv header failed.")
		return
	}

	c := &wsFlvConsumer{
		logger: logger,
		conn:   conn,
		w:      flvWriter,
	}

	cid = stream.StartConsume(c, media.FLVPacket, "net=websocket-flv,"+addr)

	for !c.closed {
		deadLine := time.Time{}

		if err := conn.SetReadDeadline(deadLine); err != nil {
			break
		}
		var temp [1]byte
		if _, err := conn.Read(temp[:]); err != nil {
			if !c.closed {
				logger.Errorf("websocket error; %v.", err)
			}
			break
		}
	}
}
