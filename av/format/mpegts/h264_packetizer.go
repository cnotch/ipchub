// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mpegts

import (
	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/h264"
)

type h264Packetizer struct {
	meta          *codec.VideoMeta
	tsframeWriter FrameWriter
	metaReady     bool
	nextDts       float64
	dtsStep       float64
}

func NewH264Packetizer(meta *codec.VideoMeta, tsframeWriter FrameWriter) Packetizer {
	h264p := &h264Packetizer{
		meta:          meta,
		tsframeWriter: tsframeWriter,
	}

	h264p.prepareMetadata()
	
	return h264p
}

func (h264p *h264Packetizer) prepareMetadata() error {
	if h264p.metaReady {
		return nil
	}

	if !h264.MetadataIsReady(h264p.meta) {
		// not enough
		return nil
	}

	if h264p.meta.FixedFrameRate {
		h264p.dtsStep = 1000.0 / h264p.meta.FrameRate
	} else { // TODO:
		h264p.dtsStep = 1000.0 / 30
	}
	h264p.metaReady = true

	return nil
}

func (h264p *h264Packetizer) Packetize(basePts int64, frame *codec.Frame) error {
	if frame.Payload[0]&0x1F == h264.NalSps {
		return h264p.prepareMetadata()
	}

	if frame.Payload[0]&0x1F == h264.NalPps {
		return h264p.prepareMetadata()
	}

	dts := int64(h264p.nextDts)
	h264p.nextDts += h264p.dtsStep
	pts := frame.AbsTimestamp - basePts + ptsDelay
	if dts > pts {
		pts = dts
	}

	// set fields
	tsframe := &Frame{
		Pid:      tsVideoPid,
		StreamID: tsVideoAvc,
		Dts:      dts * 90,
		Pts:      pts * 90,
		Payload:  frame.Payload,
		key:      frame.Payload[0]&0x1F == h264.NalIdrSlice,
	}

	tsframe.prepareAvcHeader(h264p.meta.Sps, h264p.meta.Pps)

	return h264p.tsframeWriter.WriteMpegtsFrame(tsframe)
}
