// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hls

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/cnotch/ipchub/protos/mpegts"
	"github.com/cnotch/ipchub/utils/murmur"
	"github.com/cnotch/xlog"
)

// drop the segment when duration of ts too small.
const hlsSegmentMinDurationMs = 100

// in ms, for HLS aac flush the audio
const hlsAacDelay = 100
const remainSegmets = 3

// Muxer the HLS stream(m3u8 and ts files).
type Muxer struct {
	path        string // 流路径
	hlsFragment int    // 每个片段长度

	// m3u8 segments
	l           sync.RWMutex
	segments    []*segment
	memory      bool   // 使用内存存储缓存到硬盘
	segmentPath string // 缓存路径

	sequenceNo int      // 片段序号
	current    *segment //current segment

	// last http access time
	lastAccessTime time.Time
	logger         *xlog.Logger

	audioRate   int
	afCache     *mpegts.Frame // audio frame cache
	afCacheBuff bytes.Buffer
	// time jitter for aac
	aacJitter *hlsAacJitter
}

// NewMuxer .
func NewMuxer(path string, hlsFragment int, segmentPath string, audioRate int, logger *xlog.Logger) (*Muxer, error) {
	muxer := &Muxer{
		path:           path,
		hlsFragment:    hlsFragment,
		memory:         segmentPath == "",
		segmentPath:    segmentPath,
		logger:         logger,
		lastAccessTime: time.Now(),
		sequenceNo:     0,
		audioRate:      audioRate,
		aacJitter:      newHlsAacJitter(),
	}

	if err := muxer.segmentOpen(0); err != nil {
		return nil, err
	}

	// set the current segment to sequence header,
	// when close the segement, it will write a discontinuity to m3u8 file.
	muxer.current.isSequenceHeader = true
	return muxer, nil
}

// open a new segment, a new ts file
// segmentStartDts use to calc the segment duration, use 0 for the first segment of hls
func (muxer *Muxer) segmentOpen(segmentStartDts int64) (err error) {
	if nil != muxer.current {
		// has already opened, ignore segment open
		return
	}

	// new segment
	muxer.sequenceNo++
	curr := newSegment(muxer.memory)
	curr.sequenceNo = muxer.sequenceNo
	curr.segmentStartPts = segmentStartDts
	curr.uri = "/streams" + muxer.path + "/" + strconv.Itoa(muxer.sequenceNo) + ".ts"

	tsFileName := fmt.Sprintf("%d_%d.ts", murmur.OfString(muxer.path), curr.sequenceNo)
	tsFilePath := filepath.Join(muxer.segmentPath, tsFileName)
	if err = curr.file.open(tsFilePath); err != nil {
		return
	}

	muxer.current = curr
	return
}

// WriteMpegtsFrame implements mpegts.FrameWriter
func (muxer *Muxer) WriteMpegtsFrame(frame *mpegts.Frame) (err error) {
	// if current is NULL, segment is not open, ignore the flush event.
	if nil == muxer.current {
		return
	}
	if len(frame.Payload) <= 0 {
		return
	}

	if frame.IsAudio() {
		if muxer.afCache == nil {
			pts := muxer.aacJitter.onBufferStart(frame.Pts, muxer.audioRate)
			headerFrame := *frame
			headerFrame.Dts = pts
			headerFrame.Pts = pts
			muxer.afCache = &headerFrame
			muxer.afCacheBuff.Write(frame.Payload)
		} else {
			muxer.afCacheBuff.Write(frame.Header)
			muxer.afCacheBuff.Write(frame.Payload)
			muxer.aacJitter.onBufferContinue()
		}

		if frame.Pts-muxer.afCache.Pts > hlsAacDelay*90 {
			return muxer.flushAudioCache()
		}

		// reap when current source is pure audio.
		// it maybe changed when stream info changed,
		// for example, pure audio when start, audio/video when publishing,
		// pure audio again for audio disabled.
		// so we reap event when the audio incoming when segment overflow.
		// we use absolutely overflow of segment to make jwplayer/ffplay happy
		if muxer.isSegmentAbsolutelyOverflow() {
			if err = muxer.reapSegment(frame.Pts); err != nil {
				return
			}
		}
		return
	}

	if frame.IsKeyFrame() && muxer.isSegmentOverflow() {
		if err = muxer.reapSegment(frame.Pts); err != nil {
			return
		}
	}

	// flush video when got one
	if err = muxer.flushFrame(frame); err != nil {
		return
	}
	return
}

func (muxer *Muxer) flushAudioCache() (err error) {
	if muxer.afCache == nil {
		return
	}

	muxer.afCache.Payload = muxer.afCacheBuff.Bytes()
	err = muxer.flushFrame(muxer.afCache)
	muxer.afCache = nil
	muxer.afCacheBuff.Reset()
	return
}

