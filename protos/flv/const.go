// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

// flv 标记类型ID
const (
	TagAudio    = 0x08
	TagVideo    = 0x09
	TagAmf3Data = 0x0F // 15
	TagAmf0Data = 0x12 // 18 onMetaData
)

// Amf0类型常量
const (
	Amf0TypeNumber            = 0x00
	Amf0TypeBoolean           = 0x01
	Amf0TypeString            = 0x02
	Amf0TypeObject            = 0x03
	Amf0TypeMovieClip         = 0x04 //reserved, not supported
	AMF0TypeNull              = 0x05
	Amf0TypeUndefined         = 0x06
	Amf0TypeReference         = 0x07
	Amf0TypeEcmaArray         = 0x08
	Amf0TypeObjectEnd         = 0x09
	Amf0TypeStrictArray       = 0x0A
	Amf0TypeDate              = 0x0B
	Amf0TypeLongString        = 0x0C
	Amf0TypeUnSupported       = 0x0D
	Amf0TypeRecordSet         = 0x0E
	Amf0TypeXMLDocument       = 0x0F
	Amf0TypeTypedObject       = 0x10
	Amf0TypeAVMplusObject     = 0x11
	Amf0TypeOriginStrictArray = 0x20
	Amf0TypeInvalid           = 0x3F
)

// Amf0 数据名称常量
const (
	// TagAmfNData 关联的数据
	Amf0DataOnMetaData   = "onMetaData"
	Amf0DataOnCustomData = "onCustomData"
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
	CodecVideoReserved               = 0
	CodecVideoReserved1              = 1
	CodecVideoSorensonH263           = 2
	CodecVideoScreenVideo            = 3
	CodecVideoOn2VP6                 = 4
	CodecVideoOn2VP6WithAlphaChannel = 5
	CodecVideoScreenVideoVersion2    = 6
	CodecVideoAVC                    = 7 // h264
	CodecVideoReserved2              = 8
	CodecVideoHEVC                   = 13 // 事实扩展标识 h265
)

// VideoCodecName 视频编解码器名称
func VideoCodecName(codec int32) string {
	switch codec {
	case CodecVideoSorensonH263:
		return "H263"
	case CodecVideoScreenVideo:
		return "ScreenVideo"
	case CodecVideoOn2VP6:
		return "On2VP6"
	case CodecVideoOn2VP6WithAlphaChannel:
		return "On2VP6WithAlphaChannel"
	case CodecVideoScreenVideoVersion2:
		return "ScreenVideoVersion2"
	case CodecVideoAVC:
		return "H264"
	case CodecVideoHEVC:
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
	CodecVideoAVCTypeSequenceHeader    = 0
	CodecVideoAVCTypeNALU              = 1
	CodecVideoAVCTypeSequenceHeaderEOF = 2
	CodecVideoAVCTypeReserved          = 3
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
	CodecVideoAVCFrameReserved             = 0
	CodecVideoAVCFrameKeyFrame             = 1 // video h264 key frame
	CodecVideoAVCFrameInterFrame           = 2 // video h264 inter frame
	CodecVideoAVCFrameDisposableInterFrame = 3
	CodecVideoAVCFrameGeneratedKeyFrame    = 4
	CodecVideoAVCFrameVideoInfoFrame       = 5
	CodecVideoAVCFrameReserved1            = 6
)

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
	CodecAudioLinearPCMPlatformEndian         = 0
	CodecAudioADPCM                           = 1
	CodecAudioMP3                             = 2
	CodecAudioLinearPCMLittleEndian           = 3
	CodecAudioNellymoser16kHzMono             = 4
	CodecAudioNellymoser8kHzMono              = 5
	CodecAudioNellymoser                      = 6
	CodecAudioReservedG711AlawLogarithmicPCM  = 7
	CodecAudioReservedG711MuLawLogarithmicPCM = 8
	CodecAudioReserved                        = 9
	CodecAudioAAC                             = 10
	CodecAudioSpeex                           = 11
	CodecAudioReservedMP3Of8kHz               = 14
	CodecAudioReservedDeviceSpecificSound     = 15
	CodecAudioReserved1                       = 16
)

// AudioCodecName 音频编码器名称
func AudioCodecName(codec int32) string {
	switch codec {
	case CodecAudioLinearPCMPlatformEndian:
		return "LinearPCMPlatformEndian"
	case CodecAudioADPCM:
		return "ADPCM"
	case CodecAudioLinearPCMLittleEndian:
		return "LinearPCMLittleEndian"
	case CodecAudioNellymoser16kHzMono:
		return "Nellymoser16kHzMono"
	case CodecAudioNellymoser8kHzMono:
		return "Nellymoser8kHzMono"
	case CodecAudioNellymoser:
		return "Nellymoser"
	case CodecAudioAAC:
		return "AAC"
	case CodecAudioSpeex:
		return "Speex"
	default:
		return ""
	}
}

// flv format
// AACPacketType IF SoundFormat == 10 UI8
// The following values are defined:
//     0 = AAC sequence header
//     1 = AAC raw
const (
	CodecAudioTypeSequenceHeader = 0 // 序列头
	CodecAudioTypeRawData        = 1 // 原始数据
	CodecAudioTypeReserved       = 2
)

// the FLV/RTMP supported audio sample rate.
// Sampling rate. The following values are defined:
// 0 = 5.5 kHz = 5512 Hz
// 1 = 11 kHz = 11025 Hz
// 2 = 22 kHz = 22050 Hz
// 3 = 44 kHz = 44100 Hz
const (
	CodecAudioSampleRate5512     = 0
	CodecAudioSampleRate11025    = 1
	CodecAudioSampleRate22050    = 2
	CodecAudioSampleRate44100    = 3
	CodecAudioSampleRateReserved = 4
)

// the FLV/RTMP supported audio sample size.
// Size of each audio sample. This parameter only pertains to
// uncompressed formats. Compressed formats always decode
// to 16 bits internally.
// 0 = 8-bit samples
// 1 = 16-bit samples
const (
	CodecAudioSampleSize8bit     = 0
	CodecAudioSampleSize16bit    = 1
	CodecAudioSampleSizeReserved = 2
)

// the FLV/RTMP supported audio sound type/channel.
// Mono or stereo sound
// 0 = Mono sound
// 1 = Stereo sound
const (
	CodecAudioSoundTypeMono     = 0
	CodecAudioSoundTypeStereo   = 1
	CodecAudioSoundTypeReserved = 2
)
