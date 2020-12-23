// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

import "testing"

func TestRawSPS_DecodeString(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{"case1", "121056E500", false},
		{"case2", "1190", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sps RawSPS
			if err := sps.DecodeString(tt.config); (err != nil) != tt.wantErr {
				t.Errorf("RawSPS.DecodeString() error = %v, wantErr %v", err, tt.wantErr)
			}
			profile := sps.Profile
			_ = profile
		})
	}
}

func TestRawSPS_Encode2Bytes(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		configBytes [2]byte
	}{
		{"case1", "121056E500", [2]byte{0x12, 0x10}},
		{"case2", "1190", [2]byte{0x11, 0x90}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sps RawSPS
			sps.DecodeString(tt.config)
			config :=sps.Encode2Bytes()
			if config != tt.configBytes {
				t.Errorf("RawSPS.Encode2Bytes() return = %v, want = %v", config, tt.configBytes)
			}
			profile := sps.Profile
			_ = profile
		})
	}
}
