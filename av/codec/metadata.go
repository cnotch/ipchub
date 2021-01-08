// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package codec

// VideoMeta 视频元数据
type VideoMeta struct {
	Codec          string  `json:"codec"`
	Width          int     `json:"width,omitempty"`
	Height         int     `json:"height,omitempty"`
	FixedFrameRate bool    `json:"fixedframerate,omitempty"`
	FrameRate      float64 `json:"framerate,omitempty"`
	DataRate       float64 `json:"datarate,omitempty"`
	ClockRate      int     `json:"clockrate,omitempty"`
	Sps            []byte  `json:"-"`
	Pps            []byte  `json:"-"`
	Vps            []byte  `json:"-"`

	// // 媒体的参数集，如 sdp中的 sprop_xxx
	// parameterSets `json:"-"`
	// // 不同封装和传输方式的特别参数
	// // 比如 RTP 封装:  streamid, packetization-mode,  profile-level-id 等
	// specificParams `json:"-"`
}

// AudioMeta 音频元数据
type AudioMeta struct {
	Codec      string  `json:"codec"`
	SampleRate int     `json:"samplerate,omitempty"`
	SampleSize int     `json:"samplesize,omitempty"`
	Channels   int     `json:"channels,omitempty"`
	DataRate   float64 `json:"datarate,omitempty"`
	Sps        []byte  `json:"-"` // sps

	// // 媒体的参数集，如 sdp中的 sprop_xxx
	// parameterSets `json:"-"`
	// // 不同封装和传输方式的特别参数
	// // 比如 RTP 封装:  streamid, mode, profile-level-id,sizelength, indexlength,indexdeltalength 等
	// specificParams `json:"-"`
}

type parameterSets [][]byte

func (pss *parameterSets) ParameterSet(idx int) []byte {
	if len(*pss) <= idx {
		return nil
	}
	return (*pss)[idx]
}
func (pss *parameterSets) SetParameterSet(idx int, paramSet []byte) {
	if len(*pss) <= idx {
		temp := make(parameterSets, idx+1)
		copy(temp, *pss)
		*pss = temp
	}
	(*pss)[idx] = paramSet
}

type specificParams map[string]string

func (params *specificParams) SpecificParam(name string) (value string, ok bool) {
	if params == nil {
		return
	}
	value, ok = (*params)[name]
	return
}

func (params *specificParams) SetSpecificParam(name, value string) {
	if params == nil {
		*params = make(specificParams)
	}
	(*params)[name] = value
}
