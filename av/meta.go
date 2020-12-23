// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package av

// VideoMeta 视频元数据
type VideoMeta struct {
	Codec     string  `json:"codec"`
	DataRate  float64 `json:"datarate,omitempty"`
	Width     int     `json:"width,omitempty"`
	Height    int     `json:"height,omitempty"`
	FrameRate float64 `json:"framerate,omitempty"`
	Sps       []byte  `json:"-"`
	Pps       []byte  `json:"-"`
	Vps       []byte  `json:"-"`
}

// AudioMeta 音频元数据
type AudioMeta struct {
	Codec      string  `json:"codec"`
	DataRate   float64 `json:"datarate,omitempty"`
	SampleRate int     `json:"samplerate,omitempty"`
	SampleSize int     `json:"samplesize,omitempty"`
	Channels   int     `json:"channels,omitempty"`
	Sps        []byte  `json:"-"` // sps
}
