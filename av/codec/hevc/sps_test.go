// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hevc

import (
	"testing"
)

func TestH265RawSPS_DecodeString(t *testing.T) {
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
			"QgEBAWAAAAMAkAAAAwAAAwBdoAKAgC0WWVmkkyuAQAAA+kAAF3AC",
			1280,
			720,
			float64(24000) / float64(1001),
			false,
		},
		{
			"base64_2",
			"QgEBBAgAAAMAnQgAAAMAAF2wAoCALRZZWaSTK4BAAAADAEAAAAeC",
			1280,
			720,
			30,
			false,
		},
		{
			"tpl500-265",
			"AAAAAUIBAQFgAAADAAADAAADAAADAJagAWggBln3ja5JMmuWMAgAAAMACAAAAwB4QA==",
			2880,
			1620,
			15,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sps := &H265RawSPS{}
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

func Benchmark_SPSDecode(b *testing.B) {
	spsstr := "QgEBAWAAAAMAkAAAAwAAAwBdoAKAgC0WWVmkkyuAQAAA+kAAF3ACQgEBAWAAAAMAkAAAAwAAAwBdoAKAgC0WWVmkkyuAQAAA+kAAF3AC"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sps := &H265RawSPS{}
			_ = sps.DecodeString(spsstr)
		}
	})
}
