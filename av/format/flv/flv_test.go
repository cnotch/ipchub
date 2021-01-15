// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/format/rtp"
	"github.com/cnotch/ipchub/av/format/sdp"
	"github.com/cnotch/xlog"
)

var muxerTestCases = []struct {
	sdpFile string
	rtpFile string
	flvFile string
}{
	{"game.sdp", "game.rtp", "game.flv"},
	{"music.sdp", "music.rtp", "music.flv"},
	{"265.sdp", "265.rtp", "265.flv"},
}

func TestFlvWriter(t *testing.T) {
	assertsPath := "../../../test/asserts/"

	for _, tt := range muxerTestCases {
		t.Run(tt.rtpFile, func(t *testing.T) {

			sdpraw, err := ioutil.ReadFile(assertsPath + tt.sdpFile)
			if err != nil {
				panic("Couldn't open sdp")
			}

			file, err := os.Open(assertsPath + tt.rtpFile)
			if err != nil {
				panic("Couldn't open rtp")
			}
			defer file.Close()
			reader := bufio.NewReader(file)

			out, err := os.OpenFile(assertsPath+tt.flvFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
			if err != nil {
				panic("Couldn't open flv")
			}
			defer out.Close()
			var video codec.VideoMeta
			var audio codec.AudioMeta
			sdp.ParseMetadata(string(sdpraw), &video, &audio)
			writer, err := NewWriter(out, 5)
			flvMuxer, _ := NewMuxer(&video, &audio, writer, xlog.L())

			rtpDemuxer, _ := rtp.NewDemuxer(&video, &audio, flvMuxer, xlog.L())
			channels := []int{int(rtp.ChannelVideo), int(rtp.ChannelVideoControl), int(rtp.ChannelAudio), int(rtp.ChannelAudioControl)}
			for {
				packet, err := rtp.ReadPacket(reader, channels)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Logf("read packet error :%s", err.Error())
				}
				rtpDemuxer.WriteRtpPacket(packet)
			}

			<-time.After(time.Millisecond * 1000)
			rtpDemuxer.Close()
			flvMuxer.Close()
		})
	}
}
