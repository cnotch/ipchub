// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hls

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cnotch/scheduler"
)

const hlsRemainSegments = 3

// Playlist the HLS playlist(m3u8 and ts files).
type Playlist struct {
	// m3u8 segments
	l        sync.RWMutex
	segments []*segment

	// last http access time
	lastAccessTime int64
}

// NewPlaylist .
func NewPlaylist() *Playlist {
	return &Playlist{
		lastAccessTime: time.Now().UnixNano(),
	}

}

var m3u8Pool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 512))
	},
}

// M3u8 获取 m3u8 播放列表
func (pl *Playlist) M3u8(token string) ([]byte, error) {
	atomic.StoreInt64(&pl.lastAccessTime, time.Now().UnixNano())
	w := m3u8Pool.Get().(*bytes.Buffer)
	w.Reset()
	defer m3u8Pool.Put(w)

	pl.l.RLock()
	defer pl.l.RUnlock()
	segments := pl.segments

	if len(segments) < hlsRemainSegments {
		return nil, errors.New("playlist is not enough,maybe the HLS stream just started")
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

		if len(token) > 0 {
			fmt.Fprintf(w, "#EXTINF:%.3f,\n%s?token=%s\n",
				seg.duration,
				seg.uri, token)
		} else {
			fmt.Fprintf(w, "#EXTINF:%.3f,\n%s\n",
				seg.duration,
				seg.uri)
		}
	}

	return w.Bytes(), nil
}

// Segment 获取 segment
func (pl *Playlist) Segment(seq int) (io.Reader, int, error) {
	atomic.StoreInt64(&pl.lastAccessTime, time.Now().UnixNano())
	pl.l.RLock()
	defer pl.l.RUnlock()

	for _, seg := range pl.segments {
		if seg.sequenceNo == seq {
			return seg.file.get()
		}
	}
	return nil, 0, errors.New("Not found TSFile")
}

// LastAccessTime 最后hls访问时间
func (pl *Playlist) LastAccessTime() time.Time {
	lastAccessTime := atomic.LoadInt64(&pl.lastAccessTime)
	return time.Unix(0, lastAccessTime)
}

// Close .
func (pl *Playlist) Close() error {
	pl.l.Lock()
	defer pl.l.Unlock()
	pl.clearSegments(0)

	return nil
}

func (pl *Playlist) addSegment(seg *segment) {
	pl.l.Lock()
	defer pl.l.Unlock()
	pl.segments = append(pl.segments, seg)

	pl.clearSegments(hlsRemainSegments)
}

func (pl *Playlist) clearSegments(remain int) {
	if len(pl.segments) > remain {
		for i := 0; i < len(pl.segments)-remain; i++ {
			if err := pl.segments[i].file.delete(); err != nil {
				// 延时异步删除
				file := pl.segments[i].file
				duration := time.Duration(pl.segments[i].duration * float64(time.Second))
				uri := pl.segments[i].uri
				scheduler.AfterFunc(duration, func() {
					file.delete()
				}, "delete "+uri)
			}
			pl.segments[i] = nil
		}
		copy(pl.segments[:remain], pl.segments[len(pl.segments)-remain:])
		pl.segments = pl.segments[:remain]
	}
	return
}
