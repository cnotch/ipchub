// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scan

import (
	"testing"
	"unicode"
)

func TestPair_Scan(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		wantKey   string
		wantValue string
		wantOk    bool
	}{
		{
			"不带引号",
			"a=chj",
			"a",
			"chj",
			true,
		},
		{
			"带引号",
			"a=\"chj\"",
			"a",
			"chj",
			true,
		},
		{
			"带空个",
			" \ta=  \"chj\"\t",
			"a",
			"chj",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gotKey, gotValue, gotOk := EqualPair.Scan(tt.args)
			if gotKey != tt.wantKey {
				t.Errorf("Pair.Scan() gotKey = %v, want %v", gotKey, tt.wantKey)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Pair.Scan() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Pair.Scan() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestPair_ScanMultiRune(t *testing.T) {
	chinesePair := NewPair('是', unicode.IsSpace)
	tests := []struct {
		name      string
		args      string
		wantKey   string
		wantValue string
		wantOk    bool
	}{
		{
			"不带空格",
			"a是chj",
			"a",
			"chj",
			true,
		},
		{
			"不带空格",
			"a是 chj\t",
			"a",
			"chj",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gotKey, gotValue, gotOk := chinesePair.Scan(tt.args)
			if gotKey != tt.wantKey {
				t.Errorf("Pair.Scan() gotKey = %v, want %v", gotKey, tt.wantKey)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Pair.Scan() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Pair.Scan() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Benchmark_Pair_Scan(b *testing.B) {
	s := `realm="Another Streaming Media"`
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key, value, ok := EqualPair.Scan(s)
			_ = key
			_ = value
			_ = ok
		}
	})
}
