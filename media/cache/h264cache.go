// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"sync"

	"github.com/cnotch/ipchub/av/h264"
	"github.com/cnotch/ipchub/protos/rtp"
	"github.com/cnotch/queue"
)

// H264Cache 画面组缓存(Group of Pictures).
type H264Cache struct {
	cacheGop bool
	l        sync.RWMutex
	gop      queue.Queue
	spsPack  Pack // 序列参数集包
	ppsPack  Pack // 图像参数集包
}

// NewH264Cache 创建 H264 缓存
func NewH264Cache(cacheGop bool) PackCache {
	return &H264Cache{
		cacheGop: cacheGop,
	}
}

// CachePack 向H264Cache中缓存包
func (cache *H264Cache) CachePack(pack Pack) {
	rtppack := pack.(*rtp.Packet)

	if rtppack.Channel != rtp.ChannelVideo {
		return
	}

	// 判断是否是参数和关键帧包
	sps, pps, islice := cache.getPalyloadType(rtppack.Payload())

	cache.l.Lock()
	defer cache.l.Unlock()

	if sps { // 新序列参数,重置图像参数和 GopCache
		cache.spsPack = pack
		return
	}

	if pps { // 新图像参数，重置 GopCahce
		cache.ppsPack = pack
		return
	}

	if cache.cacheGop { // 需要缓存 GOP
		if islice { // 关键帧
			cache.gop.Reset()
			cache.gop.Push(pack)
		} else if cache.gop.Len() > 0 { // 必须关键帧作为cache的第一个包
			cache.gop.Push(pack)
		}
	}
}

// Reset 重置H264Cache缓存
func (cache *H264Cache) Reset() {
	cache.l.Lock()
	defer cache.l.Unlock()

	cache.spsPack = nil
	cache.ppsPack = nil
	cache.gop.Reset()
}

// PushTo 入列到指定的队列
func (cache *H264Cache) PushTo(q *queue.SyncQueue) int {
	bytes := 0
	cache.l.RLock()
	defer cache.l.RUnlock()

	// 写参数包
	if cache.spsPack != nil {
		q.Queue().Push(cache.spsPack)
		bytes += cache.spsPack.Size()
	}
	if cache.ppsPack != nil {
		q.Queue().Push(cache.ppsPack)
		bytes += cache.ppsPack.Size()
	}

	// 如果必要，写 GopCache
	if cache.cacheGop {
		packs := cache.gop.Elems()
		q.Queue().PushN(packs) // 启动阶段调用，无需加锁
		for _, p := range packs {
			bytes += p.(Pack).Size()
		}
	}

	return bytes
}

func (cache *H264Cache) getPalyloadType(payload []byte) (sps, pps, islice bool) {
	if len(payload) < 3 {
		return
	}

	// +---------------+
	// |0|1|2|3|4|5|6|7|
	// +-+-+-+-+-+-+-+-+
	// |F|NRI|  Type   |
	// +---------------+
	naluTypeInRtp := payload[0] & 0x1F
	switch naluTypeInRtp {
	case h264.NalStapaInRtp, h264.NalStapbInRtp, h264.NalMtap16InRtp, h264.NalMtap24InRtp:
		// 组合包
		// 	0                   1                   2                   3
		// 	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//  |                          RTP Header                           |
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
		off := 1
		// 循环读取被封装的NAL
		for {
			// nal长度
			nalSize := ((uint16(payload[off])) << 8) | uint16(payload[off+1])
			if nalSize < 1 {
				return
			}

			off += 2
			realNALU := byte(payload[off] & 0x1f)
			cache.nalType(realNALU, &sps, &pps, &islice) // 当前NAL类型
			off += int(nalSize)

			if off >= len(payload) { // 扫描完成
				break
			}
		}
	case h264.NalFuAInRtp, h264.NalFuBInRtp:
		// 分片包
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
		if (fuHeader>>7)&1 == 1 { // 仅对第一个分片进行检测
			// Start
			realNALU := byte(fuHeader & 0x1f)
			cache.nalType(realNALU, &sps, &pps, &islice)
		}
	default:
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
		cache.nalType(naluTypeInRtp, &sps, &pps, &islice)
	}
	return
}

func (cache *H264Cache) nalType(nalType byte, sps, pps, islice *bool) {
	switch nalType {
	case h264.NalSps:
		*sps = true
	case h264.NalPps:
		*pps = true
	case h264.NalIdrSlice:
		*islice = true
	}
	return
}
