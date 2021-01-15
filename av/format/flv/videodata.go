// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cnotch/ipchub/av/codec/hevc"
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
	CodecIDHEVC                   = 12 // 事实扩展标识 h265
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

// H2645PacketType IF CodecID == 7 or 12 UI8
// The following values are defined:
//     0 = AVC sequence header
//     1 = AVC NALU
//     2 = AVC end of sequence (lower level NALU sequence ender is
//         not required or supported)
const (
	H2645PacketTypeSequenceHeader    = 0
	H2645PacketTypeNALU              = 1
	H2645PacketTypeSequenceHeaderEOF = 2
	H2645PacketTypeReserved          = 3
)

// VideoData flv Tag 中的的视频数据
//
// 对于 CodecID == CodecIDAVC，Body 值:
// IF AVCPacketType == AVCPacketTypeSequenceHeader
//  AVCDecoderConfigurationRecord
// ELSE
// 　One or more NALUs (Full frames are required)
//
// 对于 CodecID == CodecIDAVC，Body 值:
// IF H2645PacketType == H2645PacketTypeSequenceHeader
//  AVCDecoderConfigurationRecord
// ELSE
// 　One or more NALUs (Full frames are required)
//
// 对于 CodecID == CodecIDHEVC，Body 值:
// IF H2645PacketType == H2645PacketTypeSequenceHeader
//  HEVCDecoderConfigurationRecord
// ELSE
// 　One or more NALUs (Full frames are required)
type VideoData struct {
	FrameType       byte   // 4 bits; 帧类型
	CodecID         byte   // 4 bits; 编解码器标识
	H2645PacketType byte   // 8 bits; 仅 AVC/HEVC 编码有效，AVC 包类型
	CompositionTime uint32 // 24 bits; 仅 AVC/HEVC 编码有效，表示PTS 与 DTS 的时间偏移值，单位 ms，记作 CTS。
	Body            []byte // 原始视频
}

var _ TagData = &VideoData{}

// Unmarshal .
// Note: Unmarshal not copy the data
func (videoData *VideoData) Unmarshal(data []byte) error {
	if len(data) < 1 {
		return errors.New("data.length < 1")
	}

	offset := 0

	videoData.FrameType = data[offset] >> 4
	videoData.CodecID = data[offset] & 0x0f
	offset++

	if videoData.CodecID == CodecIDAVC || videoData.CodecID == CodecIDHEVC {
		if len(data) < 5 {
			return errors.New("data.length < 5")
		}
		temp := binary.BigEndian.Uint32(data[offset:])
		videoData.H2645PacketType = byte(temp >> 24)
		videoData.CompositionTime = temp & 0x00ffffff
		offset += 4

		if videoData.H2645PacketType == H2645PacketTypeNALU {
			if len(data) < 9 {
				return errors.New("data.length < 9")
			}
			size := int(binary.BigEndian.Uint32(data[offset:]))
			offset += 4
			if size > len(data)-offset {
				return fmt.Errorf("data.length < %d", size+offset)
			}
		}
	}

	videoData.Body = data[offset:]
	return nil
}

// MarshalSize .
func (videoData *VideoData) MarshalSize() int {
	if videoData.H2645PacketType == H2645PacketTypeNALU {
		return 9 + len(videoData.Body)
	}
	return 5 + len(videoData.Body)
}

// Marshal .
func (videoData *VideoData) Marshal() ([]byte, error) {
	buff := make([]byte, videoData.MarshalSize())
	offset := 0
	buff[offset] = (videoData.FrameType << 4) | (videoData.CodecID & 0x0f)

	offset++
	if videoData.CodecID == CodecIDAVC || videoData.CodecID == CodecIDHEVC {
		binary.BigEndian.PutUint32(buff[offset:],
			(uint32(videoData.H2645PacketType)<<24)|(videoData.CompositionTime&0x00ffffff))
		offset += 4

		if videoData.H2645PacketType == H2645PacketTypeNALU {
			binary.BigEndian.PutUint32(buff[offset:], uint32(len(videoData.Body)))
			offset += 4
		}
	}

	offset += copy(buff[offset:], videoData.Body)

	return buff[:offset], nil
}

