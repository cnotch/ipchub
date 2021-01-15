// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/hevc"
)

type h265Packetizer struct {
	meta      *codec.VideoMeta
	tagWriter TagWriter
	spsMuxed  bool
	nextDts   float64
	dtsStep   float64
}

func NewH265Packetizer(meta *codec.VideoMeta, tagWriter TagWriter) Packetizer {
	h265p := &h265Packetizer{
		meta:      meta,
		tagWriter: tagWriter,
	}

	if meta.FrameRate > 0 {
		h265p.dtsStep = 1000.0 / meta.FrameRate
	}
	return h265p
}

func (h265p *h265Packetizer) PacketizeSequenceHeader() error {
	if h265p.spsMuxed {
		return nil
	}

	if !hevc.MetadataIsReady(h265p.meta) {
		// not enough
		return nil
	}

	h265p.spsMuxed = true

	if h265p.meta.FixedFrameRate {
		h265p.dtsStep = 1000.0 / h265p.meta.FrameRate
	} else { // TODO:
		h265p.dtsStep = 1000.0 / 30
	}

	record := NewHEVCDecoderConfigurationRecord(h265p.meta.Vps, h265p.meta.Sps, h265p.meta.Pps)
	body, _ := record.Marshal()

	videoData := &VideoData{
		FrameType:       FrameTypeKeyFrame,
		CodecID:         CodecIDHEVC,
		H2645PacketType: H2645PacketTypeSequenceHeader,
		CompositionTime: 0,
		Body:            body,
	}
	data, _ := videoData.Marshal()

	tag := &Tag{
		TagType:   TagTypeVideo,
		DataSize:  uint32(len(data)),
		Timestamp: 0,
		StreamID:  0,
		Data:      data,
	}

	return h265p.tagWriter.WriteFlvTag(tag)
}

func (h265p *h265Packetizer) Packetize(basePts int64, frame *codec.Frame) error {
	nalType := (frame.Payload[0] >> 1) & 0x3f
	if nalType == hevc.NalVps ||
		nalType == hevc.NalSps ||
		nalType == hevc.NalPps {
		return h265p.PacketizeSequenceHeader()
	}

	dts := int64(h265p.nextDts)
	h265p.nextDts += h265p.dtsStep
	pts := frame.AbsTimestamp - basePts + ptsDelay
	if dts > pts {
		pts = dts
	}

	videoData := &VideoData{
		FrameType:       FrameTypeInterFrame,
		CodecID:         CodecIDHEVC,
		H2645PacketType: H2645PacketTypeNALU,
		CompositionTime: uint32(pts - dts),
		Body:            frame.Payload,
	}

	if nalType >= hevc.NalBlaWLp && nalType <= hevc.NalCraNut {
		videoData.FrameType = FrameTypeKeyFrame
	}
	data, _ := videoData.Marshal()

	tag := &Tag{
		TagType:   TagTypeVideo,
		DataSize:  uint32(len(data)),
		Timestamp: uint32(dts),
		StreamID:  0,
		Data:      data,
	}

	return h265p.tagWriter.WriteFlvTag(tag)
}
