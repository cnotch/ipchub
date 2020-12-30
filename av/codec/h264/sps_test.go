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
			"base64_1",
			"Z2QAH6zZQFAFuhAAAAMAEAAAAwPI8YMZYA==",
			1280,
			720,
			30,
			false,
		},
		{
			"base64_2",
			"Z3oAH7y0AoAt0IAAAAMAgAAAHkeMGVA=",
			1280,
			720,
			30,
			false,
		},
		{
			"base64_3",
			"Z2QAM6wspADwAQ+wFSAgICgAAB9IAAdTBO0LFok=",
			3840,
			2160,
			float64(60000) / float64(1001*2),
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
