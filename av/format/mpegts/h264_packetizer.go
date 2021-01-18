// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mpegts

import (
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/h264"
)

type h264Packetizer struct {
	meta          *codec.VideoMeta
	tsframeWriter FrameWriter
}

func NewH264Packetizer(meta *codec.VideoMeta, tsframeWriter FrameWriter) Packetizer {
	h264p := &h264Packetizer{
		meta:          meta,
		tsframeWriter: tsframeWriter,
	}
	return h264p
}

func (h264p *h264Packetizer) Packetize(frame *codec.Frame) error {
	nalType := frame.Payload[0] & 0x1F

	dts := frame.Dts * 90000 / int64(time.Second) // 90000Hz
	pts := frame.Pts * 90000 / int64(time.Second) // 90000Hz
	// set fields
	tsframe := &Frame{
		Pid:      tsVideoPid,
		StreamID: tsVideoAvc,
		Dts:      dts,
		Pts:      pts,
		Payload:  frame.Payload,
		key:      nalType == h264.NalIdrSlice,
	}

	tsframe.prepareAvcHeader(h264p.meta.Sps, h264p.meta.Pps)

	return h264p.tsframeWriter.WriteMpegtsFrame(tsframe)
}
