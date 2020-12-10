// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanner_Scan(t *testing.T) {
	raw := "cao,hong,ju,ok"
	t.Run("Scan", func(t *testing.T) {
		advance, token, ok := Comma.Scan(raw)
		assert.True(t, ok)
		assert.Equal(t, "cao", token)
		assert.Equal(t, "hong,ju,ok", advance)
		i := 0
		for ok {
			advance, token, ok = Comma.Scan(advance)
			if ok {
				i++
			}
		}
		assert.Equal(t, 2, i)
		assert.Equal(t, "ok", token)
	})
}

func Benchmark_Scanner_Scan(b *testing.B) {
	s := `realm="Another Streaming Media", nonce="60a76a995a0cb012f1707abc188f60cb"`
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			realm := ""
			nonce := ""
			ok := true
			advance := s
			token := ""

			for ok {
				advance, token, ok = Comma.Scan(advance)
				k, v, _ := EqualPair.Scan(token)
				switch k {
				case "realm":
					realm = v
				case "nonce":
					nonce = v
				}
			}
			_ = realm
			_ = nonce
		}
	})
}
