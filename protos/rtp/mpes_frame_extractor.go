// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"github.com/cnotch/ipchub/av/aac"
)

type mpesFrameExtractor struct {
	w           FrameWriter
	sizeLength  int
	indexLength int
	// extractFunc func(packet *Packet) error
	syncClock   SyncClock
	rtpTimeUnit int
}

// NewMPESFrameExtractor 实例化 MPES 帧提取器
func NewMPESFrameExtractor(w FrameWriter, rtpTimeUnit int) FrameExtractor {
	fe := &mpesFrameExtractor{
		w:           w,
		sizeLength:  13,
		indexLength: 3,
		rtpTimeUnit: rtpTimeUnit,
	}
	return fe
}

func (fe *mpesFrameExtractor) Control(p *Packet) error {
	fe.syncClock.Decode(p.Data)
	return nil
}

//  以下是当 sizelength=13;indexlength=3;indexdeltalength=3 时
//  Au-header = 13+3 bits(2byte) 的示意图
// 	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
//  |       AU-headers-length     |
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
//  |       AU-header(1)          |
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
//  |       AU-header(2)          |
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
//  |       ...                   |
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
//  |       AU-header(n)          |
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
//  |       pading bits           |
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
// 当 sizelength=6;indexlength=2;indexdeltalength=2 时
// 单帧封装时，rtp payload的长度 = AU-header-lengths(两个字节) + AU-header(6+2) + AU的长度
func (fe *mpesFrameExtractor) Extract(packet *Packet) (err error) {
	if fe.syncClock.NTPTime == 0 { // 未收到同步时钟信息，忽略任意包
		return
	}
	return fe.extractFor2ByteAUHeader(packet)
}

func (fe *mpesFrameExtractor) extractFor2ByteAUHeader(packet *Packet) (err error) {
	payload := packet.Payload()

	// AU-headers-length 2bytes
	auHeadersLength := uint16(0) | (uint16(payload[0]) << 8) | uint16(payload[1])
	// AU-headers-length / 16
	auHeadersCount := auHeadersLength >> 4
	// AU 帧数据偏移位置
	framesPayloadOffset := 2 + int(auHeadersCount)<<1

	auHeaders := payload[2:framesPayloadOffset]
	framesPayload := payload[framesPayloadOffset:]
	frameTimeStamp := packet.Timestamp
	for i := 0; i < int(auHeadersCount); i++ {
		auHeader := uint16(0) | (uint16(auHeaders[0]) << 8) | uint16(auHeaders[1])
		frameSize := auHeader >> fe.indexLength
		frame := &Frame{
			FrameType: FrameAudio,
			NTPTime:   fe.rtp2ntp(frameTimeStamp),
			RTPTime:   frameTimeStamp,
			Payload:   framesPayload[:frameSize],
		}
		if err = fe.w.WriteFrame(frame); err != nil {
			return
		}

		// 下一帧
		auHeaders = auHeaders[2:]
		framesPayload = framesPayload[frameSize:]
		frameTimeStamp += aac.SamplesPerFrame // 每帧采样数
	}

	return
}

func (fe *mpesFrameExtractor) extractFor1ByteAUHeader(packet *Packet) (err error) {
	payload := packet.Payload()

	// AU-headers-length 2bytes
	auHeadersLength := uint16(0) | (uint16(payload[0]) << 8) | uint16(payload[1])
	// AU-headers-length / 16
	auHeadersCount := auHeadersLength >> 4
	// AU 帧数据偏移位置
	framesPayloadOffset := 2 + int(auHeadersCount)

	auHeaders := payload[2:framesPayloadOffset]
	framesPayload := payload[framesPayloadOffset:]
	frameTimeStamp := packet.Timestamp
	for i := 0; i < int(auHeadersCount); i++ {
		auHeader := auHeaders[0]
		frameSize := auHeader >> fe.indexLength
		frame := &Frame{
			FrameType: FrameAudio,
			NTPTime:   fe.rtp2ntp(frameTimeStamp),
			RTPTime:   frameTimeStamp,
			Payload:   framesPayload[:frameSize],
		}
		if err = fe.w.WriteFrame(frame); err != nil {
			return
		}

		// 下一帧
		auHeaders = auHeaders[1:]
		framesPayload = framesPayload[frameSize:]
		frameTimeStamp += aac.SamplesPerFrame // 每帧采样数
	}

	return
}

func (fe *mpesFrameExtractor) rtp2ntp(timestamp uint32) int64 {
	return fe.syncClock.Rtp2Ntp(timestamp, fe.rtpTimeUnit)
}
