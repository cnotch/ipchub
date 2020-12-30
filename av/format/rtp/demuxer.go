// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"runtime/debug"

	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// Depacketizer 解包器
type Depacketizer interface {
	Control(p *Packet) error
	Depacketize(p *Packet) error
}

// Demuxer 帧转换器
type Demuxer struct {
	closed       bool
	recvQueue    *queue.SyncQueue
	depacketizeFuncs [4]func(packet *Packet) error
	logger       *xlog.Logger
}

func emptyDepacketize(*Packet) error { return nil }

// NewDemuxer 创建 rtp.Packet 解封装处理器。
func NewDemuxer(videoDepacketizer Depacketizer, audioDepacketizer Depacketizer, logger *xlog.Logger) *Demuxer {
	fc := &Demuxer{
		recvQueue: queue.NewSyncQueue(),
		closed:    false,
		logger:    logger,
	}

	if videoDepacketizer != nil {
		fc.depacketizeFuncs[ChannelVideo] = videoDepacketizer.Depacketize
		fc.depacketizeFuncs[ChannelVideoControl] = videoDepacketizer.Control
	} else {
		fc.depacketizeFuncs[ChannelVideo] = emptyDepacketize
		fc.depacketizeFuncs[ChannelVideoControl] = emptyDepacketize
	}

	if audioDepacketizer != nil {
		fc.depacketizeFuncs[ChannelAudio] = audioDepacketizer.Depacketize
		fc.depacketizeFuncs[ChannelAudioControl] = audioDepacketizer.Control
	} else {
		fc.depacketizeFuncs[ChannelAudio] = emptyDepacketize
		fc.depacketizeFuncs[ChannelAudioControl] = emptyDepacketize
	}

	go fc.convert()
	return fc
}

func (dm *Demuxer) convert() {
	defer func() {
		defer func() { // 避免 handler 再 panic
			recover()
		}()

		if r := recover(); r != nil {
			dm.logger.Errorf("FrameConverter routine panic；r = %v \n %s", r, debug.Stack())
		}

		// 尽早通知GC，回收内存
		dm.recvQueue.Reset()
	}()

	for !dm.closed {
		p := dm.recvQueue.Pop()
		if p == nil {
			if !dm.closed {
				dm.logger.Warn("FrameConverter:receive nil packet")
			}
			continue
		}

		packet := p.(*Packet)
		if err := dm.depacketizeFuncs[int(packet.Channel)](packet); err != nil {
			dm.logger.Errorf("FrameConverter: extract rtp frame error :%s", err.Error())
			// break
		}
	}
}

// Close .
func (dm *Demuxer) Close() error {
	if dm.closed {
		return nil
	}

	dm.closed = true
	dm.recvQueue.Signal()
	return nil
}

// WritePacket .
func (fc *Demuxer) WriteRtpPacket(packet *Packet) error {
	fc.recvQueue.Push(packet)
	return nil
}
