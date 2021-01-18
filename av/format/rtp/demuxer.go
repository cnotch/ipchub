// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// 网络播放时 PTS（Presentation Time Stamp）的延时
const (
	ptsDelay = int64(time.Second)
)

// Depacketizer 解包器
type Depacketizer interface {
	Control(basePts *int64, p *Packet) error
	Depacketize(basePts int64, p *Packet) error
}

type emptyDepacketizer struct{}

func (emptyDepacketizer) Control(basePts *int64, p *Packet) error    { return nil }
func (emptyDepacketizer) Depacketize(basePts int64, p *Packet) error { return nil }

// Demuxer 帧转换器
type Demuxer struct {
	closed    bool
	recvQueue *queue.SyncQueue
	vdp       Depacketizer
	adp       Depacketizer
	logger    *xlog.Logger
}

func emptyDepacketize(*int64, *Packet) error { return nil }

// NewDemuxer 创建 rtp.Packet 解封装处理器。
func NewDemuxer(video *codec.VideoMeta, audio *codec.AudioMeta, fw codec.FrameWriter, logger *xlog.Logger) (*Demuxer, error) {
	demuxer := &Demuxer{
		recvQueue: queue.NewSyncQueue(),
		closed:    false,
		logger:    logger,
	}

	switch video.Codec {
	case "H264":
		demuxer.vdp = NewH264Depacketizer(video, fw)
	case "H265":
		demuxer.vdp = NewH265Depacketizer(video, fw)
	default:
		return nil, fmt.Errorf("rtp demuxer unsupport video codec type:%s", video.Codec)
	}
	if audio.Codec == "AAC" {
		demuxer.adp = NewAacDepacketizer(audio, fw)
	} else {
		demuxer.adp = emptyDepacketizer{}
	}

	go demuxer.process()
	return demuxer, nil
}

func (demuxer *Demuxer) process() {
	defer func() {
		defer func() { // 避免 handler 再 panic
			recover()
		}()

		if r := recover(); r != nil {
			demuxer.logger.Errorf("FrameConverter routine panic；r = %v \n %s", r, debug.Stack())
		}

		// 尽早通知GC，回收内存
		demuxer.recvQueue.Reset()
	}()

	var basePts int64
	for !demuxer.closed {
		p := demuxer.recvQueue.Pop()
		if p == nil {
			if !demuxer.closed {
				demuxer.logger.Warn("FrameConverter:receive nil packet")
			}
			continue
		}

		packet := p.(*Packet)
		var err error
		switch packet.Channel {
		case ChannelVideo:
			err = demuxer.vdp.Depacketize(basePts, packet)
		case ChannelVideoControl:
			err = demuxer.vdp.Control(&basePts, packet)
		case ChannelAudio:
			err = demuxer.adp.Depacketize(basePts, packet)
		case ChannelAudioControl:
			err = demuxer.adp.Control(&basePts, packet)
		}

		if err != nil {
			demuxer.logger.Errorf("rtp demuxer: depackeetize rtp frame error :%s", err.Error())
			// break
		}
	}
}

// Close .
func (demuxer *Demuxer) Close() error {
	if demuxer.closed {
		return nil
	}

	demuxer.closed = true
	demuxer.recvQueue.Signal()
	return nil
}

// WritePacket .
func (demuxer *Demuxer) WriteRtpPacket(packet *Packet) error {
	demuxer.recvQueue.Push(packet)
	return nil
}
