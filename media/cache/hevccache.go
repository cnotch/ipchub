// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"sync"

	"github.com/cnotch/ipchub/av/codec/hevc"
	"github.com/cnotch/ipchub/av/format/rtp"
	"github.com/cnotch/queue"
)

// HevcCache 画面组缓存(Group of Pictures).
type HevcCache struct {
	cacheGop bool
	l        sync.RWMutex
	gop      queue.Queue
	vps      *rtp.Packet // 视频参数集包
	sps      *rtp.Packet // 序列参数集包
	pps      *rtp.Packet // 图像参数集包
}

// NewHevcCache 创建 HEVC 缓存
func NewHevcCache(cacheGop bool) *HevcCache {
	return &HevcCache{
		cacheGop: cacheGop,
	}
}

// CachePack 向HevcCache中缓存包
func (cache *HevcCache) CachePack(pack Pack) {
	rtppack := pack.(*rtp.Packet)

	if rtppack.Channel != rtp.ChannelVideo {
		return
	}

	// 判断是否是参数和关键帧包
	vps, sps, pps, islice := cache.getPalyloadType(rtppack.Payload())

	cache.l.Lock()
	defer cache.l.Unlock()

	if vps { // 视频参数
		cache.vps = rtppack
		return
	}

	if sps { // 序列头参数
		cache.sps = rtppack
		return
	}

	if pps { // 图像参数
		cache.pps = rtppack
		return
	}

	if cache.cacheGop { // 需要缓存 GOP
		if islice { // 关键帧
			cache.gop.Reset()
			cache.gop.Push(rtppack)
		} else if cache.gop.Len() > 0 {
			cache.gop.Push(rtppack)
		}
	}
}

// Reset 重置HevcCache缓存
func (cache *HevcCache) Reset() {
	cache.l.Lock()
	defer cache.l.Unlock()

	cache.vps = nil
	cache.sps = nil
	cache.pps = nil
	cache.gop.Reset()
}

// PushTo 入列到指定的队列
func (cache *HevcCache) PushTo(q *queue.SyncQueue) int {
	bytes := 0
	cache.l.RLock()
	defer cache.l.RUnlock()

	// 写参数包
	if cache.vps != nil {
		q.Queue().Push(cache.vps)
		bytes += cache.vps.Size()
	}

	if cache.sps != nil {
		q.Queue().Push(cache.sps)
		bytes += cache.sps.Size()
	}

	if cache.pps != nil {
		q.Queue().Push(cache.pps)
		bytes += cache.pps.Size()
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

func (cache *HevcCache) getPalyloadType(payload []byte) (vps, sps, pps, islice bool) {
	if len(payload) < 3 {
		return
	}

	// +---------------+---------------+
	// |0|1|2|3|4|5|6|7|0|1|2|3|4|5|6|7|
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |F|   Type    |  LayerId  | TID |
	// +-------------+-----------------+
	naluType := (payload[0] >> 1) & 0x3f

	// 在RTP中的扩展,分片(FU)
	if naluType == hevc.NalFuInRtp {
		//  0 1 2 3 4 5 6 7
		// +-+-+-+-+-+-+-+-+
		// |S|E|  FuType   |
		// +---------------+
		naluType = payload[2] & 0x3f
		if (payload[2]>>7)&1 == 1 { // 第一个分片
			cache.nalType(naluType, &vps, &sps, &pps, &islice)
		}
		return
	}

	// 如果是原生的 HEVC NAL
	if naluType <= hevc.NalRsvNvcl47 {
		cache.nalType(naluType, &vps, &sps, &pps, &islice)
		return
	}
	return
}

func (cache *HevcCache) nalType(nalType byte, vps, sps, pps, islice *bool) {
	if nalType >= hevc.NalBlaWLp && nalType <= hevc.NalCraNut {
		*islice = true
		return
	}

	switch nalType {
	case hevc.NalVps:
		*vps = true
	case hevc.NalSps:
		*sps = true
	case hevc.NalPps:
		*pps = true
	}
	return
}
