// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"io"
	"runtime/debug"
	"time"

	"github.com/cnotch/ipchub/stats"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// Consumer 消费者接口
type Consumer interface {
	Consume(pack Pack)
	io.Closer
}

// consumption 流媒体消费者
type consumption struct {
	startOn    time.Time        // 启动时间
	stream     *Stream          // 被消费的流
	cid        CID              // 消费ID
	consumer   Consumer         // 消费者
	packetType PacketType       // 消费的包类型
	extra      string           // 消费者额外信息
	recvQueue  *queue.SyncQueue // 接收媒体源数据的队列
	closed     bool             // 消费者是否关闭
	Flow       stats.Flow       // 流量统计
	logger     *xlog.Logger     // 日志对象
}

func (c *consumption) ID() CID {
	return c.cid
}

// Close 关闭消费者
func (c *consumption) Close() error {
	if c.closed {
		return nil
	}

	c.closed = true
	c.recvQueue.Signal()
	return nil
}

// 向消费者发送媒体包
func (c *consumption) send(pack Pack) {
	c.recvQueue.Push(pack)
	c.Flow.AddIn(int64(pack.Size()))
}

// 向消费者发送一个图像组
func (c *consumption) sendGop(cache packCache) int {
	bytes := cache.PushTo(c.recvQueue)
	c.Flow.AddIn(int64(bytes))
	return bytes
}

func (c *consumption) consume() {
	defer func() {
		defer func() { // 避免 handler 再 panic
			recover()
		}()

		if r := recover(); r != nil {
			c.logger.Errorf("consume routine panic；r = %v \n %s", r, debug.Stack())
		}

		// 停止消费
		c.stream.StopConsume(c.cid)
		c.consumer.Close()

		// 尽早通知GC，回收内存
		c.recvQueue.Reset()
		c.stream = nil
	}()

	for !c.closed {
		p := c.recvQueue.Pop()
		if p == nil {
			if !c.closed {
				c.logger.Warn("receive nil pack")
			}
			continue
		}

		pack := p.(Pack)
		c.consumer.Consume(pack)
		c.Flow.AddOut(int64(pack.Size()))
	}
}

// ConsumptionInfo 消费者信息
type ConsumptionInfo struct {
	ID         uint32           `json:"id"`
	StartOn    string           `json:"start_on"`
	PacketType string           `json:"packet_type"`
	Extra      string           `json:"extra"`
	Flow       stats.FlowSample `json:"flow"` // 转换成 K
}

// Info 获取消费者信息
func (c *consumption) Info() ConsumptionInfo {
	flow := c.Flow.GetSample()
	flow.InBytes /= 1024
	flow.OutBytes /= 1024

	return ConsumptionInfo{
		ID:         uint32(c.cid),
		StartOn:    c.startOn.Format(time.RFC3339Nano),
		PacketType: c.packetType.String(),
		Extra:      c.extra,
		Flow:       flow,
	}
}
