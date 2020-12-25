// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"io"

	"github.com/cnotch/ipchub/av"
	"github.com/cnotch/ipchub/protos/rtp"
)

type flvMuxer interface {
	TypeFlags() byte
	av.FrameWriter
	io.Closer
}

type emptyFlvMuxer struct{}

func (emptyFlvMuxer) TypeFlags() byte                  { return 0 }
func (emptyFlvMuxer) WriteFrame(frame *av.Frame) error { return nil }
func (emptyFlvMuxer) Close() error                     { return nil }

type frameConverter interface {
	rtp.PacketWriter
	io.Closer
}

type emptyFrameConverter struct{}

func (emptyFrameConverter) TypeFlags() byte               { return 0 }
func (emptyFrameConverter) WritePacket(*rtp.Packet) error { return nil }
func (emptyFrameConverter) Close() error                  { return nil }
