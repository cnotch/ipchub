// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"fmt"
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/h264"
)

type h264Depacketizer struct {
	fragments []*Packet // 分片包
	meta      *codec.VideoMeta
	metaReady bool
	nextDts   float64
	dtsStep   float64
	startOn   time.Time
	w         codec.FrameWriter
	syncClock SyncClock
}

// NewH264Depacketizer 实例化 H264 帧提取器
func NewH264Depacketizer(meta *codec.VideoMeta, w codec.FrameWriter) Depacketizer {
	h264dp := &h264Depacketizer{
		meta:      meta,
		fragments: make([]*Packet, 0, 16),
		w:         w,
	}
	h264dp.syncClock.RTPTimeUnit = float64(time.Second) / float64(meta.ClockRate)
	return h264dp
}

func (h264dp *h264Depacketizer) Control(basePts *int64, p *Packet) error {
	if ok := h264dp.syncClock.Decode(p.Data); ok {
		if *basePts == 0 {
			*basePts = h264dp.syncClock.NTPTime
		}
	}
	return nil
}

func (h264dp *h264Depacketizer) Depacketize(basePts int64, packet *Packet) (err error) {
	if h264dp.syncClock.NTPTime == 0 { // 未收到同步时钟信息，忽略任意包
		return
	}

	payload := packet.Payload()
	if len(payload) < 3 {
		return
	}

	// +---------------+
	// |0|1|2|3|4|5|6|7|
	// +-+-+-+-+-+-+-+-+
	// |F|NRI|  Type   |
	// +---------------+
	naluType := payload[0] & h264.NalTypeBitmask

	switch {
	case naluType < h264.NalStapaInRtp:
		// h264 原生 nal 包
		// 	0                   1                   2                   3
		// 	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//  |F|NRI|  type   |                                               |
		//  +-+-+-+-+-+-+-+-+                                               |
		//  |                                                               |
		//  |               Bytes 2..n of a Single NAL unit                 |
		//  |                                                               |
		//  |                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//  |                               :...OPTIONAL RTP padding        |
		//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		frame := &codec.Frame{
			MediaType: codec.MediaTypeVideo,
			Payload:   payload,
		}
		err = h264dp.writeFrame(basePts, packet.Timestamp, frame)
	case naluType == h264.NalStapaInRtp:
		err = h264dp.depacketizeStapa(basePts, packet)
	case naluType == h264.NalFuAInRtp:
		err = h264dp.depacketizeFuA(basePts, packet)
	default:
		err = fmt.Errorf("nalu type %d is currently not handled", naluType)
	}
	return
}

func (h264dp *h264Depacketizer) depacketizeStapa(basePts int64, packet *Packet) (err error) {
	payload := packet.Payload()
	header := payload[0]

	// 	0                   1                   2                   3
	// 	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//  |STAP-A NAL HDR |         NALU 1 Size           | NALU 1 HDR    |
	//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//  |                         NALU 1 Data                           |
	//  :                                                               :
	//  +               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//  |               | NALU 2 Size                   | NALU 2 HDR    |
	//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//  |                         NALU 2 Data                           |
	//  :                                                               :
	//  |                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//  |                               :...OPTIONAL RTP padding        |
	//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	off := 1 // 跳过 STAP-A NAL HDR
	// 循环读取被封装的NAL
	for {
		// nal长度
		nalSize := ((uint16(payload[off])) << 8) | uint16(payload[off+1])
		if nalSize < 1 {
			return
		}

		off += 2
		frame := &codec.Frame{
			MediaType: codec.MediaTypeVideo,
			Payload:   make([]byte, nalSize),
		}
		copy(frame.Payload, payload[off:])
		frame.Payload[0] = 0 | (header & 0x60) | (frame.Payload[0] & 0x1F)
		if err = h264dp.writeFrame(basePts, packet.Timestamp,frame); err != nil {
			return
		}

		off += int(nalSize)
		if off >= len(payload) { // 扫描完成
			break
		}
	}
	return
}

func (h264dp *h264Depacketizer) depacketizeFuA(basePts int64, packet *Packet) (err error) {
	payload := packet.Payload()
	header := payload[0]

	// 	0                   1                   2                   3
	// 	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//  | FU indicator  |   FU header   |                               |
	//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               |
	//  |                                                               |
	//  |                         FU payload                            |
	//  |                                                               |
	//  |                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//  |                               :...OPTIONAL RTP padding        |
	//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// +---------------+
	// |0|1|2|3|4|5|6|7|
	// +-+-+-+-+-+-+-+-+
	// |S|E|R|  Type   |
	// +---------------+
	fuHeader := payload[1]

	if (fuHeader>>7)&1 == 1 { // 第一个分片包
		h264dp.fragments = h264dp.fragments[:0]
	}
	if len(h264dp.fragments) != 0 &&
		h264dp.fragments[len(h264dp.fragments)-1].SequenceNumber != packet.SequenceNumber-1 {
		// Packet loss ?
		h264dp.fragments = h264dp.fragments[:0]
		return
	}

	// 缓存片段
	h264dp.fragments = append(h264dp.fragments, packet)

	if (fuHeader>>6)&1 == 1 { // 最后一个片段
		frameLen := 1 // 计数帧总长,初始 naluType header len
		for _, fragment := range h264dp.fragments {
			frameLen += len(fragment.Payload()) - 2
		}

		frame := &codec.Frame{
			MediaType: codec.MediaTypeVideo,
			Payload:   make([]byte, frameLen)}

		frame.Payload[0] = (header & 0x60) | (fuHeader & 0x1F)
		offset := 1
		for _, fragment := range h264dp.fragments {
			payload := fragment.Payload()[2:]
			copy(frame.Payload[offset:], payload)
			offset += len(payload)
		}
		// 清空分片缓存
		h264dp.fragments = h264dp.fragments[:0]

		err = h264dp.writeFrame(basePts, packet.Timestamp,frame)
	}

	return
}

func (h264dp *h264Depacketizer) rtp2ntp(timestamp uint32) int64 {
	return h264dp.syncClock.Rtp2Ntp(timestamp)
}

func (h264dp *h264Depacketizer) writeFrame(basePts int64, rtpTimestamp uint32, frame *codec.Frame) error {
	nalType := frame.Payload[0] & 0x1f
	switch nalType {
	case h264.NalSps:
		if len(h264dp.meta.Sps) == 0 {
			h264dp.meta.Sps = frame.Payload
		}
	case h264.NalPps:
		if len(h264dp.meta.Pps) == 0 {
			h264dp.meta.Pps = frame.Payload
		}
	case h264.NalFillerData: // ?ignore...
		return nil
	}

	if !h264dp.metaReady {
		if !h264.MetadataIsReady(h264dp.meta) {
			return nil
		}
		if h264dp.meta.FixedFrameRate {
			h264dp.dtsStep = float64(time.Second) / h264dp.meta.FrameRate
		} else {
			h264dp.startOn = time.Now()
		}
		h264dp.metaReady = true
	}

	frame.Pts = h264dp.rtp2ntp(rtpTimestamp) - basePts+ptsDelay
	if h264dp.dtsStep > 0 {
		frame.Dts = int64(h264dp.nextDts)
		h264dp.nextDts += h264dp.dtsStep
	} else {
		frame.Dts = int64(time.Now().Sub(h264dp.startOn))
	}
	return h264dp.w.WriteFrame(frame)
}
