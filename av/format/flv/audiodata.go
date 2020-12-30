// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import "errors"

// SoundFormat UB [4]
// Format of SoundData. The following values are defined:
//     0 = Linear PCM, platform endian
//     1 = ADPCM
//     2 = MP3
//     3 = Linear PCM, little endian
//     4 = Nellymoser 16 kHz mono
//     5 = Nellymoser 8 kHz mono
//     6 = Nellymoser
//     7 = G.711 A-law logarithmic PCM
//     8 = G.711 mu-law logarithmic PCM
//     9 = reserved
//     10 = AAC
//     11 = Speex
//     14 = MP3 8 kHz
//     15 = Device-specific sound
// Formats 7, 8, 14, and 15 are reserved.
// AAC is supported in Flash Player 9,0,115,0 and higher.
// Speex is supported in Flash Player 10 and higher.
const (
	SoundFormatLinearPCMPlatformEndian         = 0
	SoundFormatADPCM                           = 1
	SoundFormatMP3                             = 2
	SoundFormatLinearPCMLittleEndian           = 3
	SoundFormatNellymoser16kHzMono             = 4
	SoundFormatNellymoser8kHzMono              = 5
	SoundFormatNellymoser                      = 6
	SoundFormatReservedG711AlawLogarithmicPCM  = 7
	SoundFormatReservedG711MuLawLogarithmicPCM = 8
	SoundFormatReserved                        = 9
	SoundFormatAAC                             = 10
	SoundFormatSpeex                           = 11
	SoundFormatReservedMP3Of8kHz               = 14
	SoundFormatReservedDeviceSpecificSound     = 15
	SoundFormatReserved1                       = 16
)

// SoundFormatName 音频格式名称
func SoundFormatName(codec int32) string {
	switch codec {
	case SoundFormatLinearPCMPlatformEndian:
		return "LinearPCMPlatformEndian"
	case SoundFormatADPCM:
		return "ADPCM"
	case SoundFormatLinearPCMLittleEndian:
		return "LinearPCMLittleEndian"
	case SoundFormatNellymoser16kHzMono:
		return "Nellymoser16kHzMono"
	case SoundFormatNellymoser8kHzMono:
		return "Nellymoser8kHzMono"
	case SoundFormatNellymoser:
		return "Nellymoser"
	case SoundFormatAAC:
		return "AAC"
	case SoundFormatSpeex:
		return "Speex"
	default:
		return ""
	}
}

// the FLV/RTMP supported audio sample rate.
// Sampling rate. The following values are defined:
// 0 = 5.5 kHz = 5512 Hz
// 1 = 11 kHz = 11025 Hz
// 2 = 22 kHz = 22050 Hz
// 3 = 44 kHz = 44100 Hz
const (
	SoundRate5512     = 0
	SoundRate11025    = 1
	SoundRate22050    = 2
	SoundRate44100    = 3
	SoundRateReserved = 4
)

// the FLV/RTMP supported audio sample size.
// Size of each audio sample. This parameter only pertains to
// uncompressed formats. Compressed formats always decode
// to 16 bits internally.
// 0 = 8-bit samples
// 1 = 16-bit samples
const (
	SoundeSize8bit     = 0
	SoundeSize16bit    = 1
	SoundeSizeReserved = 2
)

// the FLV/RTMP supported audio sound type/channel.
// Mono or stereo sound
// 0 = Mono sound
// 1 = Stereo sound
const (
	SoundTypeMono     = 0
	SoundTypeStereo   = 1
	SoundTypeReserved = 2
)

// flv format
// AACPacketType IF SoundFormat == 10 UI8
// The following values are defined:
//     0 = AAC sequence header
//     1 = AAC raw
const (
	AACPacketTypeSequenceHeader = 0 // 序列头
	AACPacketTypeRawData        = 1 // 原始数据
	AACPacketTypeReserved       = 2
)

// AudioData flv Tag 中的的音频数据
//
// 对于 SoundFormat == SoundFormatAAC，Body 值:
// IF AACPacketType == AACPacketTypeSequenceHeader
// 　AudioSpecificConfig 参考 AAC.RawSPS
// ELSE
// 　Raw AAC frame data
type AudioData struct {
	SoundFormat   byte   // 4 bits; 音频编码格式
	SoundRate     byte   // 2 bits; 音频采样率
	SoundSize     byte   // 1 bits; 音频采用大小
	SoundType     byte   // 1 bits; 音频通道类型
	AACPacketType byte   // 8 bits; AAC 编码音频的包类型，仅但 SoundFormat 为 AAC 有效
	Body          []byte // 原始音频
}

// Unmarshal .
// Note: Unmarshal not copy the data
func (audioData *AudioData) Unmarshal(data []byte) error {
	if len(data) < 2 {
		return errors.New("data.length < 2")
	}

	offset := 0
	audioData.SoundFormat = data[offset] >> 4
	audioData.SoundRate = (data[offset] >> 2) & 0x03
	audioData.SoundSize = (data[offset] >> 1) & 0x01
	audioData.SoundType = data[offset] & 0x01

	offset++
	if audioData.SoundFormat == SoundFormatAAC {
		audioData.AACPacketType = data[offset]
		offset++
	}
	audioData.Body = data[offset:]

	return nil
}

// MarshalSize .
func (audioData *AudioData) MarshalSize() int {
	return 2 + len(audioData.Body)
}

// Marshal .
func (audioData *AudioData) Marshal() ([]byte, error) {
	buff := make([]byte, audioData.MarshalSize())
	offset := 0
	buff[offset] = (audioData.SoundFormat << 4) |
		((audioData.SoundRate & 0x03) << 2) |
		((audioData.SoundSize & 0x01) << 1) |
		(audioData.SoundType & 0x01)

	offset++
	if audioData.SoundFormat == SoundFormatAAC {
		buff[offset] = audioData.AACPacketType
		offset++
	}

	offset += copy(buff[offset:], audioData.Body)

	return buff[:offset], nil
}
