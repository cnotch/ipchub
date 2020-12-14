// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"fmt"

	"github.com/cnotch/ipchub/av/h264"
)

type h264FrameExtractor struct {
	fragments []*Packet // 分片包
	w         FrameWriter
}

// NewH264FrameExtractor 实例化 H264 帧提取器
func NewH264FrameExtractor(w FrameWriter) FrameExtractor {
	return &h264FrameExtractor{
		make([]*Packet, 0, 16),
		w,
	}
}

func (fe *h264FrameExtractor) Extract(packet *Packet) (err error) {
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
		err = fe.w.Write(&Frame{FrameVideo, packet.Timestamp, payload})
	case naluType == h264.NalStapaInRtp:
		err = fe.extractStapa(packet)
	case naluType == h264.NalFuAInRtp:
		err = fe.extractFuA(packet)
	default:
		err = fmt.Errorf("nalu type %d is currently not handled", naluType)
	}
	return
}

func (fe *h264FrameExtractor) extractStapa(packet *Packet) (err error) {
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
		frame := &Frame{
			FrameVideo,
			packet.Timestamp,
			make([]byte, nalSize),
		}
		copy(frame.Payload, payload[off:])
		frame.Payload[0] = 0 | (header & 0x60) | (frame.Payload[0] & 0x1F)
		if err = fe.w.Write(frame); err != nil {
			return
		}

		off += int(nalSize)
		if off >= len(payload) { // 扫描完成
			break
		}
	}
	return
}

func (fe *h264FrameExtractor) extractFuA(packet *Packet) (err error) {
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
		fe.fragments = fe.fragments[:0]
	}
	if len(fe.fragments) != 0 &&
		fe.fragments[len(fe.fragments)-1].SequenceNumber != packet.SequenceNumber-1 {
		// Packet loss ?
		fe.fragments = fe.fragments[:0]
		return
	}

	// 缓存片段
	fe.fragments = append(fe.fragments, packet)

	if (fuHeader>>6)&1 == 1 { // 最后一个片段
		frameLen := 1 // 计数帧总长,初始 naluType header len
		for _, fragment := range fe.fragments {
			frameLen += len(fragment.Payload()) - 2
		}

		frame := &Frame{
			FrameType: FrameVideo,
			Timestamp: fe.fragments[0].Timestamp,
			Payload:   make([]byte, frameLen)}

		frame.Payload[0] = 0 | (header & 0x60) | (fuHeader & 0x1F)
		offset := 1
		for _, fragment := range fe.fragments {
			payload := fragment.Payload()[2:]
			copy(frame.Payload[1:], payload)
			offset += len(payload)
		}
		// 清空分片缓存
		fe.fragments = fe.fragments[:0]

		err = fe.w.Write(frame)
	}

	return
}
