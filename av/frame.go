// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package av

// 帧类型
const (
	FrameVideo = byte(iota)
	FrameAudio
)

// Frame 音视频完整帧
type Frame struct {
	FrameType    byte   // 帧类型
	AbsTimestamp int64  // 绝对时间戳(主要用于表示 pts)，单位为 ms 的 UNIX 时间
	Payload      []byte // 媒体数据载荷
}

// FrameWriter 包装 WriteFrame 方法的接口
type FrameWriter interface {
	WriteFrame(frame *Frame) error
}
