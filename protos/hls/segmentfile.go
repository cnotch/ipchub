// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hls

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/cnotch/ipchub/protos/mpegts"
)

type segmentFile interface {
	open(path string) error
	close() error
	writeFrame(frame *mpegts.Frame) error
	get() (io.Reader, error)
	delete() error
}

type memorySegmentFile struct {
	buff *bytes.Buffer
	w    mpegts.FrameWriter
}

func newMemorySegmentFile() segmentFile {
	return &memorySegmentFile{
		buff: bytes.NewBuffer(nil),
	}
}

func (mf *memorySegmentFile) open(path string) (err error) {
	mf.buff.Reset()
	mf.w, err = mpegts.NewWriter(mf.buff)
	return
}

func (mf *memorySegmentFile) writeFrame(frame *mpegts.Frame) (err error) {
	return mf.w.WriteMpegtsFrame(frame)
}

func (mf *memorySegmentFile) close() (err error) {
	mf.w = nil
	return
}

func (mf *memorySegmentFile) get() (io.Reader, error) {
	return bytes.NewReader(mf.buff.Bytes()), nil
}

func (mf *memorySegmentFile) delete() error {
	return nil
}

type persistentSegmentFile struct {
	path string
	file *os.File
	buff *bufio.Writer
	w    mpegts.FrameWriter
}

func newPersistentSegmentFile() segmentFile {
	return &persistentSegmentFile{}
}

func (pf *persistentSegmentFile) open(path string) (err error) {
	pf.path = path
	pf.file, err = os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("open file error, file=%s", path)
	}

	pf.buff = bufio.NewWriterSize(pf.file, 64*1024)
	pf.w, err = mpegts.NewWriter(pf.buff)
	return
}

func (pf *persistentSegmentFile) writeFrame(frame *mpegts.Frame) (err error) {
	return pf.w.WriteMpegtsFrame(frame)
}

func (pf *persistentSegmentFile) close() (err error) {
	if nil == pf.file {
		return
	}

	pf.buff.Flush()
	pf.file.Close()

	// after close, rest the file write to nil
	pf.file = nil
	pf.buff = nil
	pf.w = nil
	return nil
}

func (pf *persistentSegmentFile) get() (reader io.Reader, err error) {
	var f *os.File
	if f, err = os.Open(pf.path); err != nil {
		return nil, err
	}

	return f, nil
}

func (pf *persistentSegmentFile) delete() error {
	return os.Remove(pf.path)
}
