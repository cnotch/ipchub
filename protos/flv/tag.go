// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"encoding/binary"
	"io"
)

// flv 标记类型ID
const (
	TagTypeAudio    = 0x08
	TagTypeVideo    = 0x09
	TagTypeAmf0Data = 0x12 // 18
)

// flv Tag Header Size, total is 11Byte.
// 	filter + type 1Byte
// 	data size 	3Byte
// 	timestamp 	3Byte
// 	timestampEx 1Byte
// 	streamID 	3Byte always is 0
const (
	TagHeaderSize = 11
)

// Tag FLV Tag
type Tag struct {
	Reserved  byte   // 2 bits; 用于 FMS 的保留字段值为 0
	Filter    byte   // 1 bits; 未加密文件中此值为 0，加密文件中此值为 1
	TagType   byte   // 5 bits
	DataSize  uint32 // 24 bits; Tag 中 Data 长度
	Timestamp uint32 // 24 bits(Timestamp) + 8 bits(TimestampExtended); 单位是毫秒的时间戳，FLV 文件中第一个 Tag 的 DTS 总为 0
	StreamID  uint32 // 24 bits; 总为 0
	Data      []byte // Tag 包含的数据
}

// Size tag 的总大小（包括 Header + Data）
func (tag Tag) Size() int {
	return TagHeaderSize + len(tag.Data)
}

// Read 根据规范的格式从 r 中读取 flv Tag。
func (tag *Tag) Read(r io.Reader) error {
	var tagHeader [TagHeaderSize]byte
	if _, err := io.ReadFull(r, tagHeader[:]); err != nil {
		return err
	}

	offset := 0

	// filter & type
	tag.Filter = (tagHeader[offset] << 2) >> 7
	tag.TagType = tagHeader[offset] & 0x1F
	offset++

	// data size
	tag.DataSize = binary.BigEndian.Uint32(tagHeader[offset:])
	tag.DataSize = tag.DataSize >> 8
	offset += 3

	// timestamp & timestamp extended
	timestamp := binary.BigEndian.Uint32(tagHeader[offset:])
	tag.Timestamp = (timestamp >> 8) | (timestamp << 24)
	offset += 3 // 为 stream id 多留一个高位字节

	// stream id
	tag.StreamID = binary.BigEndian.Uint32(tagHeader[offset:]) & 0xffffff

	tag.Data = make([]byte, tag.DataSize)
	if _, err := io.ReadFull(r, tag.Data); err != nil {
		return err
	}
	return nil
}

// Write 根据规范将 flv Tag 输出到 w。
func (tag *Tag) Write(w io.Writer) error {
	var tagHeader [TagHeaderSize + 1]byte  // 为 stream id 多留一个高位字节
	offset := 0

	// data size
	binary.BigEndian.PutUint32(tagHeader[offset:], uint32(len(tag.Data)))

	// type
	tagHeader[offset] = ((tag.Filter & 0x1) << 5) | (tag.TagType & 0x1f)
	offset += 4

	// timestamp
	binary.BigEndian.PutUint32(tagHeader[offset:], (tag.Timestamp<<8)|(tag.Timestamp>>24))
	offset += 4

	// stream id
	binary.BigEndian.PutUint32(tagHeader[offset:], tag.StreamID<<8)
	offset += 3

	// write tag header
	if _, err := w.Write(tagHeader[:offset]); err != nil {
		return err
	}

	// write tag data
	if _, err := w.Write(tag.Data); err != nil {
		return err
	}

	return nil
}

// TagData tag data
type TagData interface {
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
}