func (muxer *Muxer) flushFrame(frame *mpegts.Frame) (err error) {
	muxer.current.updateDuration(frame.Pts)
	if err = muxer.current.file.writeFrame(frame); err != nil {
		return
	}
	return
}

// close segment(ts)
func (muxer *Muxer) segmentClose(muxerClosed bool) (err error) {
	if nil == muxer.current {
		return
	}

	muxer.l.Lock()
	defer muxer.l.Unlock()

	muxer.current.file.close()
	remain := remainSegmets
	if muxerClosed {
		remain = 0
	}

	// valid, add to segments if segment duration is ok
	if muxer.current.duration*1000 >= hlsSegmentMinDurationMs {
		muxer.segments = append(muxer.segments, muxer.current)
		muxer.current = nil
	} else {
		// reuse current segment index
		muxer.sequenceNo--
		muxer.current.file.delete()
	}

	// 仅保留3个
	if len(muxer.segments) > remain {
		for i := 0; i < len(muxer.segments)-remain; i++ {
			muxer.segments[i].file.delete()
			muxer.segments[i] = nil
		}
		copy(muxer.segments[:remain], muxer.segments[len(muxer.segments)-remain:])
		muxer.segments = muxer.segments[:remain]
	}

	return
}

// reopen the muxer for a new hls segment,
// close current segment, open a new segment,
// then write the key frame to the new segment.
// so, user must reap_segment then flush_video to hls muxer.
func (muxer *Muxer) reapSegment(segmentStartDts int64) (err error) {
	if err = muxer.segmentClose(false); err != nil {
		return
	}

	if err = muxer.segmentOpen(segmentStartDts); err != nil {
		return
	}

	// segment open, flush the audio.
	// @see: ngx_rtmp_hls_open_fragment
	/* start fragment with audio to make iPhone happy */
	err = muxer.flushAudioCache()

	return
}

// whether segment overflow,
// that is whether the current segment duration>=(the segment in config)
func (muxer *Muxer) isSegmentOverflow() bool {
	return muxer.current.duration >= float64(muxer.hlsFragment)
}

// whether segment absolutely overflow, for pure audio to reap segment,
// that is whether the current segment duration>=2*(the segment in config)
func (muxer *Muxer) isSegmentAbsolutelyOverflow() bool {
	if nil == muxer.current {
		return true
	}

	res := muxer.current.duration >= float64(2*muxer.hlsFragment)

	return res
}

var m3u8Pool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 512))
	},
}

// M3u8 获取 m3u8 播放列表
func (muxer *Muxer) M3u8() ([]byte, error) {
	muxer.lastAccessTime = time.Now()
	w := m3u8Pool.Get().(*bytes.Buffer)
	w.Reset()
	defer m3u8Pool.Put(w)

	muxer.l.RLock()
	defer muxer.l.RUnlock()
	segments := muxer.segments

	if len(segments) < remainSegmets {
		return nil, errors.New("Playlist is not enough,Maybe the HLS stream just started")
	}

	seq := segments[0].sequenceNo
	var maxDuration float64
	for _, seg := range segments {
		if seg.duration > maxDuration {
			maxDuration = seg.duration
		}
	}
	duration := int32(maxDuration + 1)
	// 描述部分
	fmt.Fprintf(w,
		"#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-ALLOW-CACHE:NO\n#EXT-X-TARGETDURATION:%d\n#EXT-X-MEDIA-SEQUENCE:%d\n\n",
		duration, seq)

	// 列表部分
	for _, seg := range segments {
		if seg.isSequenceHeader {
			// #EXT-X-DISCONTINUITY\n
			fmt.Fprint(w, "#EXT-X-DISCONTINUITY\n")
		}

		fmt.Fprintf(w, "#EXTINF:%.3f,\n%s\n",
			seg.duration,
			seg.uri)
	}

	return w.Bytes(), nil
}

// Segment 获取 segment
func (muxer *Muxer) Segment(seq int) (io.Reader, int, error) {
	muxer.lastAccessTime = time.Now()
	muxer.l.RLock()
	defer muxer.l.RUnlock()

	for _, seg := range muxer.segments {
		if seg.sequenceNo == seq {
			return seg.file.get()
		}
	}
	return nil, 0, errors.New("Not found TSFile")
}

// LastAccessTime 最后hls访问时间
func (muxer *Muxer) LastAccessTime() time.Time {
	return muxer.lastAccessTime
}

// Close .
func (muxer *Muxer) Close() error {
	muxer.flushAudioCache()

	return muxer.segmentClose(true)
}
