// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"io"

	"github.com/cnotch/ipchub/av"
	"github.com/cnotch/ipchub/protos"
	"github.com/cnotch/ipchub/protos/rtp"
	"github.com/cnotch/queue"
)

// Pack .
type Pack = protos.Pack

type packCache interface {
	CachePack(pack Pack)
	PushTo(q *queue.SyncQueue) int
	Reset()
}

var _ packCache = emptyCache{}

type emptyCache struct {
}

func (emptyCache) CachePack(Pack)                {}
func (emptyCache) PushTo(q *queue.SyncQueue) int { return 0 }
func (emptyCache) Reset()                        {}

type flvMuxer interface {
	TypeFlags() byte
	av.FrameWriter
	io.Closer
}

var _ flvMuxer = emptyFlvMuxer{}

type emptyFlvMuxer struct{}

func (emptyFlvMuxer) TypeFlags() byte                  { return 0 }
func (emptyFlvMuxer) WriteFrame(frame *av.Frame) error { return nil }
func (emptyFlvMuxer) Close() error                     { return nil }

type frameConverter interface {
	rtp.PacketWriter
	io.Closer
}

var _ frameConverter = emptyFrameConverter{}

type emptyFrameConverter struct{}

func (emptyFrameConverter) TypeFlags() byte               { return 0 }
func (emptyFrameConverter) WritePacket(*rtp.Packet) error { return nil }
func (emptyFrameConverter) Close() error                  { return nil }
