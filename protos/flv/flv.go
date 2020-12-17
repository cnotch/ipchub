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
	FlvHeaderSize  = 9
	TypeFlagsVideo = 0x04
	TypeFlagsAudio = 0x01
)

var flvHeaderTemplate = []byte{0x46, 0x4c, 0x56, 0x01, 0x00, 0x00, 0x00, 0x00, 0x09}

// Reader flv Reader
type Reader struct {
	r               io.Reader
	header          [FlvHeaderSize]byte // flv header
	previousTagSize uint32
}

// NewReader .
func NewReader(r io.Reader) (*Reader, error) {
	reader := &Reader{
		r:               r,
		previousTagSize: 0,
	}

	// read flv header
	if _, err := io.ReadFull(r, reader.header[:]); err != nil {
		return nil, err
	}
	// 简单验证
	if reader.header[0] != 'F' ||
		reader.header[1] != 'L' ||
		reader.header[2] != 'V' {
		return nil, errors.New("Signatures must is 'FLV'")
	}

	return reader, nil
}

// Read read flv tag
func (r *Reader) Read() (*Tag, error) {
	// read PreviousTagSize
	var previousTagSize uint32
	var buff [4]byte
	if _, err := io.ReadFull(r.r, buff[:]); err != nil {
		return nil, err
	}

	// 验证 PreviousTagSize
	previousTagSize = binary.BigEndian.Uint32(buff[:])
	if previousTagSize != r.previousTagSize {
		return nil, fmt.Errorf("PreviousTagSize mismatches, expect '%d' but  actual '%d'",
			r.previousTagSize, previousTagSize)
	}

	var tag Tag
	if err := tag.Read(r.r); err != nil {
		return nil, err
	}

	// save PreviousTagSize
	r.previousTagSize = uint32(tag.Size())
	return &tag, nil
}

// HasVideo flv include video stream.
func (r *Reader) HasVideo() bool {
	return r.header[5]&TypeFlagsVideo != 0
}

// HasAudio flv include audio stream.
func (r *Reader) HasAudio() bool {
	return r.header[5]&TypeFlagsAudio != 0
}

// Writer flv Writer
type Writer struct {
	w              io.Writer
	previousTagLen uint32
}

// NewWriter .
func NewWriter(w io.Writer, typeFlags byte) (*Writer, error) {
	if typeFlags&0x05 == 0 {
		return nil, errors.New("TypeFlags not include any streams")
	}

	writer := &Writer{
		w:              w,
		previousTagLen: 0,
	}

	var flvHeader [FlvHeaderSize]byte
	copy(flvHeader[:], flvHeaderTemplate[:])
	flvHeader[5] = typeFlags & (TypeFlagsVideo | TypeFlagsAudio)
	if _, err := w.Write(flvHeader[:]); err != nil {
		return nil, err
	}

	return writer, nil
}

// Write write flv tag
func (w *Writer) Write(tag *Tag) error {
	var buff [4]byte
	// write PreviousTagSize
	binary.BigEndian.PutUint32(buff[0:], w.previousTagLen)
	if _, err := w.w.Write(buff[:]); err != nil {
		return err
	}

	if err := tag.Write(w.w); err != nil {
		return err
	}
	w.previousTagLen = uint32(tag.Size())
	return nil
}
