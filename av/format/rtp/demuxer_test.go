// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/format/sdp"
	"github.com/cnotch/xlog"
	"github.com/stretchr/testify/assert"
)

var demuxerTestCases = []struct {
	sdpFile string
	rtpFile string
	frames  frameWriter
}{
	{"game.sdp", "game.rtp", frameWriter{1354, 1937}},
	{"music.sdp", "music.rtp", frameWriter{1505, 2569}},
	{"265.sdp", "265.rtp", frameWriter{5828, 11395}},
	// {"4k.rtp", frameWriter{898, 1359, 28, 0, 27}},
}

func TestDemuxer(t *testing.T) {
	assertsPath := "../../../test/asserts/"
	channels := []int{int(ChannelVideo), int(ChannelVideoControl), int(ChannelAudio), int(ChannelAudioControl)}
	for _, tt := range demuxerTestCases {
		t.Run(tt.rtpFile, func(t *testing.T) {
			sdpbytes, err := ioutil.ReadFile(assertsPath + tt.sdpFile)
			if err != nil {
				t.Error(err)
				return
			}
			var video codec.VideoMeta
			var audio codec.AudioMeta
			err = sdp.ParseMetadata(string(sdpbytes), &video, &audio)
			if err != nil {
				t.Error(err)
				return
			}

			file, err := os.Open(assertsPath + tt.rtpFile)
			if err != nil {
				t.Error(err)
				return
			}
			defer file.Close()

			reader := bufio.NewReader(file)
			fw := &frameWriter{}
			demuxer, err := NewDemuxer(&video, &audio, fw, xlog.L())
			if err != nil {
				t.Error(err)
			}
			defer demuxer.Close()

			for {
				packet, err := ReadPacket(reader, channels)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("read packet error :%s", err.Error())
					break
				}
				demuxer.WriteRtpPacket(packet)
			}
			<-time.After(time.Second)

			assert.Equal(t, tt.frames, *fw)
		})
	}
}

type frameWriter struct {
	videoFrames int
	audioFrames int
}

func (fw *frameWriter) WriteFrame(frame *codec.Frame) (err error) {
	dts := frame.Dts / int64(time.Millisecond)
	pts := frame.Pts / int64(time.Millisecond)
	_ = dts
	_ = pts
	if frame.MediaType == codec.MediaTypeVideo {
		fw.videoFrames++
	} else {
		fw.audioFrames++
	}
	return
}
