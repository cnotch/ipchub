// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

// 帧类型
const (
	FrameVideo = byte(iota)
	FrameAudio
)

// Frame 音视频帧
type Frame struct {
	FrameType byte   // 帧类型
	Timestamp uint32 // rtp 包中的时间戳
	Payload   []byte // 媒体数据载荷
}

// FrameWriter 包装 WriteFrame 方法的接口
type FrameWriter interface {
	WriteFrame(frame *Frame) error
}

// FrameExtractor 帧提取器
type FrameExtractor interface {
	Extract(p *Packet) error
}

// CreateFrameExtractor 帧提取器创建方式
type CreateFrameExtractor func(w FrameWriter) FrameExtractor
