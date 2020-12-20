// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"sync"

	"github.com/cnotch/ipchub/protos/flv"
)

// FlvCache Flv包缓存.
type FlvCache struct {
	cacheGop bool
	l        sync.RWMutex
	gop      PackBuffer
	// cached meta data
	metaData *flv.Tag
	// cached video sequence header
	videoSequenceHeader *flv.Tag
	// cached aideo sequence header
	audioSequenceHeader *flv.Tag
}

// NewFlvCache 创建FlvCache实例
func NewFlvCache(cacheGop bool) PackCache {
	return &FlvCache{
		cacheGop: cacheGop,
	}
}

// CachePack 向FlvCache中缓存包
func (cache *FlvCache) CachePack(pack Pack) {
	tag := pack.(*flv.Tag)

	cache.l.Lock()
	defer cache.l.Unlock()

	if tag.IsMetadata() {
		cache.metaData = tag
		return
	}
	if flv.VideoH264IsSequenceHeader(tag.Data) {
		cache.videoSequenceHeader = tag
		return
	}
	if flv.AudioIsSequenceHeader(tag.Data) {
		cache.audioSequenceHeader = tag
		return
	}

	if cache.cacheGop { // 如果启用 FlvCache
		if flv.VideoH264IsKeyframe(tag.Data) { // 关键帧，重置GOP
			cache.gop.Reset()
			cache.gop.WritePack(pack)
		} else if cache.gop.Len() > 0 { // 必须关键帧作为cache的第一个包
			cache.gop.WritePack(pack)
		}
	}
}

// Reset 重置FlvCache缓存
func (cache *FlvCache) Reset() {
	cache.l.Lock()
	cache.gop.Reset()
	cache.metaData = nil
	cache.videoSequenceHeader = nil
	cache.audioSequenceHeader = nil
	cache.l.Unlock()
}

// EnqueueTo 入列到指定的队列
func (cache *FlvCache) EnqueueTo(q *PackQueue) int {
	bytes := 0
	cache.l.RLock()

	gop := cache.gop.Packs()
	if len(gop) > 0 {
		tag := gop[0].(*flv.Tag)
		// 更新 Timestamp
		if cache.metaData != nil {
			cache.metaData.Timestamp = tag.Timestamp
		}
		if cache.videoSequenceHeader != nil {
			cache.videoSequenceHeader.Timestamp = tag.Timestamp
		}
		if cache.audioSequenceHeader != nil {
			cache.audioSequenceHeader.Timestamp = tag.Timestamp
		}
	}

	//write meta data
	if nil != cache.metaData {
		q.Buffer().WritePack(cache.metaData)
		bytes += cache.metaData.Size()
	}

	//write video data
	if nil != cache.videoSequenceHeader {
		q.Buffer().WritePack(cache.videoSequenceHeader)
		bytes += cache.videoSequenceHeader.Size()
	}

	//write audio data
	if nil != cache.audioSequenceHeader {
		q.Buffer().WritePack(cache.audioSequenceHeader)
		bytes += cache.audioSequenceHeader.Size()
	}

	// write gop
	q.Buffer().Write(gop) // 启动阶段调用，无需加锁
	for _, p := range gop {
		bytes += p.Size()
	}

	cache.l.RUnlock()
	return bytes
}
