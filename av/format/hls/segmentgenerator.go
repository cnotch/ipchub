// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hls

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/cnotch/ipchub/av/format/mpegts"
	"github.com/cnotch/ipchub/utils/murmur"
	"github.com/cnotch/xlog"
)

// drop the segment when duration of ts too small.
const hlsSegmentMinDurationMs = 100

// in ms, for HLS aac flush the audio
const hlsAacDelay = 100

// SegmentGenerator generate the HLS ts segment.
type SegmentGenerator struct {
	playlist    *Playlist // 播放列表
	path        string    // 流路径
	hlsFragment int       // 每个片段长度

	memory      bool   // 使用内存存储缓存到硬盘
	segmentPath string // 缓存文件路径

	sequenceNo int      // 片段序号
	current    *segment //current segment

	logger *xlog.Logger

	audioRate   int
	afCache     *mpegts.Frame // audio frame cache
	afCacheBuff bytes.Buffer
	// time jitter for aac
	aacJitter *hlsAacJitter
}

// NewSegmentGenerator .
func NewSegmentGenerator(playlist *Playlist, path string, hlsFragment int, segmentPath string, audioRate int, logger *xlog.Logger) (*SegmentGenerator, error) {
	sg := &SegmentGenerator{
		playlist:    playlist,
		path:        path,
		hlsFragment: hlsFragment,
		memory:      segmentPath == "",
		segmentPath: segmentPath,
		logger:      logger,
		sequenceNo:  0,
		audioRate:   audioRate,
		aacJitter:   newHlsAacJitter(),
	}

	if err := sg.segmentOpen(0); err != nil {
		return nil, err
	}

	// set the current segment to sequence header,
	// when close the segement, it will write a discontinuity to m3u8 file.
	sg.current.isSequenceHeader = true
	return sg, nil
}

// open a new segment, a new ts file
// segmentStartDts use to calc the segment duration, use 0 for the first segment of hls
func (sg *SegmentGenerator) segmentOpen(segmentStartDts int64) (err error) {
	if nil != sg.current {
		// has already opened, ignore segment open
		return
	}

	// new segment
	sg.sequenceNo++
	curr := newSegment(sg.memory)
	curr.sequenceNo = sg.sequenceNo
	curr.segmentStartPts = segmentStartDts
	curr.uri = "/streams" + sg.path + "/" + strconv.Itoa(sg.sequenceNo) + ".ts"

	tsFileName := fmt.Sprintf("%d_%d.ts", murmur.OfString(sg.path), curr.sequenceNo)
	tsFilePath := filepath.Join(sg.segmentPath, tsFileName)
	if err = curr.file.open(tsFilePath); err != nil {
		return
	}

	sg.current = curr
	return
}

// WriteMpegtsFrame implements mpegts.FrameWriter
func (sg *SegmentGenerator) WriteMpegtsFrame(frame *mpegts.Frame) (err error) {
	// if current is NULL, segment is not open, ignore the flush event.
	if nil == sg.current {
		return
	}
	if len(frame.Payload) <= 0 {
		return
	}

	if frame.IsAudio() {
		if sg.afCache == nil {
			pts := sg.aacJitter.onBufferStart(frame.Pts, sg.audioRate)
			headerFrame := *frame
			headerFrame.Dts = pts
			headerFrame.Pts = pts
			sg.afCache = &headerFrame
			sg.afCacheBuff.Write(frame.Payload)
		} else {
			sg.afCacheBuff.Write(frame.Header)
			sg.afCacheBuff.Write(frame.Payload)
			sg.aacJitter.onBufferContinue()
		}

		if frame.Pts-sg.afCache.Pts > hlsAacDelay*90 {
			return sg.flushAudioCache()
		}

		// reap when current source is pure audio.
		// it maybe changed when stream info changed,
		// for example, pure audio when start, audio/video when publishing,
		// pure audio again for audio disabled.
		// so we reap event when the audio incoming when segment overflow.
		// we use absolutely overflow of segment to make jwplayer/ffplay happy
		if sg.isSegmentAbsolutelyOverflow() {
			if err = sg.reapSegment(frame.Pts); err != nil {
				return
			}
		}
		return
	}

	if frame.IsKeyFrame() && sg.isSegmentOverflow() {
		if err = sg.reapSegment(frame.Pts); err != nil {
			return
		}
	}

	// flush video when got one
	if err = sg.flushFrame(frame); err != nil {
		return
	}
	return
}

func (sg *SegmentGenerator) flushAudioCache() (err error) {
	if sg.afCache == nil {
		return
	}

	sg.afCache.Payload = sg.afCacheBuff.Bytes()
	err = sg.flushFrame(sg.afCache)
	sg.afCache = nil
	sg.afCacheBuff.Reset()
	return
}

func (sg *SegmentGenerator) flushFrame(frame *mpegts.Frame) (err error) {
	sg.current.updateDuration(frame.Pts)
	if err = sg.current.file.writeFrame(frame); err != nil {
		return
	}
	return
}

// close segment(ts)
func (sg *SegmentGenerator) segmentClose() (err error) {
	if nil == sg.current {
		return
	}

	curr := sg.current
	sg.current = nil
	curr.file.close()
	if curr.duration*1000 < hlsSegmentMinDurationMs {
		// reuse current segment index
		sg.sequenceNo--
		curr.file.delete()
	} else {
		sg.playlist.addSegment(curr)
	}
	return
}

// reopen the sg for a new hls segment,
// close current segment, open a new segment,
// then write the key frame to the new segment.
// so, user must reap_segment then flush_video to hls sg.
func (sg *SegmentGenerator) reapSegment(segmentStartDts int64) (err error) {
	if err = sg.segmentClose(); err != nil {
		return
	}

	if err = sg.segmentOpen(segmentStartDts); err != nil {
		return
	}

	// segment open, flush the audio.
	// @see: ngx_rtmp_hls_open_fragment
	/* start fragment with audio to make iPhone happy */
	err = sg.flushAudioCache()

	return
}

// whether segment overflow,
// that is whether the current segment duration>=(the segment in config)
func (sg *SegmentGenerator) isSegmentOverflow() bool {
	return sg.current.duration >= float64(sg.hlsFragment)
}

// whether segment absolutely overflow, for pure audio to reap segment,
// that is whether the current segment duration>=2*(the segment in config)
func (sg *SegmentGenerator) isSegmentAbsolutelyOverflow() bool {
	if nil == sg.current {
		return true
	}

	res := sg.current.duration >= float64(2*sg.hlsFragment)

	return res
}

// Close .
func (sg *SegmentGenerator) Close() error {
	if nil == sg.current {
		return nil
	}

	curr := sg.current
	sg.current = nil
	curr.file.close()
	curr.file.delete()
	return nil
}
