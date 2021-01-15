// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Translate from FFmpeg mpeg4audio.h mpeg4audio.c
//
package aac

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/cnotch/ipchub/utils/bits"
)

// RawSps AudioSpecificConfig 的别名
type RawSPS = AudioSpecificConfig

// AudioSpecificConfig .
type AudioSpecificConfig struct {
	ObjectType       uint8
	SamplingIndex    uint8
	SampleRate       int
	ChannelConfig    uint8
	Sbr              int ///< -1 implicit, 1 presence
	ExtObjectType    uint8
	ExtSamplingIndex uint8
	ExtSampleRate    int
	ExtChannelConfig uint8
	Channels         uint8
	Ps               int ///< -1 implicit, 1 presence
	FrameLengthShort int
}

// DecodeString 从 hex 字串解码 sps
func (asc *AudioSpecificConfig) DecodeString(config string) error {
	data, err := hex.DecodeString(config)
	if err != nil {
		return err
	}
	return asc.Decode(data)
}

// Decode 从字节序列中解码 sps
func (asc *AudioSpecificConfig) Decode(config []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("AudioSpecificConfig decode panic；r = %v \n %s", r, debug.Stack())
		}
	}()

	r := bits.NewReader(config)

	asc.ObjectType = getObjectType(r)
	asc.SamplingIndex, asc.SampleRate = getSampleRate(r)
	asc.ChannelConfig = r.ReadUint8(4)
	if int(asc.ChannelConfig) < len(aacAudioChannels) {
		asc.Channels = aacAudioChannels[asc.ChannelConfig]
	}
	asc.Sbr = -1
	asc.Ps = -1
	if asc.ObjectType == AOT_SBR || (asc.ObjectType == AOT_PS &&
		0 == r.Peek(3)&0x03 && 0 == r.Peek(9)&0x3F) { // check for W6132 Annex YYYY draft MP3onMP4
		if asc.ObjectType == AOT_PS {
			asc.Ps = 1
		}
		asc.ExtObjectType = AOT_SBR
		asc.Sbr = 1
		asc.ExtSamplingIndex, asc.ExtSampleRate = getSampleRate(r)
		asc.ObjectType = getObjectType(r)
		if asc.ObjectType == AOT_ER_BSAC {
			asc.ExtChannelConfig = r.ReadUint8(4)
		}
	} else {
		asc.ExtObjectType = AOT_NULL
		asc.ExtSampleRate = 0
	}

	if asc.ObjectType == AOT_ALS {
		r.Skip(5)
		if uint32(r.Peek(24)) != binary.BigEndian.Uint32([]byte{0, 'A', 'L', 'S'}) {
			r.Skip(24)
		}

		if err = asc.parseConfigALS(r); err != nil {
			return
		}
	}

	if asc.ExtObjectType != AOT_SBR {
		for r.BitsLeft() > 15 {
			if r.Peek(11) == 0x2b7 { // sync extension
				r.Skip(11)
				asc.ExtObjectType = getObjectType(r)
				if asc.ExtObjectType == AOT_SBR {
					asc.Sbr = int(r.ReadBit())
					if asc.Sbr == 1 {
						asc.ExtSamplingIndex, asc.ExtSampleRate = getSampleRate(r)
						if asc.ExtSampleRate == asc.SampleRate {
							asc.Sbr = -1
						}
					}

				}
				if r.BitsLeft() > 11 && r.Read(11) == 0x548 {
					asc.Ps = int(r.ReadBit())
				}

				break
			} else {
				r.Skip(1) // skip 1 bit
			}
		}
	}

	//PS requires SBR
	if asc.Sbr == 0 {
		asc.Ps = 0
	}
	//Limit implicit PS to the HE-AACv2 Profile
	if (asc.Ps == -1 && asc.ObjectType != AOT_AAC_LC) || (asc.Channels&^0x01) != 0 {
		asc.Ps = 0
	}
	return
}

func (asc *AudioSpecificConfig) ToAdtsHeader(payloadSize int) ADTSHeader {
	sampleRateIdx := asc.SamplingIndex
	if asc.ExtSampleRate > 0 {
		sampleRateIdx = asc.ExtSamplingIndex
	}

	return NewADTSHeader(asc.ObjectType-1, sampleRateIdx, asc.ChannelConfig, payloadSize)
}

func Encode2BytesASC(objType, samplingIdx, channelConfig byte) []byte {
	var config = make([]byte, 2)
	config[0] = objType<<3 | (samplingIdx>>1)&0x07
	config[1] = samplingIdx<<7 | (channelConfig&0x0f)<<3
	return config
}

var errInvalidData = errors.New("Invalid data found when processing input")

func (asc *AudioSpecificConfig) parseConfigALS(r *bits.Reader) (err error) {
	if r.BitsLeft() < 112 {
		return errInvalidData
	}

	if r.Read(32) != binary.BigEndian.Uint32([]byte{'A', 'L', 'S', 0}) {
		return errInvalidData
	}

	// override AudioSpecificConfig channel configuration and sample rate
	// which are buggy in old ALS conformance files
	asc.SampleRate = r.ReadInt(32)

	if asc.SampleRate <= 0 {
		return errInvalidData
	}

	// skip number of samples
	r.Skip(32)

	// read number of channels
	asc.ChannelConfig = 0
	asc.Channels = uint8(r.ReadInt(16) + 1)
	return
}

func getObjectType(r *bits.Reader) (objType uint8) {
	objType = r.ReadUint8(5)

	if AOT_ESCAPE == objType {
		objType = r.ReadUint8(6) + 32
	}
	return
}

func getSampleRate(r *bits.Reader) (sampleRateIdx uint8, sampleRate int) {
	sampleRateIdx = r.ReadUint8(4)
	if sampleRateIdx == 0xf {
		sampleRate = r.ReadInt(24)
	} else {
		sampleRate = SampleRate(int(sampleRateIdx))
	}
	return
}
