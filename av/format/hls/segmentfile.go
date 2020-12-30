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
	"sync"

	"github.com/cnotch/ipchub/av/format/mpegts"
)

type segmentFile interface {
	open(path string) error
	close() error
	writeFrame(frame *mpegts.Frame) error
	get() (io.Reader, int, error)
	delete() error
}

var segmentPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 512*1024))
	},
}

type memorySegmentFile struct {
	file *bytes.Buffer
	w    mpegts.FrameWriter
}

func newMemorySegmentFile() segmentFile {
	return &memorySegmentFile{}
}

func (mf *memorySegmentFile) open(path string) (err error) {
	mf.file = segmentPool.Get().(*bytes.Buffer)
	mf.file.Reset()
	mf.w, err = mpegts.NewWriter(mf.file)
	return
}

func (mf *memorySegmentFile) writeFrame(frame *mpegts.Frame) (err error) {
	return mf.w.WriteMpegtsFrame(frame)
}

func (mf *memorySegmentFile) close() (err error) {
	mf.w = nil
	return
}

func (mf *memorySegmentFile) get() (io.Reader, int, error) {
	data := mf.file.Bytes()
	return bytes.NewReader(data), len(data), nil
}

func (mf *memorySegmentFile) delete() error {
	if mf.file != nil {
		segmentPool.Put(mf.file)
		mf.file = nil
	}
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

func (pf *persistentSegmentFile) get() (reader io.Reader, size int, err error) {
	var finfo os.FileInfo
	finfo, err = os.Stat(pf.path)
	if err != nil {
		return
	}
	var f *os.File
	if f, err = os.Open(pf.path); err != nil {
		return nil, 0, err
	}

	return f, int(finfo.Size()), nil
}

func (pf *persistentSegmentFile) delete() error {
	pf.close()
	return os.Remove(pf.path)
}
