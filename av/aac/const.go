// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

const (
	// SamplesPerFrame 每帧采样数
	SamplesPerFrame = 1024
)

// AAC Profile 表示使用哪个级别的 AAC。
// 如 01 Low Complexity(LC) – AAC LC
const (
	ProfileMain = iota // 0 Main profile
	ProfileLC          // 1 Low Complexity profile (LC)
	ProfileSSR         // 2 Scalable Sampling Rate profile (SSR)
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
	return sampleRates[sampleRate]
}

// 采用频率
var sampleRates = []int{
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
