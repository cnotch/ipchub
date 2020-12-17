// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

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
