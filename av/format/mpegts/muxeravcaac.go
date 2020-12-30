// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mpegts

import (
	"fmt"
	"io"
	"runtime/debug"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/aac"
	"github.com/cnotch/ipchub/av/codec/h264"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// 网络播放时 PTS（Presentation Time Stamp）的延时
// 影响视频 Tag 的 CTS 和音频的 DTS（Decoding Time Stamp）
const (
	dtsDelay = 0
	ptsDelay = 1000
)

// MuxerAvcAac flv muxer from av.Frame(H264[+AAC])
type MuxerAvcAac struct {
	videoMeta     codec.VideoMeta
	audioMeta     codec.AudioMeta
	hasAudio      bool
	audioSps      aac.RawSPS
	recvQueue     *queue.SyncQueue
	tsframeWriter FrameWriter
	closed        bool
	basePts       int64
	nextDts       float64
	dtsStep       float64
	logger        *xlog.Logger // 日志对象
}

// NewMuxerAvcAac .
func NewMuxerAvcAac(videoMeta codec.VideoMeta, audioMeta codec.AudioMeta, tsframeWriter FrameWriter, logger *xlog.Logger) (*MuxerAvcAac, error) {
	muxer := &MuxerAvcAac{
		recvQueue:     queue.NewSyncQueue(),
		videoMeta:     videoMeta,
		audioMeta:     audioMeta,
		hasAudio:      audioMeta.Codec == "AAC",
		tsframeWriter: tsframeWriter,
		closed:        false,
		nextDts:       dtsDelay,
		logger:        logger,
	}

	if videoMeta.FrameRate > 0 {
		muxer.dtsStep = 1000.0 / videoMeta.FrameRate
	}

	if muxer.hasAudio {
		if err := muxer.prepareAacSps(); err != nil {
			return nil, err
		}
	}
	go muxer.process()
	return muxer, nil
}

func (muxer *MuxerAvcAac) prepareAacSps() (err error) {
	if err = muxer.audioSps.Decode(muxer.audioMeta.Sps); err != nil {
		return
	}

	if 0 == muxer.audioSps.Profile || 0x1f == muxer.audioSps.Profile {
		err = fmt.Errorf("hls decdoe audio aac sequence header failed, aac profile=%d", muxer.audioSps.Profile)
		return
	}

	// the profile = object_id + 1
	// @see aac-mp4a-format-ISO_IEC_14496-3+2001.pdf, page 78,
	//      Table 1. A.9 MPEG-2 Audio profiles and MPEG-4 Audio object types
	// so the aac_profile should plus 1, not minus 1, and nginx-rtmp used it to
	// downcast aac SSR to LC.
	muxer.audioSps.Profile--
	return
}

// WriteFrame .
func (muxer *MuxerAvcAac) WriteFrame(frame *codec.Frame) error {
	muxer.recvQueue.Push(frame)
	return nil
}

// Close .
func (muxer *MuxerAvcAac) Close() error {
	if muxer.closed {
		return nil
	}

	muxer.closed = true
	muxer.recvQueue.Signal()
	return nil
}

func (muxer *MuxerAvcAac) process() {
	defer func() {
		defer func() { // 避免 handler 再 panic
			recover()
		}()

		if r := recover(); r != nil {
			muxer.logger.Errorf("tsmuxer routine panic；r = %v \n %s", r, debug.Stack())
		}

		// 尽早通知GC，回收内存
		muxer.recvQueue.Reset()
		if closer, ok := muxer.tsframeWriter.(io.Closer); ok {
			closer.Close()
		}
	}()

	for !muxer.closed {
		f := muxer.recvQueue.Pop()
		if f == nil {
			if !muxer.closed {
				muxer.logger.Warn("tsmuxer:receive nil frame")
			}
			continue
		}

		frame := f.(*codec.Frame)
		if muxer.basePts == 0 {
			muxer.basePts = frame.AbsTimestamp
		}

		if frame.FrameType == codec.FrameVideo {
			if err := muxer.muxVideoTag(frame); err != nil {
				muxer.logger.Errorf("tsmuxer: muxVideoFrame error - %s", err.Error())
			}
		} else {
			if err := muxer.muxAudioTag(frame); err != nil {
				muxer.logger.Errorf("tsmuxer: muxAudioFrame error - %s", err.Error())
			}
		}
	}
}

func (muxer *MuxerAvcAac) muxVideoTag(frame *codec.Frame) (err error) {
	if frame.Payload[0]&0x1F == h264.NalSps {
		if len(muxer.videoMeta.Sps) == 0 {
			muxer.videoMeta.Sps = frame.Payload
		}
		var rawSps h264.RawSPS
		err = rawSps.Decode(muxer.videoMeta.Sps)
		if err != nil {
			return
		}

		muxer.videoMeta.Width = rawSps.Width()
		muxer.videoMeta.Height = rawSps.Height()
		muxer.videoMeta.FrameRate = rawSps.FrameRate()
		muxer.dtsStep = 1000.0 / muxer.videoMeta.FrameRate
		return
	}

	if frame.Payload[0]&0x1F == h264.NalPps {
		if len(muxer.videoMeta.Pps) == 0 {
			muxer.videoMeta.Pps = frame.Payload
		}
		return
	}

	dts := int64(muxer.nextDts)
	muxer.nextDts += muxer.dtsStep
	pts := frame.AbsTimestamp - muxer.basePts + ptsDelay
	if dts > pts {
		pts = dts
	}

	// set fields
	tsframe := &Frame{
		Pid:      tsVideoPid,
		StreamID: tsVideoAvc,
		Dts:      dts * 90,
		Pts:      pts * 90,
		Payload:  frame.Payload,
		key:      frame.Payload[0]&0x1F == h264.NalIdrSlice,
	}

	tsframe.prepareAvcHeader(muxer.videoMeta.Sps, muxer.videoMeta.Pps)

	return muxer.tsframeWriter.WriteMpegtsFrame(tsframe)
}

func (muxer *MuxerAvcAac) muxAudioTag(frame *codec.Frame) error {
	pts := frame.AbsTimestamp - muxer.basePts + ptsDelay
	pts *= 90

	// set fields
	tsframe := &Frame{
		Pid:      tsAudioPid,
		StreamID: tsAudioAac,
		Dts:      pts,
		Pts:      pts,
		Payload:  frame.Payload,
	}

	tsframe.prepareAacHeader(&muxer.audioSps)
	return muxer.tsframeWriter.WriteMpegtsFrame(tsframe)
}