// AVCDecoderConfigurationRecord .
// aligned(8) class AVCDecoderConfigurationRecord {
//     unsigned int(8) configurationVersion = 1;
//     unsigned int(8) AVCProfileIndication;
//     unsigned int(8) profile_compatibility;
//     unsigned int(8) AVCLevelIndication;

//     bit(6) reserved = '111111'b;
//     unsigned int(2) lengthSizeMinusOne;

//     bit(3) reserved = '111'b;
//     unsigned int(5) numOfSequenceParameterSets;
//
//     for (i=0; i< numOfSequenceParameterSets; i++) {
//     	unsigned int(16) sequenceParameterSetLength ;
//         bit(8*sequenceParameterSetLength) sequenceParameterSetNALUnit;
//     }
//     unsigned int(8) numOfPictureParameterSets;
//     for (i=0; i< numOfPictureParameterSets; i++) {
//     	unsigned int(16) pictureParameterSetLength;
//     	bit(8*pictureParameterSetLength) pictureParameterSetNALUnit;
//     }
// }
type AVCDecoderConfigurationRecord struct {
	ConfigurationVersion byte
	AVCProfileIndication byte
	ProfileCompatibility byte
	AVCLevelIndication   byte
	SPS                  []byte
	PPS                  []byte
}

// NewAVCDecoderConfigurationRecord creates and initializes a new AVCDecoderConfigurationRecord
func NewAVCDecoderConfigurationRecord(sps, pps []byte) *AVCDecoderConfigurationRecord {
	return &AVCDecoderConfigurationRecord{
		ConfigurationVersion: 1,
		AVCProfileIndication: sps[1],
		ProfileCompatibility: sps[2],
		AVCLevelIndication:   sps[3],
		SPS:                  sps,
		PPS:                  pps,
	}
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

// MarshalSize .
func (record *AVCDecoderConfigurationRecord) MarshalSize() int {
	return 4 + 2 + 2 + len(record.SPS) + 1 + 2 + len(record.PPS)
}

// Marshal .
func (record *AVCDecoderConfigurationRecord) Marshal() ([]byte, error) {
	buff := make([]byte, record.MarshalSize())

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

// HEVCDecoderConfigurationRecord .
type HEVCDecoderConfigurationRecord struct {
	ConfigurationVersion uint8

	GeneralProfileSpace              uint8
	GeneralTierFlag                  uint8
	GeneralProfileIDC                uint8
	GeneralProfileCompatibilityFlags uint32
	GeneralConstraintIndicatorFlags  uint64
	GeneralLevelIDC                  uint8

	LengthSizeMinusOne uint8

	MaxSubLayers          uint8
	TemporalIdNestingFlag uint8

	ChromaFormatIDC      uint8
	BitDepthLumaMinus8   uint8
	BitDepthChromaMinus8 uint8

	VPS []byte
	SPS []byte
	PPS []byte
}

// NewHEVCDecoderConfigurationRecord creates and initializes a new HEVCDecoderConfigurationRecord
func NewHEVCDecoderConfigurationRecord(vps, sps, pps []byte) *HEVCDecoderConfigurationRecord {
	record := &HEVCDecoderConfigurationRecord{
		ConfigurationVersion:             1,
		LengthSizeMinusOne:               3, // 4 bytes
		GeneralProfileCompatibilityFlags: 0xffffffff,
		GeneralConstraintIndicatorFlags:  0xffffffffffff,
		VPS:                              vps,
		SPS:                              sps,
		PPS:                              pps,
	}

	record.init()
	return record
}

func (record *HEVCDecoderConfigurationRecord) init() error {
	var rawVps hevc.H265RawVPS
	if err := rawVps.Decode(record.VPS); err != nil {
		return err
	}
	if rawVps.Vps_max_sub_layers_minus1+1 > record.MaxSubLayers {
		record.MaxSubLayers = rawVps.Vps_max_sub_layers_minus1 + 1
	}
	record.applyPLT(&rawVps.Profile_tier_level)

	var rawSps hevc.H265RawSPS
	if err := rawSps.Decode(record.SPS); err != nil {
		return err
	}
	if rawSps.Sps_max_sub_layers_minus1+1 > record.MaxSubLayers {
		record.MaxSubLayers = rawSps.Sps_max_sub_layers_minus1 + 1
	}

	// sps_temporal_id_nesting_flag
	record.TemporalIdNestingFlag = rawSps.Sps_temporal_id_nesting_flag
	record.applyPLT(&rawSps.Profile_tier_level)

	record.ChromaFormatIDC = rawSps.Chroma_format_idc
	record.BitDepthLumaMinus8 = rawSps.Bit_depth_luma_minus8
	record.BitDepthChromaMinus8 = rawSps.Bit_depth_chroma_minus8

	return nil
}

func (record *HEVCDecoderConfigurationRecord) applyPLT(ptl *hevc.H265RawProfileTierLevel) {
	record.GeneralProfileSpace = ptl.General_profile_space

	if ptl.General_tier_flag > record.GeneralTierFlag {
		record.GeneralLevelIDC = ptl.General_level_idc

		record.GeneralTierFlag = ptl.General_tier_flag
	} else {
		if ptl.General_level_idc > record.GeneralLevelIDC {
			record.GeneralLevelIDC = ptl.General_level_idc
		}
	}

	if ptl.General_profile_idc > record.GeneralProfileIDC {
		record.GeneralProfileIDC = ptl.General_profile_idc
	}

	record.GeneralProfileCompatibilityFlags &= ptl.GeneralProfileCompatibilityFlags
	record.GeneralConstraintIndicatorFlags &= ptl.GeneralConstraintIndicatorFlags
}

// Unmarshal .
func (record *HEVCDecoderConfigurationRecord) Unmarshal(data []byte) error {
	if len(data) < 23 {
		return errors.New("data.length < 23")
	}
	offset := 0

	// unsigned int(8) configurationVersion = 1;
	record.ConfigurationVersion = data[offset]
	offset++

	// unsigned int(2) general_profile_space;
	// unsigned int(1) general_tier_flag;
	// unsigned int(5) general_profile_idc;
	record.GeneralProfileSpace = data[offset] >> 6
	record.GeneralTierFlag = (data[offset] >> 5) & 0x01
	record.GeneralProfileIDC = data[offset] & 0x1f
	offset++

	// unsigned int(32) general_profile_compatibility_flags
	record.GeneralProfileCompatibilityFlags = binary.BigEndian.Uint32(data[offset:])
	offset += 4

	// unsigned int(48) general_constraint_indicator_flags
	record.GeneralConstraintIndicatorFlags = uint64(binary.BigEndian.Uint32(data[offset:]))
	record.GeneralConstraintIndicatorFlags <<= 16
	offset += 4
	record.GeneralConstraintIndicatorFlags |= uint64(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	// unsigned int(8) general_level_idc;
	record.GeneralLevelIDC = data[offset]
	offset++

	// bit(4) reserved = ‘1111’b;
	// unsigned int(12) min_spatial_segmentation_idc;
	// bit(6) reserved = ‘111111’b;
	// unsigned int(2) parallelismType;
	offset += 2
	offset++

	// bit(6) reserved = ‘111111’b;
	// unsigned int(2) chromaFormat;
	record.ChromaFormatIDC = data[offset] & 0x03
	offset++

	// bit(5) reserved = ‘11111’b;
	// unsigned int(3) bitDepthLumaMinus8;
	record.BitDepthLumaMinus8 = data[offset] & 0x07
	offset++

	// bit(5) reserved = ‘11111’b;
	// unsigned int(3) bitDepthChromaMinus8;
	record.BitDepthChromaMinus8 = data[offset] & 0x07
	offset++

	// bit(16) avgFrameRate;
	offset += 2

	// bit(2) constantFrameRate;
	// bit(3) MaxSubLayers;
	// bit(1) temporalIdNested;
	// unsigned int(2) lengthSizeMinusOne;
	record.MaxSubLayers = (data[offset] >> 3) & 0x07
	record.TemporalIdNestingFlag = (data[offset] >> 2) & 0x01
	record.LengthSizeMinusOne = data[offset] & 0x03
	offset++

	// num of vps sps pps
	numNals := int(data[offset])
	offset++

	for i := 0; i < numNals; i++ {
		if len(data) < offset+5 {
			return errors.New("Insufficient data")
		}
		nalType := data[offset]
		offset++

		// num of vps
		num := binary.BigEndian.Uint16(data[offset:])
		offset += 2

		// length
		length := binary.BigEndian.Uint16(data[offset:])
		offset += 2
		if num != 1 {
			return errors.New("Multiple VPS or SPS or PPS NAL is not supported")
		}
		if len(data) < offset+int(length) {
			return errors.New("Insufficient raw data")
		}
		raw := data[offset : offset+int(length)]
		offset += int(length)
		switch nalType {
		case hevc.NalVps:
			record.VPS = raw
		case hevc.NalSps:
			record.SPS = raw
		case hevc.NalPps:
			record.PPS = raw
		default:
			return errors.New("Only VPS SPS PPS NAL is supported")
		}
	}
	return nil
}

// MarshalSize .
func (record *HEVCDecoderConfigurationRecord) MarshalSize() int {
	return 23 + 5 + len(record.VPS) + 5 + len(record.SPS) + 5 + len(record.PPS)
}

// Marshal .
func (record *HEVCDecoderConfigurationRecord) Marshal() ([]byte, error) {
	buff := make([]byte, record.MarshalSize())
	offset := 0

	// unsigned int(8) configurationVersion = 1;
	buff[offset] = 0x1
	offset++

	// unsigned int(2) general_profile_space;
	// unsigned int(1) general_tier_flag;
	// unsigned int(5) general_profile_idc;
	buff[offset] = record.GeneralProfileSpace<<6 | record.GeneralTierFlag<<5 | record.GeneralProfileIDC
	offset++

	// unsigned int(32) general_profile_compatibility_flags
	binary.BigEndian.PutUint32(buff[offset:], record.GeneralProfileCompatibilityFlags)
	offset += 4

	// unsigned int(48) general_constraint_indicator_flags
	binary.BigEndian.PutUint32(buff[offset:], uint32(record.GeneralConstraintIndicatorFlags>>16))
	offset += 4
	binary.BigEndian.PutUint16(buff[offset:], uint16(record.GeneralConstraintIndicatorFlags))
	offset += 2

	// unsigned int(8) general_level_idc;
	buff[offset] = record.GeneralLevelIDC
	offset++

	// bit(4) reserved = ‘1111’b;
	// unsigned int(12) min_spatial_segmentation_idc;
	// bit(6) reserved = ‘111111’b;
	// unsigned int(2) parallelismType;
	// TODO chef: 这两个字段没有解析
	binary.BigEndian.PutUint16(buff[offset:], 0xf000)
	offset += 2
	buff[offset] = 0xfc
	offset++

	// bit(6) reserved = ‘111111’b;
	// unsigned int(2) chromaFormat;
	buff[offset] = record.ChromaFormatIDC | 0xfc
	offset++

	// bit(5) reserved = ‘11111’b;
	// unsigned int(3) bitDepthLumaMinus8;
	buff[offset] = record.BitDepthLumaMinus8 | 0xf8
	offset++

	// bit(5) reserved = ‘11111’b;
	// unsigned int(3) bitDepthChromaMinus8;
	buff[offset] = record.BitDepthChromaMinus8 | 0xf8
	offset++

	// bit(16) avgFrameRate;
	binary.BigEndian.PutUint16(buff[offset:], 0)
	offset += 2

	// bit(2) constantFrameRate;
	// bit(3) numTemporalLayers;
	// bit(1) temporalIdNested;
	// unsigned int(2) lengthSizeMinusOne;
	buff[offset] = 0<<6 | record.MaxSubLayers<<3 | record.TemporalIdNestingFlag<<2 | record.LengthSizeMinusOne
	offset++

	// num of vps sps pps
	buff[offset] = 0x03
	offset++

	pset := []struct {
		nalType uint8
		data    []byte
	}{
		{hevc.NalVps, record.VPS},
		{hevc.NalSps, record.SPS},
		{hevc.NalPps, record.PPS},
	}
	for _, ps := range pset {
		buff[offset] = ps.nalType
		offset++

		// num of vps
		binary.BigEndian.PutUint16(buff[offset:], 1)
		offset += 2

		// length
		binary.BigEndian.PutUint16(buff[offset:], uint16(len(ps.data)))
		offset += 2

		copy(buff[offset:], ps.data)
		offset += len(ps.data)
	}

	return buff, nil
}
