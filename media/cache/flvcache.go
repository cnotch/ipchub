// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"sync"

	"github.com/cnotch/ipchub/protos/flv"
	"github.com/cnotch/queue"
)

// FlvCache Flv包缓存.
type FlvCache struct {
	cacheGop bool
	l        sync.RWMutex
	gop      queue.Queue
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
	if tag.IsH264SequenceHeader() {
		cache.videoSequenceHeader = tag
		return
	}
	if tag.IsAACSequenceHeader() {
		cache.audioSequenceHeader = tag
		return
	}

	if cache.cacheGop { // 如果启用 FlvCache
		if tag.IsH264KeyFrame() { // 关键帧，重置GOP
			cache.gop.Reset()
			cache.gop.Push(pack)
		} else if cache.gop.Len() > 0 { // 必须关键帧作为cache的第一个包
			cache.gop.Push(pack)
		}
	}
}

// Reset 重置FlvCache缓存
func (cache *FlvCache) Reset() {
	cache.l.Lock()
	defer cache.l.Unlock()
	cache.gop.Reset()
	cache.metaData = nil
	cache.videoSequenceHeader = nil
	cache.audioSequenceHeader = nil
}

// PushTo 入列到指定的队列
func (cache *FlvCache) PushTo(q *queue.SyncQueue) int {
	cache.l.RLock()
	defer cache.l.RUnlock()

	bytes := 0

	gop := cache.gop.Elems()
	initTimestamp := uint32(0)
	if len(gop) > 0 {
		tag := gop[0].(*flv.Tag)
		initTimestamp = tag.Timestamp
	}

	//write meta data
	if nil != cache.metaData {
		metaData := *cache.metaData
		metaData.Timestamp = initTimestamp
		q.Queue().Push(&metaData)
		bytes += metaData.Size()
	}

	//write video data
	if nil != cache.videoSequenceHeader {
		videoSequenceHeader := *cache.videoSequenceHeader
		videoSequenceHeader.Timestamp = initTimestamp
		q.Queue().Push(&videoSequenceHeader)
		bytes += videoSequenceHeader.Size()
	}

	//write audio data
	if nil != cache.audioSequenceHeader {
		audioSequenceHeader := *cache.audioSequenceHeader
		audioSequenceHeader.Timestamp = initTimestamp
		q.Queue().Push(&audioSequenceHeader)
		bytes += audioSequenceHeader.Size()
	}

	// write gop
	q.Queue().PushN(gop) // 启动阶段调用，无需加锁
	for _, p := range gop {
		bytes += p.(Pack).Size()
	}

	return bytes
}
