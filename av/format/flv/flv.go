// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// flv Header Size, total is 9Byte.
// 	Signatures	3Byte	'FLV' = 0x46 0x4c 0x56
// 	Version		1Byte	0x01
// 	TypeFlags	1Byte 	bit0:audio bit2:video
// 	DataOffset	4Byte	FLV Header Length
const (
	FlvHeaderSize   = 9
	TypeFlagsVideo  = 0x04
	TypeFlagsAudio  = 0x01
	TypeFlagsOffset = 4
)

var flvHeaderTemplate = []byte{0x46, 0x4c, 0x56, 0x01, 0x00, 0x00, 0x00, 0x00, 0x09}

// Reader flv Reader
type Reader struct {
	r      io.Reader
	Header [FlvHeaderSize]byte // flv header
}

// NewReader .
func NewReader(r io.Reader) (*Reader, error) {
	reader := &Reader{
		r: r,
	}

	// read flv header
	if _, err := io.ReadFull(r, reader.Header[:]); err != nil {
		return nil, err
	}
	// 简单验证
	if reader.Header[0] != 'F' ||
		reader.Header[1] != 'L' ||
		reader.Header[2] != 'V' {
		return nil, errors.New("Signatures must is 'FLV'")
	}

	if previousTagSize, err := reader.readTagSize(); err != nil {
		return nil, err
	} else if previousTagSize != 0 {
		return nil, errors.New("First 'PreviousTagSize' must is  0")
	}

	return reader, nil
}

func (r *Reader) readTagSize() (tagSize uint32, err error) {
	var buff [4]byte
	if _, err := io.ReadFull(r.r, buff[:]); err != nil {
		return 0, err
	}

	tagSize = binary.BigEndian.Uint32(buff[:])
	return
}

// ReadFlvTag read flv tag
func (r *Reader) ReadFlvTag() (*Tag, error) {

	var tag Tag
	if err := tag.Read(r.r); err != nil {
		return nil, err
	}

	if tagSize, err := r.readTagSize(); err != nil {
		return nil, err
	} else if tagSize != uint32(tag.Size()) {
		return nil, fmt.Errorf("PreviousTagSize mismatches, expect '%d' but  actual '%d'",
			tag.Size(), tagSize)
	}
	return &tag, nil
}

// HasVideo flv include video stream.
func (r *Reader) HasVideo() bool {
	return r.Header[TypeFlagsOffset]&TypeFlagsVideo != 0
}

// HasAudio flv include audio stream.
func (r *Reader) HasAudio() bool {
	return r.Header[TypeFlagsOffset]&TypeFlagsAudio != 0
}

const uninitializedTimestampDelta = 0xffffffff

// Writer flv Writer
type Writer struct {
	w              io.Writer
	timestampDelta uint32 // 流在中间输出时的相对时间戳
}

// NewWriter .
func NewWriter(w io.Writer, typeFlags byte) (*Writer, error) {
	if typeFlags&0x05 == 0 {
		return nil, errors.New("TypeFlags not include any streams")
	}

	writer := &Writer{
		w:              w,
		timestampDelta: uninitializedTimestampDelta,
	}

	var flvHeader [FlvHeaderSize]byte
	copy(flvHeader[:], flvHeaderTemplate[:])
	flvHeader[TypeFlagsOffset] = typeFlags & (TypeFlagsVideo | TypeFlagsAudio)
	if _, err := w.Write(flvHeader[:]); err != nil {
		return nil, err
	}

	if err := writer.writeTagSize(0); err != nil {
		return nil, err
	}

	return writer, nil
}

func (w *Writer) writeTagSize(tagSize uint32) error {
	var buff [4]byte
	// write PreviousTagSize
	binary.BigEndian.PutUint32(buff[:], tagSize)
	if _, err := w.w.Write(buff[:]); err != nil {
		return err
	}
	return nil
}

// WriteFlvTag write flv tag
func (w *Writer) WriteFlvTag(tag *Tag) error {
	// 记录第一个Tag的时间戳
	if w.timestampDelta == uninitializedTimestampDelta {
		w.timestampDelta = tag.Timestamp
	}

	if err := writeTag(w.w, tag, w.timestampDelta); err != nil {
		return err
	}

	return w.writeTagSize(uint32(tag.Size()))
}
