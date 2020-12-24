// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"runtime/debug"

	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// FrameConverter 帧转换器
type FrameConverter struct {
	closed       bool
	recvQueue    *queue.SyncQueue
	extractFuncs [4]func(packet *Packet) error
	logger       *xlog.Logger
}

func emptyExtract(*Packet) error { return nil }

// NewFrameConverter 创建 Packet 到 Frame 的转换器。
func NewFrameConverter(videoExtractor FrameExtractor, audioExtractor FrameExtractor, logger *xlog.Logger) *FrameConverter {
	fc := &FrameConverter{
		recvQueue: queue.NewSyncQueue(),
		closed:    false,
		logger:    logger,
	}

	if videoExtractor != nil {
		fc.extractFuncs[ChannelVideo] = videoExtractor.Extract
		fc.extractFuncs[ChannelVideoControl] = videoExtractor.Control
	} else {
		fc.extractFuncs[ChannelVideo] = emptyExtract
		fc.extractFuncs[ChannelVideoControl] = emptyExtract
	}

	if audioExtractor != nil {
		fc.extractFuncs[ChannelAudio] = audioExtractor.Extract
		fc.extractFuncs[ChannelAudioControl] = audioExtractor.Control
	} else {
		fc.extractFuncs[ChannelAudio] = emptyExtract
		fc.extractFuncs[ChannelAudioControl] = emptyExtract
	}

	go fc.convert()
	return fc
}

func (fc *FrameConverter) convert() {
	defer func() {
		defer func() { // 避免 handler 再 panic
			recover()
		}()

		if r := recover(); r != nil {
			fc.logger.Errorf("FrameConverter routine panic；r = %v \n %s", r, debug.Stack())
		}

		// 尽早通知GC，回收内存
		fc.recvQueue.Reset()
	}()

	for !fc.closed {
		p := fc.recvQueue.Pop()
		if p == nil {
			if !fc.closed {
				fc.logger.Warn("FrameConverter:receive nil packet")
			}
			continue
		}

		packet := p.(*Packet)
		if err := fc.extractFuncs[int(packet.Channel)](packet); err != nil {
			fc.logger.Errorf("FrameConverter: extract rtp frame error :%s", err.Error())
			// break
		}
	}
}

// Close .
func (fc *FrameConverter) Close() error {
	if fc.closed {
		return nil
	}

	fc.closed = true
	fc.recvQueue.Signal()
	return nil
}

// WritePacket .
func (fc *FrameConverter) WritePacket(packet *Packet) error {
	fc.recvQueue.Push(packet)
	return nil
}
