// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/hevc"
)

type h265Packetizer struct {
	meta      *codec.VideoMeta
	tagWriter TagWriter
}

func NewH265Packetizer(meta *codec.VideoMeta, tagWriter TagWriter) Packetizer {
	h265p := &h265Packetizer{
		meta:      meta,
		tagWriter: tagWriter,
	}
	return h265p
}

func (h265p *h265Packetizer) PacketizeSequenceHeader() error {
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

func (h265p *h265Packetizer) Packetize(frame *codec.Frame) error {
	nalType := (frame.Payload[0] >> 1) & 0x3f
	dts := frame.Dts / int64(time.Millisecond)
	pts := frame.Pts / int64(time.Millisecond)

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
