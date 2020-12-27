// Copyright calabashdad. https://github.com/calabashdad/seal.git
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hls

// the wrapper of m3u8 segment from specification:
// 3.3.2.  EXTINF
// The EXTINF tag specifies the duration of a media segment.
type segment struct {
	// duration in seconds in m3u8.
	duration float64
	// sequence number in m3u8.
	sequenceNo int
	// ts uri in m3u8.
	uri string
	// ts full file to write.
	// fullPath string
	// the file to write ts.
	file segmentFile
	// current segment start pts for m3u8
	segmentStartPts int64
	// whether current segement is sequence header.
	isSequenceHeader bool
}

func newSegment(memory bool) *segment {
	seg := &segment{}
	if memory {
		seg.file = newMemorySegmentFile()
	} else {
		seg.file = newPersistentSegmentFile()
	}
	return seg
}

func (seg *segment) updateDuration(currentFramePts int64) {

	// we use video/audio to update segment duration,
	// so when reap segment, some previous audio frame will
	// update the segment duration, which is nagetive,
	// just ignore it.
	if currentFramePts < seg.segmentStartPts {
		return
	}

	seg.duration = float64(currentFramePts-seg.segmentStartPts) / 90000.0
}
