// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/hevc"
)

type h265Depacketizer struct {
	fragments []*Packet // 分片包
	meta     *codec.VideoMeta
	w         codec.FrameWriter
	syncClock SyncClock
}

// NewH265Depacketizer 实例化 H265 帧提取器
func NewH265Depacketizer(meta *codec.VideoMeta, w codec.FrameWriter) Depacketizer {
	fe := &h265Depacketizer{
		meta:     meta,
		fragments: make([]*Packet, 0, 16),
		w:         w,
	}
	fe.syncClock.RTPTimeUnit = 1000.0 / float64(meta.ClockRate)
	return fe
}

func (h265dp *h265Depacketizer) Control(p *Packet) error {
	h265dp.syncClock.Decode(p.Data)
	return nil
}

/*
 * decode the HEVC payload header according to section 4 of draft version 6:
 *
 *    0                   1
 *    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
 *   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *   |F|   Type    |  LayerId  | TID |
 *   +-------------+-----------------+
 *
 *      Forbidden zero (F): 1 bit
 *      NAL unit type (Type): 6 bits
 *      NUH layer ID (LayerId): 6 bits
 *      NUH temporal ID plus 1 (TID): 3 bits
 *    decode the FU header
 *
 *     0 1 2 3 4 5 6 7
 *    +-+-+-+-+-+-+-+-+
 *    |S|E|  FuType   |
 *    +---------------+
 *
 *       Start fragment (S): 1 bit
 *       End fragment (E): 1 bit
 *       FuType: 6 bits
 */
func (h265dp *h265Depacketizer) Depacketize(packet *Packet) (err error) {
	if h265dp.syncClock.NTPTime == 0 { // 未收到同步时钟信息，忽略任意包
		return
	}

	payload := packet.Payload()
	if len(payload) < 3 {
		return
	}

	naluType := (payload[0] >> 1) & 0x3f

	switch naluType {
	case hevc.NalStapInRtp: // 在RTP中的聚合（AP）
		return h265dp.depacketizeStap(packet)
	case hevc.NalFuInRtp: // 在RTP中的扩展,分片(FU)
		return h265dp.depacketizeFu(packet)
	default:
		frame := &codec.Frame{
			MediaType:    codec.MediaTypeVideo,
			AbsTimestamp: h265dp.rtp2ntp(packet.Timestamp),
			Payload:      payload,
		}
		err = h265dp.writeFrame(frame)
		return
	}
}

func (h265dp *h265Depacketizer) depacketizeStap(packet *Packet) (err error) {
	payload := packet.Payload()
	off := 2 // 跳过 STAP NAL HDR

	// 循环读取被封装的NAL
	for {
		// nal长度
		nalSize := ((uint16(payload[off])) << 8) | uint16(payload[off+1])
		if nalSize < 1 {
			return
		}

		off += 2
		frame := &codec.Frame{
			MediaType:    codec.MediaTypeVideo,
			AbsTimestamp: h265dp.rtp2ntp(packet.Timestamp),
			Payload:      make([]byte, nalSize),
		}
		copy(frame.Payload, payload[off:])
		if err = h265dp.writeFrame(frame); err != nil {
			return
		}
		off += int(nalSize)
		if off >= len(payload) { // 扫描完成
			break
		}
	}
	return
}

func (h265dp *h265Depacketizer) depacketizeFu(packet *Packet) (err error) {
	payload := packet.Payload()
	rawDataOffset := 3 // 原始数据的偏移 = FU indicator + header

	//  0 1 2 3 4 5 6 7
	// +-+-+-+-+-+-+-+-+
	// |S|E|  FuType   |
	// +---------------+
	fuHeader := payload[2]

	if (fuHeader>>7)&1 == 1 { // 第一个分片包
		h265dp.fragments = h265dp.fragments[:0]
		// 缓存片段
		h265dp.fragments = append(h265dp.fragments, packet)
		return
	}

	if len(h265dp.fragments) == 0 || (len(h265dp.fragments) != 0 &&
		h265dp.fragments[len(h265dp.fragments)-1].SequenceNumber != packet.SequenceNumber-1) {
		// Packet loss ?
		h265dp.fragments = h265dp.fragments[:0]
		return
	}

	// 缓存其他片段
	h265dp.fragments = append(h265dp.fragments, packet)

	if (fuHeader>>6)&1 == 1 { // 最后一个片段
		frameLen := 2 // 计算帧总长,初始 naluType header len
		for _, fragment := range h265dp.fragments {
			frameLen += len(fragment.Payload()) - rawDataOffset
		}

		frame := &codec.Frame{
			MediaType:    codec.MediaTypeVideo,
			AbsTimestamp: h265dp.rtp2ntp(packet.Timestamp),
			Payload:      make([]byte, frameLen),
		}

		frame.Payload[0] = (payload[0] & 0x81) | (fuHeader&0x3f)<<1
		frame.Payload[1] = payload[1]
		offset := 2
		for _, fragment := range h265dp.fragments {
			payload := fragment.Payload()[rawDataOffset:]
			copy(frame.Payload[offset:], payload)
			offset += len(payload)
		}
		// 清空分片缓存
		h265dp.fragments = h265dp.fragments[:0]

		err = h265dp.writeFrame(frame)
	}

	return
}

func (h265dp *h265Depacketizer) rtp2ntp(timestamp uint32) int64 {
	return h265dp.syncClock.Rtp2Ntp(timestamp)
}

func (h265dp *h265Depacketizer) writeFrame(frame *codec.Frame) error {
	nalType := (frame.Payload[0] >> 1) & 0x3f
	switch nalType {
	case hevc.NalVps:
		if len(h265dp.meta.Vps) == 0 {
			h265dp.meta.Vps = frame.Payload
		}
	case hevc.NalSps:
		if len(h265dp.meta.Sps) == 0 {
			h265dp.meta.Sps = frame.Payload
		}
	case hevc.NalPps:
		if len(h265dp.meta.Pps) == 0 {
			h265dp.meta.Pps = frame.Payload
		}
	}
	return h265dp.w.WriteFrame(frame)
}
