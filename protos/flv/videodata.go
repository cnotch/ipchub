// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"encoding/binary"
	"errors"
)

// E.4.3.1 VIDEODATA
// Frame Type UB [4]
// Type of video frame. The following values are defined:
//     1 = key frame (for AVC, a seekable frame)
//     2 = inter frame (for AVC, a non-seekable frame)
//     3 = disposable inter frame (H.263 only)
//     4 = generated key frame (reserved for server use only)
//     5 = video info/command frame
const (
	FrameTypeReserved             = 0
	FrameTypeKeyFrame             = 1 // video h264 key frame
	FrameTypeInterFrame           = 2 // video h264 inter frame
	FrameTypeDisposableInterFrame = 3
	FrameTypeGeneratedKeyFrame    = 4
	FrameTypeVideoInfoFrame       = 5
	FrameTypeReserved1            = 6
)

// E.4.3.1 VIDEODATA
// CodecID UB [4]
// Codec Identifier. The following values are defined:
//     2 = Sorenson H.263
//     3 = Screen video
//     4 = On2 VP6
//     5 = On2 VP6 with alpha channel
//     6 = Screen video version 2
//     7 = AVC / H264
//     13 = HEVC / H265
const (
	CodecIDReserved               = 0
	CodecIDReserved1              = 1
	CodecIDSorensonH263           = 2
	CodecIDScreenVideo            = 3
	CodecIDOn2VP6                 = 4
	CodecIDOn2VP6WithAlphaChannel = 5
	CodecIDScreenVideoVersion2    = 6
	CodecIDAVC                    = 7 // h264
	CodecIDReserved2              = 8
	CodecIDHEVC                   = 13 // 事实扩展标识 h265
)

// CodecIDName 视频编解码器名称
func CodecIDName(codecID int32) string {
	switch codecID {
	case CodecIDSorensonH263:
		return "H263"
	case CodecIDScreenVideo:
		return "ScreenVideo"
	case CodecIDOn2VP6:
		return "On2VP6"
	case CodecIDOn2VP6WithAlphaChannel:
		return "On2VP6WithAlphaChannel"
	case CodecIDScreenVideoVersion2:
		return "ScreenVideoVersion2"
	case CodecIDAVC:
		return "H264"
	case CodecIDHEVC:
		return "H265"
	default:
		return ""
	}
}

// AVCPacketType IF CodecID == 7 UI8
// The following values are defined:
//     0 = AVC sequence header
//     1 = AVC NALU
//     2 = AVC end of sequence (lower level NALU sequence ender is
//         not required or supported)
const (
	AVCPacketTypeSequenceHeader    = 0
	AVCPacketTypeNALU              = 1
	AVCPacketTypeSequenceHeaderEOF = 2
	AVCPacketTypeReserved          = 3
)

// VideoData flv Tag 中的的视频数据
//
// 对于 CodecID == CodecIDAVC，Body 值:
// IF AVCPacketType == AVCPacketTypeSequenceHeader
//  AVCDecoderConfigurationRecord
// ELSE
// 　One or more NALUs (Full frames are required)
type VideoData struct {
	FrameType       byte   // 4 bits; 帧类型
	CodecID         byte   // 4 bits; 编解码器标识
	AVCPacketType   byte   // 8 bits; 仅 AVC 编码有效，AVC 包类型
	CompositionTime uint32 // 24 bits; 仅 AVC 编码有效，表示PTS 与 DTS 的时间偏移值，单位 ms，记作 CTS。
	Body            []byte // 原始视频
}

// Unmarshal .
// Note: Unmarshal not copy the data
func (videoData *VideoData) Unmarshal(data []byte) error {
	if len(data) < 5 {
		return errors.New("data.length < 5")
	}

	offset := 0

	videoData.FrameType = data[offset] >> 4
	videoData.CodecID = data[offset] & 0x0f
	offset++

	if videoData.CodecID == CodecIDAVC {
		temp := binary.BigEndian.Uint32(data[offset:])
		videoData.AVCPacketType = byte(temp >> 24)
		videoData.CompositionTime = temp & 0x00ffffff
		offset += 4
	}

	videoData.Body = data[offset:]
	return nil
}

// Marshal .
func (videoData *VideoData) Marshal() ([]byte, error) {
	buff := make([]byte, 5+len(videoData.Body))
	offset := 0
	buff[offset] = (videoData.FrameType << 4) | (videoData.CodecID & 0x0f)

	offset++
	if videoData.CodecID == CodecIDAVC {
		binary.BigEndian.PutUint32(buff[offset:],
			(uint32(videoData.AVCPacketType)<<24)|(videoData.CompositionTime&0x00ffffff))
		offset += 4
	}

	offset += copy(buff[offset:], videoData.Body)

	return buff[:offset], nil
}

// AVCDecoderConfigurationRecord .
type AVCDecoderConfigurationRecord struct {
	ConfigurationVersion byte
	AVCProfileIndication byte
	ProfileCompatibility byte
	AVCLevelIndication   byte
	SPS                  []byte
	PPS                  []byte
}

// Unmarshal .
// Note: Unmarshal not copy the data
func (record *AVCDecoderConfigurationRecord) Unmarshal(data []byte) error {
	if len(data) < 11 {
		return errors.New("data.length < 11")
	}

	offset := 0

	record.ConfigurationVersion = data[offset]
	offset++

	record.AVCProfileIndication = data[offset]
	offset++

	record.ProfileCompatibility = data[offset]
	offset++

	record.AVCLevelIndication = data[offset]
	offset++

	offset += 2 //

	spsLen := binary.BigEndian.Uint16(data[offset:])
	offset += 2

	if len(data) < 11+int(spsLen) {
		return errors.New("Insufficient Data: SPS")
	}
	record.SPS = data[offset : offset+int(spsLen)]
	offset += int(spsLen)

	offset++

	ppsLen := binary.BigEndian.Uint16(data[offset:])
	offset += 2

	if len(data) < 11+int(spsLen)+int(ppsLen) {
		return errors.New("Insufficient Data: PPS")
	}
	record.PPS = data[offset : offset+int(ppsLen)]
	return nil
}

// Marshal .
func (record *AVCDecoderConfigurationRecord) Marshal() ([]byte, error) {
	buff := make([]byte, 4+2+2+len(record.SPS)+1+2+len(record.PPS))

	offset := 0

	buff[offset] = record.ConfigurationVersion
	offset++

	buff[offset] = record.AVCProfileIndication
	offset++

	buff[offset] = record.ProfileCompatibility
	offset++

	buff[offset] = record.AVCLevelIndication
	offset++

	// lengthSizeMinusOne 是 H.264 视频中 NALU 的长度，
	// 计算方法是 1 + (lengthSizeMinusOne & 3)，实际计算结果一直是4
	buff[offset] = 0xff
	offset++

	// numOfSequenceParameterSets SPS 的个数，计算方法是 numOfSequenceParameterSets & 0x1F，
	// 实际计算结果一直为1
	buff[offset] = 0xe1
	offset++

	// sequenceParameterSetLength SPS 的长度
	binary.BigEndian.PutUint16(buff[offset:], uint16(len(record.SPS)))
	offset += 2

	// SPS data
	offset += copy(buff[offset:], record.SPS)

	// numOfPictureParameterSets PPS 的个数
	buff[offset] = 0x01
	offset++

	// pictureParameterSetLength SPS 的长度
	binary.BigEndian.PutUint16(buff[offset:], uint16(len(record.PPS)))
	offset += 2

	// PPS data
	offset += copy(buff[offset:], record.PPS)

	return buff, nil
}
