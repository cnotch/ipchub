// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"io"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/format"
	"github.com/cnotch/ipchub/av/format/rtp"
	"github.com/cnotch/queue"
)

// Pack .
type Pack = format.Packet

type packCache interface {
	CachePack(pack Pack) (keyframe bool) // 返回是否是关键帧
	PushTo(q *queue.SyncQueue) int
	Reset()
}

var _ packCache = emptyCache{}

type emptyCache struct {
}

func (emptyCache) CachePack(Pack) bool           { return false }
func (emptyCache) PushTo(q *queue.SyncQueue) int { return 0 }
func (emptyCache) Reset()                        {}

type flvMuxer interface {
	TypeFlags() byte
	codec.FrameWriter
	io.Closer
}

var _ flvMuxer = emptyFlvMuxer{}

type emptyFlvMuxer struct{}

func (emptyFlvMuxer) TypeFlags() byte                     { return 0 }
func (emptyFlvMuxer) WriteFrame(frame *codec.Frame) error { return nil }
func (emptyFlvMuxer) Close() error                        { return nil }

type rtpDemuxer interface {
	rtp.PacketWriter
	io.Closer
}

var _ rtpDemuxer = emptyRtpDemuxer{}

type emptyRtpDemuxer struct{}

func (emptyRtpDemuxer) TypeFlags() byte                  { return 0 }
func (emptyRtpDemuxer) WriteRtpPacket(*rtp.Packet) error { return nil }
func (emptyRtpDemuxer) Close() error                     { return nil }
