// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/cnotch/ipchub/av"
	"github.com/cnotch/ipchub/av/h264"
	"github.com/cnotch/ipchub/utils/scan"
	"github.com/pixelbender/go-sdp/sdp"
)

func parseMeta(rawsdp string, video *av.VideoMeta, audio *av.AudioMeta) {
	sdp, err := sdp.ParseString(rawsdp)
	if err != nil {
		return
	}

	for _, media := range sdp.Media {
		switch media.Type {
		case "video":
			video.Codec = media.Format[0].Name
			if video.Codec != "" {
				for _, bw := range media.Bandwidth {
					if bw.Type == "AS" {
						video.DataRate = float64(bw.Value)
					}
				}
				parseVideoMeta(media.Format[0], video)
			}

		case "audio":
			audio.Codec = media.Format[0].Name
			if audio.Codec == "MPEG4-GENERIC" {
				audio.Codec = "AAC"
			}

			if audio.Codec != "" {
				for _, bw := range media.Bandwidth {
					if bw.Type == "AS" {
						audio.DataRate = float64(bw.Value)
					}
				}
				parseAudioMeta(media.Format[0], audio)
			}
		}
	}
}

func parseAudioMeta(m *sdp.Format, audio *av.AudioMeta) {
	audio.SampleRate = m.ClockRate
	audio.Stereo = m.Channels == 2
	audio.SampleSize = m.Channels * 8

	// parse AAC config
	if len(m.Params) == 0 {
		return
	}
	if audio.Codec == "AAC" {
		for _, p := range m.Params {
			i := strings.Index(p, "config=")
			if i < 0 {
				continue
			}
			p = p[i+len("config="):]

			endi := strings.IndexByte(p, ';')
			if endi > -1 {
				p = p[:endi]
			}
			sps, err := hex.DecodeString(p)
			if err != nil {
				return
			}
			audio.Sps = []byte{0x11, 0x90, 0x56, 0xe5, 0x00}
			copy(audio.Sps, sps)
			break
		}
	}
}

func parseVideoMeta(m *sdp.Format, video *av.VideoMeta) {
	if len(m.Params) == 0 {
		return
	}
	switch video.Codec {
	case "h264", "H264":
		video.Codec = "H264"
		for _, p := range m.Params {
			i := strings.Index(p, "sprop-parameter-sets=")
			if i < 0 {
				continue
			}
			p = p[i+len("sprop-parameter-sets="):]

			endi := strings.IndexByte(p, ';')
			if endi > -1 {
				p = p[:endi]
			}
			parseH264SpsPps(p, video)
			break
		}
	case "h265", "H265", "hevc", "HEVC":
		video.Codec = "H265"
		// TODO: parse H265 vps sps pps
	}
}

func parseH264SpsPps(s string, video *av.VideoMeta) {
	ppsStr, spsStr, ok := scan.Comma.Scan(s)
	if !ok {
		return
	}

	sps, err := base64.StdEncoding.DecodeString(spsStr)
	if err != nil {
		return
	}

	pps, err := base64.StdEncoding.DecodeString(ppsStr)
	if err != nil {
		return
	}

	var rawSps h264.RawSPS
	err = rawSps.Decode(sps)
	if err != nil {
		return
	}

	video.Width = rawSps.Width()
	video.Height = rawSps.Height()
	video.FrameRate = rawSps.FrameRate()
	video.Sps = sps
	video.Pps = pps
}
