// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package sdp

import (
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/codec/aac"
	"github.com/cnotch/ipchub/av/codec/h264"
	"github.com/cnotch/ipchub/av/codec/hevc"
	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/ipchub/utils/scan"
	"github.com/pixelbender/go-sdp/sdp"
)

func ParseMetadata(rawsdp string, video *codec.VideoMeta, audio *codec.AudioMeta) error {
	sdp, err := sdp.ParseString(rawsdp)
	if err != nil {
		return err
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
	return nil
}

func parseAudioMeta(m *sdp.Format, audio *codec.AudioMeta) {
	audio.SampleSize = 16
	audio.Channels = 2
	audio.SampleRate = 44100
	if m.ClockRate > 0 {
		audio.SampleRate = m.ClockRate
	}
	if m.Channels > 0 {
		audio.Channels = m.Channels
	}

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

			var config []byte
			var err error
			if config, err = hex.DecodeString(p); err != nil {
				config = aac.Encode2BytesASC(2,
					byte(aac.SamplingIndex(audio.SampleRate)),
					byte(audio.Channels))
			}

			// audio.SetParameterSet(aac.ParameterSetConfig, config)
			audio.Sps = config
			_ = aac.MetadataIsReady(audio)
			break
		}
	}

}

func parseVideoMeta(m *sdp.Format, video *codec.VideoMeta) {
	if m.ClockRate > 0 {
		video.ClockRate = m.ClockRate
	}

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
		for _, p := range m.Params {
			i := strings.Index(p, "sprop-")
			if i < 0 {
				continue
			}
			parseH265VpsSpsPps(p[i:], video)
			break
		}
	}
}

func parseH264SpsPps(s string, video *codec.VideoMeta) {
	ppsStr, spsStr, ok := scan.Comma.Scan(s)
	if !ok {
		return
	}

	sps, err := base64.StdEncoding.DecodeString(spsStr)
	if err == nil {
		// video.SetParameterSet(h264.ParameterSetSps, sps)
		video.Sps = utils.RemoveNaluSeparator(sps)
	}

	pps, err := base64.StdEncoding.DecodeString(ppsStr)
	if err == nil {
		// video.SetParameterSet(h264.ParameterSetPps, pps)
		video.Pps = utils.RemoveNaluSeparator(pps)
	}

	_ = h264.MetadataIsReady(video)
}

func parseH265VpsSpsPps(s string, video *codec.VideoMeta) {
	var advance, token string
	continueScan := true
	advance = s
	for continueScan {
		advance, token, continueScan = scan.Semicolon.Scan(advance)
		name, value, ok := scan.EqualPair.Scan(token)
		if ok {
			var ps []byte
			var err error
			if ps, err = base64.StdEncoding.DecodeString(value); err != nil {
				return
			}
			ps = utils.RemoveNaluSeparator(ps)

			switch name {
			case "sprop-vps":
				video.Vps = ps
			case "sprop-sps":
				video.Sps = ps
			case "sprop-pps":
				video.Pps = ps
			}
		}
	}

	_ = hevc.MetadataIsReady(video)
}
