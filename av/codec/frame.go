// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package codec

import (
	"fmt"
	"strings"
)

// MediaType 媒体类型
type MediaType int

// 媒体类型常量
const (
	MediaTypeUnknown MediaType = iota - 1 // Usually treated as MediaTypeData
	MediaTypeVideo
	MediaTypeAudio
	MediaTypeData // Opaque data information usually continuous
	MediaTypeSubtitle
	MediaTypeAttachment // Opaque data information usually sparse
	MediaTypeNB
)

// String returns a lower-case ASCII representation of the media type.
func (mt MediaType) String() string {
	switch mt {
	case MediaTypeVideo:
		return "video"
	case MediaTypeAudio:
		return "audio"
	case MediaTypeData:
		return "data"
	case MediaTypeSubtitle:
		return "subtitle"
	case MediaTypeAttachment:
		return "attachment"
	default:
		return ""
	}
}

// MarshalText marshals the MediaType to text.
func (mt *MediaType) MarshalText() ([]byte, error) {
	return []byte(mt.String()), nil
}

// UnmarshalText unmarshals text to a MediaType.
func (mt *MediaType) UnmarshalText(text []byte) error {
	if !mt.unmarshalText(string(text)) {
		return fmt.Errorf("unrecognized media type: %q", text)
	}
	return nil
}

func (mt *MediaType) unmarshalText(text string) bool {
	switch strings.ToLower(text) {
	case "video":
		*mt = MediaTypeVideo
	case "audio":
		*mt = MediaTypeAudio
	case "data":
		*mt = MediaTypeData
	case "subtitle":
		*mt = MediaTypeSubtitle
	case "attachment":
		*mt = MediaTypeAttachment
	default:
		return false
	}
	return true
}

// Frame 音视频完整帧
type Frame struct {
	MediaType        // 媒体类型
	Dts       int64  // DTS，单位为 ns
	Pts       int64  // PTS，单位为 ns
	Payload   []byte // 媒体数据载荷
}

// FrameWriter 包装 WriteFrame 方法的接口
type FrameWriter interface {
	WriteFrame(frame *Frame) error
}
