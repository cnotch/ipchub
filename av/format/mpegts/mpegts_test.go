// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mpegts

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

func TestMpegtsWriter(t *testing.T) {
	sdpraw, err := ioutil.ReadFile("../../../test/asserts/music.sdp")
	if err != nil {
		panic("Couldn't open sdp")
	}

	file, err := os.Open("../../../test/asserts/music.rtp")
	if err != nil {
		panic("Couldn't open rtp")
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	out, err := os.OpenFile("music.ts", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic("Couldn't open ts")
	}
	defer out.Close()
	var video codec.VideoMeta
	var audio codec.AudioMeta
	sdp.ParseMetadata(string(sdpraw), &video, &audio)
	writer, err := NewWriter(out)
	tsMuxer, _ := NewMuxerAvcAac(video, audio, writer, xlog.L())

	h264Depack := rtp.NewH264Depacketizer(tsMuxer)
	mpesDepack := rtp.NewAacDepacketizer(tsMuxer, audio.SampleRate)
	rtpDemuxer := rtp.NewDemuxer(h264Depack, mpesDepack, xlog.L())
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
	tsMuxer.Close()
}
