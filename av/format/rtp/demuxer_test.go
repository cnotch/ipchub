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
	"github.com/cnotch/ipchub/av/codec/h264"
	"github.com/cnotch/ipchub/av/format/sdp"
	"github.com/cnotch/xlog"
	"github.com/stretchr/testify/assert"
)

var demuxerTestCases = []struct {
	sdpFile string
	rtpFile string
	frames  frameWriter
}{
	{"game.sdp", "game.rtp", frameWriter{1354, 1937, 11, 0, 0}},
	{"music.sdp", "music.rtp", frameWriter{1505, 2569, 9, 0, 9}},
	// {"4k.rtp", frameWriter{898, 1359, 28, 0, 27}},
}

func TestDemuxer(t *testing.T) {
	channels := []int{int(ChannelVideo), int(ChannelVideoControl), int(ChannelAudio), int(ChannelAudioControl)}
	for _, tt := range demuxerTestCases {
		t.Run(tt.rtpFile, func(t *testing.T) {
			sdpbytes, err := ioutil.ReadFile("../../../test/asserts/" + tt.sdpFile)
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

			file, err := os.Open("../../../test/asserts/" + tt.rtpFile)
			if err != nil {
				t.Error(err)
				return
			}
			defer file.Close()

			reader := bufio.NewReader(file)
			fw := &frameWriter{}
			demuxer, err := NewDemuxer(&video, &audio, fw, xlog.L())
			if err!=nil{
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
	keys        int
	sps         int
	pps         int
}

func (fw *frameWriter) WriteFrame(frame *codec.Frame) (err error) {
	if frame.MediaType == codec.MediaTypeVideo {
		fw.videoFrames++
		if h264.IsSps(frame.Payload[0]) {
			fw.sps++
		}
		if h264.IsPps(frame.Payload[0]) {
			fw.pps++
		}
		if h264.IsIdrSlice(frame.Payload[0]) {
			fw.keys++
		}
	} else {
		fw.audioFrames++
	}
	return
}
