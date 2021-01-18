// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mpegts

import (
	"fmt"
	"runtime/debug"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// Packetizer 封包器
type Packetizer interface {
	Packetize(frame *codec.Frame) error
}

type emptyPacketizer struct{}

func (emptyPacketizer) Packetize(frame *codec.Frame) error { return nil }

// Muxer mpegts muxer from av.Frame(H264[+AAC])
type Muxer struct {
	recvQueue *queue.SyncQueue
	closed    bool
	logger    *xlog.Logger // 日志对象
}

// NewMuxer .
func NewMuxer(videoMeta *codec.VideoMeta, audioMeta *codec.AudioMeta, tsframeWriter FrameWriter, logger *xlog.Logger) (*Muxer, error) {
	muxer := &Muxer{
		recvQueue: queue.NewSyncQueue(),
		closed:    false,
		logger:    logger,
	}
	var vp Packetizer = emptyPacketizer{}
	var ap Packetizer = emptyPacketizer{}

	switch videoMeta.Codec {
	case "H264":
		vp = NewH264Packetizer(videoMeta, tsframeWriter)
	default:
		return nil, fmt.Errorf("ts muxer unsupport video codec type:%s", videoMeta.Codec)
	}

	switch audioMeta.Codec {
	case "AAC":
		ap = NewAacPacketizer(audioMeta, tsframeWriter)
	default:
		return nil, fmt.Errorf("ts muxer unsupport audio codec type:%s", videoMeta.Codec)
	}

	go muxer.process(vp, ap)
	return muxer, nil
}

// WriteFrame .
func (muxer *Muxer) WriteFrame(frame *codec.Frame) error {
	muxer.recvQueue.Push(frame)
	return nil
}

// Close .
func (muxer *Muxer) Close() error {
	if muxer.closed {
		return nil
	}

	muxer.closed = true
	muxer.recvQueue.Signal()
	return nil
}

func (muxer *Muxer) process(vp, ap Packetizer) {
	defer func() {
		defer func() { // 避免 handler 再 panic
			recover()
		}()

		if r := recover(); r != nil {
			muxer.logger.Errorf("ts muxer routine panic；r = %v \n %s", r, debug.Stack())
		}

		// 尽早通知GC，回收内存
		muxer.recvQueue.Reset()
	}()

	for !muxer.closed {
		f := muxer.recvQueue.Pop()
		if f == nil {
			if !muxer.closed {
				muxer.logger.Warn("tsmuxer: receive nil frame")
			}
			continue
		}

		frame := f.(*codec.Frame)

		switch frame.MediaType {
		case codec.MediaTypeVideo:
			if err := vp.Packetize(frame); err != nil {
				muxer.logger.Errorf("tsmuxer: muxVideoTag error - %s", err.Error())
			}
		case codec.MediaTypeAudio:
			if err := ap.Packetize(frame); err != nil {
				muxer.logger.Errorf("tsmuxer: muxAudioTag error - %s", err.Error())
			}
		default:
		}
	}
}
