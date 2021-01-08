// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

import "sort"

const (
	// SamplesPerFrame 每帧采样数
	SamplesPerFrame = 1024
)

// Auido Object Type
const (
	AOT_NULL            = iota     ///< Support?                Name
	AOT_AAC_MAIN                   ///< Y                       Main
	AOT_AAC_LC                     ///< Y                       Low Complexity
	AOT_AAC_SSR                    ///< N (code in SoC repo)    Scalable Sample Rate
	AOT_AAC_LTP                    ///< Y                       Long Term Prediction
	AOT_SBR                        ///< Y                       Spectral Band Replication HE-AAC
	AOT_AAC_SCALABLE               ///< N                       Scalable
	AOT_TWINVQ                     ///< N                       Twin Vector Quantizer
	AOT_CELP                       ///< N                       Code Excited Linear Prediction
	AOT_HVXC                       ///< N                       Harmonic Vector eXcitation Coding
	AOT_TTSI            = 2 + iota ///< N(code = 12)            Text-To-Speech Interface
	AOT_MAINSYNTH                  ///< N                       Main Synthesis
	AOT_WAVESYNTH                  ///< N                       Wavetable Synthesis
	AOT_MIDI                       ///< N                       General MIDI
	AOT_SAFX                       ///< N                       Algorithmic Synthesis and Audio Effects
	AOT_ER_AAC_LC                  ///< N                       Error Resilient Low Complexity
	AOT_ER_AAC_LTP      = 3 + iota ///< N(code = 19)            Error Resilient Long Term Prediction
	AOT_ER_AAC_SCALABLE            ///< N                       Error Resilient Scalable
	AOT_ER_TWINVQ                  ///< N                       Error Resilient Twin Vector Quantizer
	AOT_ER_BSAC                    ///< N                       Error Resilient Bit-Sliced Arithmetic Coding
	AOT_ER_AAC_LD                  ///< N                       Error Resilient Low Delay
	AOT_ER_CELP                    ///< N                       Error Resilient Code Excited Linear Prediction
	AOT_ER_HVXC                    ///< N                       Error Resilient Harmonic Vector eXcitation Coding
	AOT_ER_HILN                    ///< N                       Error Resilient Harmonic and Individual Lines plus Noise
	AOT_ER_PARAM                   ///< N                       Error Resilient Parametric
	AOT_SSC                        ///< N                       SinuSoidal Coding
	AOT_PS                         ///< N                       Parametric Stereo
	AOT_SURROUND                   ///< N                       MPEG Surround
	AOT_ESCAPE                     ///< Y                       Escape Value
	AOT_L1                         ///< Y                       Layer 1
	AOT_L2                         ///< Y                       Layer 2
	AOT_L3                         ///< Y                       Layer 3
	AOT_DST                        ///< N                       Direct Stream Transfer
	AOT_ALS                        ///< Y                       Audio LosslesS
	AOT_SLS                        ///< N                       Scalable LosslesS
	AOT_SLS_NON_CORE               ///< N                       Scalable LosslesS (non core)
	AOT_ER_AAC_ELD                 ///< N                       Error Resilient Enhanced Low Delay
	AOT_SMR_SIMPLE                 ///< N                       Symbolic Music Representation Simple
	AOT_SMR_MAIN                   ///< N                       Symbolic Music Representation Main
	AOT_USAC_NOSBR                 ///< N                       Unified Speech and Audio Coding (no SBR)
	AOT_SAOC                       ///< N                       Spatial Audio Object Coding
	AOT_LD_SURROUND                ///< N                       Low Delay MPEG Surround
	AOT_USAC                       ///< N                       Unified Speech and Audio Coding
)

// AAC Profile 表示使用哪个级别的 AAC。
// 如 01 Low Complexity(LC) – AAC LC
const (
	ProfileMain = AOT_AAC_MAIN - 1
	ProfileLow  = AOT_AAC_LC - 1
	ProfileSSR  = AOT_AAC_SSR - 1
	ProfileLTP  = AOT_AAC_LTP - 1
	ProfileHE   = AOT_SBR - 1
	ProfileLD   = AOT_ER_AAC_LD - 1
	ProfileHE2  = AOT_PS - 1
	ProfileELD  = AOT_ER_AAC_ELD - 1
)

// AAC 采样频率
const (
	SampleRate96000 = iota // 0
	SampleRate88200        // 1
	SampleRate64000        // 2
	SampleRate48000        // 3
	SampleRate44100        // 4
	SampleRate32000        // 5
	SampleRate24000        // 6
	SampleRate22050        // 7
	SampleRate16000        // 8
	SampleRate12000        // 9
	SampleRate11025        // 10
	SampleRate8000         // 11
	SampleRate7350         // 12
)

// SampleRate 获取采用频率具体值
func SampleRate(index int) int {
	return SampleRates[index]
}

// SamplingIndex .
func SamplingIndex(rate int) int {
	i := sort.Search(len(SampleRates), func(i int) bool { return SampleRates[i] <= rate })
	if i < len(SampleRates) && SampleRates[i] == rate {
		return i
	}
	return -1
}

// SampleRates 采用频率集合
var SampleRates = [16]int{
	96000, 88200, 64000, 48000,
	44100, 32000, 24000, 22050,
	16000, 12000, 11025, 8000,
	7350}

// ACC ChannelConfig 声道配置
// 0x00 - defined in audioDecderSpecificConfig
// 0x01 单声道（center front speaker）
// 0x02 双声道（left, right front speakers）
// 0x03 三声道（center, left, right front speakers）
// 0x04 四声道（center, left, right front speakers, rear surround speakers）
// 0x05 五声道（center, left, right front speakers, left surround, right surround rear speakers）
// 0x06 5.1声道（center, left, right front speakers, left surround, right surround rear speakers, front low frequency effects speaker)
// 0x07 7.1声道（center, left, right center front speakers, left, right outside front speakers, left surround, right surround rear speakers, front low frequency effects speaker)
// 0x08-0x0F - reserved
const (
	ChannelSpecific     = iota // 0
	ChannelMono                // 1
	ChannelStereo              // 2
	ChannelThree               // 3
	ChannelFour                // 4
	ChannelFive                // 5
	ChannelFivePlusOne         // 6
	ChannelSevenPlusOne        // 7
	ChannelReserved            // 8
)

var aacAudioChannels = [8]uint8{
	0, 1, 2, 3,
	4, 5, 6, 8,
}

// 参数集索引
const (
	ParameterSetConfig = 0
)
