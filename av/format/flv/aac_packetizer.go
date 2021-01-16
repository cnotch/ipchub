// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import "github.com/cnotch/ipchub/av/codec"

// in ms, for aac flush the audio
const aacDelay = 100

type aacPacketizer struct {
	meta         *codec.AudioMeta
	dataTemplate *AudioData
	tagWriter    TagWriter
	spsMuxed     bool
}

func NewAacPacketizer(meta *codec.AudioMeta, tagWriter TagWriter) Packetizer {
	ap := &aacPacketizer{
		meta:      meta,
		tagWriter: tagWriter,
	}
	ap.prepareTemplate()
	return ap
}

func (ap *aacPacketizer) prepareTemplate() {
	audioData := &AudioData{
		SoundFormat:   SoundFormatAAC,
		AACPacketType: AACPacketTypeRawData,
		Body:          nil,
	}

	switch ap.meta.SampleRate {
	case 5512:
		audioData.SoundRate = SoundRate5512
	case 11025:
		audioData.SoundRate = SoundRate11025
	case 22050:
		audioData.SoundRate = SoundRate22050
	case 44100:
		audioData.SoundRate = SoundRate44100
	default:
		audioData.SoundRate = SoundRate44100
	}

	if ap.meta.SampleSize == 8 {
		audioData.SoundSize = SoundeSize8bit
	} else {
		audioData.SoundSize = SoundeSize16bit
	}

	if ap.meta.Channels > 1 {
		audioData.SoundType = SoundTypeStereo
	} else {
		audioData.SoundType = SoundTypeMono
	}

	ap.dataTemplate = audioData
}

func (ap *aacPacketizer) PacketizeSequenceHeader() error {
	if ap.spsMuxed {
		return nil
	}

	ap.spsMuxed = true
	audioData := *ap.dataTemplate
	audioData.AACPacketType = AACPacketTypeSequenceHeader
	audioData.Body = ap.meta.Sps
	data, _ := audioData.Marshal()

	tag := &Tag{
		TagType:   TagTypeAudio,
		DataSize:  uint32(len(data)),
		Timestamp: 0,
		StreamID:  0,
		Data:      data,
	}
	return ap.tagWriter.WriteFlvTag(tag)
}

func (ap *aacPacketizer) Packetize(basePts int64, frame *codec.Frame) error {
	audioData := *ap.dataTemplate
	audioData.Body = frame.Payload
	data, _ := audioData.Marshal()
	pts := aacDelay + frame.AbsTimestamp - basePts + ptsDelay
	
	tag := &Tag{
		TagType:   TagTypeAudio,
		DataSize:  uint32(len(data)),
		Timestamp: uint32(pts),
		StreamID:  0,
		Data:      data,
	}
	return ap.tagWriter.WriteFlvTag(tag)
}
