// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import "github.com/cnotch/queue"

// Pack 表示流媒体包
type Pack interface {
	Size() int // 包内数据的长度
}

// PackCache 媒体包缓存接口
type PackCache interface {
	CachePack(pack Pack)
	PushTo(q *queue.SyncQueue) int
	Reset()
}

type emptyCache struct {
}

func (emptyCache) CachePack(Pack)                {}
func (emptyCache) PushTo(q *queue.SyncQueue) int { return 0 }
func (emptyCache) Reset()                        {}

// NewEmptyCache .
func NewEmptyCache() PackCache {
	return emptyCache{}
}
