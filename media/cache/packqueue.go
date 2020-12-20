// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"sync"
)

// PackQueue 媒体包的队列，它并发安全
type PackQueue struct {
	cond  *sync.Cond
	packs PackBuffer
}

// NewPackQueue 创建包队列
func NewPackQueue() *PackQueue {
	return &PackQueue{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

// Buffer 返回内部缓存
func (pq *PackQueue) Buffer() *PackBuffer {
	return &pq.packs
}

// Enqueue 入列包并发送信号
func (pq *PackQueue) Enqueue(pack Pack) {
	pq.cond.L.Lock()
	pq.packs.WritePack(pack)
	pq.cond.Signal()
	pq.cond.L.Unlock()
}

// Dequeue 出列包，如果没有等待信号做一次重试
func (pq *PackQueue) Dequeue() Pack {
	var pack Pack
	pq.cond.L.Lock()

	if pq.packs.Len() <= 0 {
		pq.cond.Wait()
	}

	pack, _ = pq.packs.ReadPack()

	pq.cond.L.Unlock()

	return pack
}

// Signal 发送信号，以便结束等待
func (pq *PackQueue) Signal() {
	pq.cond.Signal()
}

// Broadcast 广播信号，以释放所有出列的阻塞等待
func (pq *PackQueue) Broadcast() {
	pq.cond.Broadcast()
}

// Len 队列长度
func (pq *PackQueue) Len() int {
	pq.cond.L.Lock()
	defer pq.cond.L.Unlock()
	return pq.packs.Len()
}

// Clear 清空队列
func (pq *PackQueue) Clear() {
	pq.cond.L.Lock()
	defer pq.cond.L.Unlock()
	pq.packs.Reset()
}
