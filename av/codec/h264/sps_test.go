// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package h264

import (
	"testing"
)

func TestRawSPS_Parse(t *testing.T) {
	tests := []struct {
		name    string
		b64     string
		wantW   int
		wantH   int
		wantFR  float64
		wantErr bool
	}{
		{
			"music",
			"Z01AH6sSB4CL9wgAAAMACAAAAwGUeMGMTA==",
			960,
			540,
			25,
			false,
		},
		{
			"game",
			"Z2QAH6zZQFAFuhAAAAMAEAAAAwPI8YMZYA==",
			1280,
			720,
			30,
			false,
		},
		{
			"4k",
			"Z2QAM6wspADwAQ+wFSAgICgAAB9IAAdTBO0LFok=",
			3840,
			2160,
			float64(60000) / float64(1001*2),
			false,
		},
		{
			"tpl500",
			"AAAAAWdkAB6s0gLASaEAAAMAAQAAAwAehA==",
			704,
			576,
			15,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sps := &RawSPS{}
			if err := sps.DecodeString(tt.b64); (err != nil) != tt.wantErr {
				t.Errorf("RawSPS.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if sps.Width() != tt.wantW {
				t.Errorf("RawSPS.Parse() Width = %v, wantWidth %v", sps.Width(), tt.wantW)
			}
			if sps.Height() != tt.wantH {
				t.Errorf("RawSPS.Parse() Height = %v, wantHeight %v", sps.Height(), tt.wantH)
			}
			if sps.FrameRate() != tt.wantFR {
				t.Errorf("RawSPS.Parse() FrameRate = %v, wantFrameRate %v", sps.FrameRate(), tt.wantFR)
			}
		})
	}
}
