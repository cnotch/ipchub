// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/h264"
)

type h264Packetizer struct {
	meta      *codec.VideoMeta
	tagWriter TagWriter
	spsMuxed  bool
	nextDts   float64
	dtsStep   float64
}

func NewH264Packetizer(meta *codec.VideoMeta, tagWriter TagWriter) Packetizer {
	h264p := &h264Packetizer{
		meta:      meta,
		tagWriter: tagWriter,
	}

	if meta.FrameRate > 0 {
		h264p.dtsStep = 1000.0 / meta.FrameRate
	}
	return h264p
}

func (h264p *h264Packetizer) PacketizeSequenceHeader() error {
	if h264p.spsMuxed {
		return nil
	}

	if !h264.MetadataIsReady(h264p.meta) {
		// not enough
		return nil
	}

	h264p.spsMuxed = true

	if h264p.meta.FixedFrameRate {
		h264p.dtsStep = 1000.0 / h264p.meta.FrameRate
	} else { // TODO:
		h264p.dtsStep = 1000.0 / 30
	}

	record := NewAVCDecoderConfigurationRecord(h264p.meta.Sps, h264p.meta.Pps)
	body, _ := record.Marshal()

	videoData := &VideoData{
		FrameType:       FrameTypeKeyFrame,
		CodecID:         CodecIDAVC,
		AVCPacketType:   AVCPacketTypeSequenceHeader,
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

	return h264p.tagWriter.WriteFlvTag(tag)
}

func (h264p *h264Packetizer) Packetize(basePts int64, frame *codec.Frame) error {
	if frame.Payload[0]&0x1F == h264.NalSps {
		return h264p.PacketizeSequenceHeader()
	}

	if frame.Payload[0]&0x1F == h264.NalPps {
		return h264p.PacketizeSequenceHeader()
	}

	dts := int64(h264p.nextDts)
	h264p.nextDts += h264p.dtsStep
	pts := frame.AbsTimestamp - basePts + ptsDelay
	if dts > pts {
		pts = dts
	}

	videoData := &VideoData{
		FrameType:       FrameTypeInterFrame,
		CodecID:         CodecIDAVC,
		AVCPacketType:   AVCPacketTypeNALU,
		CompositionTime: uint32(pts - dts),
		Body:            frame.Payload,
	}

	if frame.Payload[0]&0x1F == h264.NalIdrSlice {
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

	return h264p.tagWriter.WriteFlvTag(tag)
}
