// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mpegts

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/cnotch/ipchub/av"
	"github.com/cnotch/ipchub/av/aac"
	"github.com/cnotch/ipchub/av/h264"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// 网络播放时 PTS（Presentation Time Stamp）的延时
// 影响视频 Tag 的 CTS 和音频的 DTS（Decoding Time Stamp）
const ptsDelay = 1000

// MuxerAvcAac flv muxer from av.Frame(H264[+AAC])
type MuxerAvcAac struct {
	videoMeta     av.VideoMeta
	audioMeta     av.AudioMeta
	hasAudio      bool
	audioSps      aac.RawSPS
	recvQueue     *queue.SyncQueue
	tsframeWriter FrameWriter
	closed        bool
	basePts       int64
	baseDts       int64
	logger        *xlog.Logger // 日志对象
}

// NewMuxerAvcAac .
func NewMuxerAvcAac(videoMeta av.VideoMeta, audioMeta av.AudioMeta, tsframeWriter FrameWriter, logger *xlog.Logger) (*MuxerAvcAac, error) {
	muxer := &MuxerAvcAac{
		recvQueue:     queue.NewSyncQueue(),
		videoMeta:     videoMeta,
		audioMeta:     audioMeta,
		hasAudio:      audioMeta.Codec == "AAC",
		tsframeWriter: tsframeWriter,
		closed:        false,
		baseDts:       time.Now().UnixNano() / int64(time.Millisecond),
		logger:        logger,
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
func (muxer *MuxerAvcAac) WriteFrame(frame *av.Frame) error {
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
	}()

	for !muxer.closed {
		f := muxer.recvQueue.Pop()
		if f == nil {
			if !muxer.closed {
				muxer.logger.Warn("tsmuxer:receive nil frame")
			}
			continue
		}

		frame := f.(*av.Frame)
		if muxer.basePts == 0 {
			muxer.basePts = frame.AbsTimestamp
		}

		if frame.FrameType == av.FrameVideo {
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

func (muxer *MuxerAvcAac) muxVideoTag(frame *av.Frame) (err error) {
	if frame.Payload[0]&0x1F == h264.NalSps {
		if len(muxer.videoMeta.Sps) == 0 {
			muxer.videoMeta.Sps = frame.Payload
		}
		return
	}

	if frame.Payload[0]&0x1F == h264.NalPps {
		if len(muxer.videoMeta.Pps) == 0 {
			muxer.videoMeta.Pps = frame.Payload
		}
		return
	}

	dts := time.Now().UnixNano()/int64(time.Millisecond) - muxer.baseDts
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

func (muxer *MuxerAvcAac) muxAudioTag(frame *av.Frame) error {
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
