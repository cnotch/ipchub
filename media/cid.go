// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"sync/atomic"
)

// PacketType 消费媒体包类型
type PacketType uint32

// 预定义消费媒体包类型
const (
	RTPPacket PacketType = iota // 根据 RTP 协议打包的媒体
	FLVPacket

	maxConsumerSequence = 0x3fff_ffff
)

// CID consumer ID
// type(2bits)+sequence(30bits)
type CID uint32

// String 类型的字串表示
func (t PacketType) String() string {
	switch t {
	case RTPPacket:
		return "RTP"
	case FLVPacket:
		return "FLV"
	default:
		return "Unknown"
	}
}

// NewCID 创建新的流消费ID
func NewCID(packetType PacketType, consumerSequenceSeed *uint32) CID {
	localid := atomic.AddUint32(consumerSequenceSeed, 1)
	if localid >= maxConsumerSequence {
		localid = 1
		atomic.StoreUint32(consumerSequenceSeed, localid)
	}
	return CID(packetType<<30) | CID(localid&maxConsumerSequence)
}

// Type 获取消费者类型
func (id CID) Type() PacketType {
	return PacketType((id >> 30) & 0x3)
}

// Sequence 获取消费者序号
func (id CID) Sequence() uint32 {
	return uint32(id & CID(maxConsumerSequence))
}
