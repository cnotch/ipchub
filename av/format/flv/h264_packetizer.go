// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/h264"
)

type h264Packetizer struct {
	meta      *codec.VideoMeta
	tagWriter TagWriter
}

func NewH264Packetizer(meta *codec.VideoMeta, tagWriter TagWriter) Packetizer {
	h264p := &h264Packetizer{
		meta:      meta,
		tagWriter: tagWriter,
	}
	return h264p
}

func (h264p *h264Packetizer) PacketizeSequenceHeader() error {
	record := NewAVCDecoderConfigurationRecord(h264p.meta.Sps, h264p.meta.Pps)
	body, _ := record.Marshal()

	videoData := &VideoData{
		FrameType:       FrameTypeKeyFrame,
		CodecID:         CodecIDAVC,
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

	return h264p.tagWriter.WriteFlvTag(tag)
}

func (h264p *h264Packetizer) Packetize(frame *codec.Frame) error {

	dts := frame.Dts / int64(time.Millisecond)
	pts := frame.Pts / int64(time.Millisecond)

	videoData := &VideoData{
		FrameType:       FrameTypeInterFrame,
		CodecID:         CodecIDAVC,
		H2645PacketType: H2645PacketTypeNALU,
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
