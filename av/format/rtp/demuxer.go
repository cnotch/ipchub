// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"fmt"
	"runtime/debug"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// depacketizer 解包器
type depacketizer interface {
	Control(p *Packet) error
	Depacketize(p *Packet) error
}

// Demuxer 帧转换器
type Demuxer struct {
	closed           bool
	recvQueue        *queue.SyncQueue
	depacketizeFuncs [4]func(packet *Packet) error
	logger           *xlog.Logger
}

func emptyDepacketize(*Packet) error { return nil }

// NewDemuxer 创建 rtp.Packet 解封装处理器。
func NewDemuxer(video *codec.VideoMeta, audio *codec.AudioMeta, fw codec.FrameWriter, logger *xlog.Logger) (*Demuxer, error) {
	fc := &Demuxer{
		recvQueue: queue.NewSyncQueue(),
		closed:    false,
		logger:    logger,
	}

	var videoDepacketizer, audioDepacketizer depacketizer
	switch video.Codec {
	case "H264":
		videoDepacketizer = NewH264Depacketizer(video, fw)
	case "H265":
		videoDepacketizer = NewH265Depacketizer(video, fw)
	}
	if videoDepacketizer == nil {
		return nil, fmt.Errorf("Unsupport video codec type:%s", video.Codec)
	}

	fc.depacketizeFuncs[ChannelVideo] = videoDepacketizer.Depacketize
	fc.depacketizeFuncs[ChannelVideoControl] = videoDepacketizer.Control

	if audio.Codec == "AAC" {
		audioDepacketizer = NewAacDepacketizer(audio, fw)
	}
	if audioDepacketizer != nil {
		fc.depacketizeFuncs[ChannelAudio] = audioDepacketizer.Depacketize
		fc.depacketizeFuncs[ChannelAudioControl] = audioDepacketizer.Control
	} else {
		fc.depacketizeFuncs[ChannelAudio] = emptyDepacketize
		fc.depacketizeFuncs[ChannelAudioControl] = emptyDepacketize
	}

	go fc.process()
	return fc, nil
}

func (dm *Demuxer) process() {
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
