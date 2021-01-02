// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"bufio"
	"io"
	"os"
	"testing"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/h264"
	"github.com/stretchr/testify/assert"
)

var demuxerTestCases = []struct {
	rtpFile string
	frames  frameWriter
}{
	{"game.rtp", frameWriter{1354, 1937, 11, 0, 0}},
	{"music.rtp", frameWriter{1505, 2569, 9, 0, 9}},
	// {"4k.rtp", frameWriter{898, 1359, 28, 0, 27}},
}

func TestDemuxer(t *testing.T) {
	channels := []int{int(ChannelVideo), int(ChannelVideoControl), int(ChannelAudio), int(ChannelAudioControl)}
	for _, tt := range demuxerTestCases {
		t.Run(tt.rtpFile, func(t *testing.T) {
			file, err := os.Open("../../../test/asserts/" + tt.rtpFile)
			if err != nil {
				t.Error(err)
				return
			}
			defer file.Close()

			reader := bufio.NewReader(file)
			fw := &frameWriter{}
			h264dp := NewH264Depacketizer(fw)
			aacdp := NewAacDepacketizer(fw, 44100)
			for {
				packet, err := ReadPacket(reader, channels)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("read packet error :%s", err.Error())
					break
				}
				switch packet.Channel {
				case ChannelAudio:
					if err := aacdp.Depacketize(packet); err != nil {
						t.Errorf("depacketiz aac error :%s", err.Error())
					}
				case ChannelVideo:
					if err := h264dp.Depacketize(packet); err != nil {
						t.Errorf("depacketiz h264 error :%s", err.Error())
					}
				case ChannelVideoControl:
					h264dp.Control(packet)
				case ChannelAudioControl:
					aacdp.Control(packet)
				}
			}
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
	if frame.FrameType == codec.FrameVideo {
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
