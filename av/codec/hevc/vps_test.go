// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hevc

import (
	"testing"
)

func TestH265RawVPS_DecodeString(t *testing.T) {
	tests := []struct {
		name    string
		b64     string
		wantErr bool
	}{
		{
			"base64_1",
			"QAEMAf//BAgAAAMAnQgAAAMAAF2VmAk=",
			false,
		},
		{
			"base64_2",
			"QAEMAf//AWAAAAMAkAAAAwAAAwBdlZgJ",
			false,
		},
		{
			"tpl500-265",
			"AAAAAUABDAH//wFgAAADAAADAAADAAADAJasCQ==",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vps := &H265RawVPS{}
			if err := vps.DecodeString(tt.b64); (err != nil) != tt.wantErr {
				t.Errorf("RawVPS.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Benchmark_VPSDecode(b *testing.B) {
	vpsstr := "QAEMAf//AWAAAAMAkAAAAwAAAwBdlZgJ"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			vps := &H265RawVPS{}
			_ = vps.DecodeString(vpsstr)
		}
	})
}
