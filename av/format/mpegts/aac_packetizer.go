// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mpegts

import (
	"fmt"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/aac"
)

// in ms, for aac flush the audio
const aacDelay = 100

type aacPacketizer struct {
	meta          *codec.AudioMeta
	tsframeWriter FrameWriter
	audioSps      *aac.RawSPS
}

func NewAacPacketizer(meta *codec.AudioMeta, tsframeWriter FrameWriter) Packetizer {
	ap := &aacPacketizer{
		meta:          meta,
		tsframeWriter: tsframeWriter,
	}
	ap.prepareAsc()
	return ap
}

func (ap *aacPacketizer) prepareAsc() (err error) {
	if ap.audioSps != nil {
		return
	}

	var asc aac.AudioSpecificConfig
	asc.Decode(ap.meta.Sps)
	if err = asc.Decode(ap.meta.Sps); err != nil {
		return
	}

	if asc.ObjectType == aac.AOT_NULL || asc.ObjectType == aac.AOT_ESCAPE {
		err = fmt.Errorf("tsmuxer decdoe audio aac sequence header failed, aac object type=%d", asc.ObjectType)
		return
	}
	ap.audioSps = &asc
	return
}

func (ap *aacPacketizer) Packetize(basePts int64, frame *codec.Frame) error {
	pts := frame.AbsTimestamp - basePts + ptsDelay
	pts *= 90

	// set fields
	tsframe := &Frame{
		Pid:      tsAudioPid,
		StreamID: tsAudioAac,
		Dts:      pts,
		Pts:      pts,
		Payload:  frame.Payload,
	}

	tsframe.prepareAacHeader(ap.audioSps)
	return ap.tsframeWriter.WriteMpegtsFrame(tsframe)
}
