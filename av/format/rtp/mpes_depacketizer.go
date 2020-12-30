// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/aac"
)

type aacDepacketizer struct {
	w           codec.FrameWriter
	sizeLength  int
	indexLength int
	// extractFunc func(packet *Packet) error
	syncClock SyncClock
}

// NewAacDepacketizer 实例化 AAC 解包器
func NewAacDepacketizer(w codec.FrameWriter, rtpTimeUnit int) Depacketizer {
	fe := &aacDepacketizer{
		w:           w,
		sizeLength:  13,
		indexLength: 3,
	}
	fe.syncClock.RTPTimeUnit = 1000.0 / float64(rtpTimeUnit)
	return fe
}

func (aacdp *aacDepacketizer) Control(p *Packet) error {
	aacdp.syncClock.Decode(p.Data)
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
func (aacdp *aacDepacketizer) Depacketize(packet *Packet) (err error) {
	if aacdp.syncClock.NTPTime == 0 { // 未收到同步时钟信息，忽略任意包
		return
	}
	return aacdp.depacketizeFor2ByteAUHeader(packet)
}

func (aacdp *aacDepacketizer) depacketizeFor2ByteAUHeader(packet *Packet) (err error) {
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
		frameSize := auHeader >> aacdp.indexLength
		frame := &codec.Frame{
			FrameType:    codec.FrameAudio,
			AbsTimestamp: aacdp.rtp2ntp(frameTimeStamp),
			Payload:      framesPayload[:frameSize],
		}
		if err = aacdp.w.WriteFrame(frame); err != nil {
			return
		}

		// 下一帧
		auHeaders = auHeaders[2:]
		framesPayload = framesPayload[frameSize:]
		frameTimeStamp += aac.SamplesPerFrame // 每帧采样数
	}

	return
}

func (aacdp *aacDepacketizer) depacketizeFor1ByteAUHeader(packet *Packet) (err error) {
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
		frameSize := auHeader >> aacdp.indexLength
		frame := &codec.Frame{
			FrameType:    codec.FrameAudio,
			AbsTimestamp: aacdp.rtp2ntp(frameTimeStamp),
			Payload:      framesPayload[:frameSize],
		}
		if err = aacdp.w.WriteFrame(frame); err != nil {
			return
		}

		// 下一帧
		auHeaders = auHeaders[1:]
		framesPayload = framesPayload[frameSize:]
		frameTimeStamp += aac.SamplesPerFrame // 每帧采样数
	}

	return
}

func (aacdp *aacDepacketizer) rtp2ntp(timestamp uint32) int64 {
	return aacdp.syncClock.Rtp2Ntp(timestamp)
}
