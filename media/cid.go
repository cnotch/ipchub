// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"sync/atomic"
)

// ConsumerType 消费者类型
type ConsumerType uint32

// 预定义消费者类型
const (
	RTPConsumer ConsumerType = iota // 根据 RTP 协议打包的媒体
	FLVConsumer

	maxConsumerSequence = 0x3fff_ffff
)

// CID consumer ID
// type(2bits)+sequence(30bits)
type CID uint32

// String 类型的字串表示
func (t ConsumerType) String() string {
	switch t {
	case RTPConsumer:
		return "RTP"
	case FLVConsumer:
		return "FLV"
	default:
		return "Unknown"
	}
}

// NewCID 创建新的流消费ID
func NewCID(consumerType ConsumerType, consumerSequenceSeed *uint32) CID {
	localid := atomic.AddUint32(consumerSequenceSeed, 1)
	if localid >= maxConsumerSequence {
		localid = 1
		atomic.StoreUint32(consumerSequenceSeed, localid)
	}
	return CID(consumerType<<30) | CID(localid&maxConsumerSequence)
}

// Type 获取消费者类型
func (id CID) Type() ConsumerType {
	return ConsumerType((id >> 30) & 0x3)
}

// Sequence 获取消费者序号
func (id CID) Sequence() uint32 {
	return uint32(id & CID(maxConsumerSequence))
}
