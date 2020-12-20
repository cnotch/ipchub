// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPackQueue(t *testing.T) {
	t.Run("PackQueue", func(t *testing.T) {
		closed := false
		nilC := 0
		pq := NewPackQueue()
		go func() {
			for !closed {
				pack := pq.Dequeue()
				if pack == nil {
					nilC++
					continue
				}
			}
		}()
		for i := 0; i < 10000; i++ {
			p := emptyPack{}
			pq.Enqueue(p)
		}
		<-time.After(time.Millisecond * 100)
		closed = true
		pq.Signal()
		<-time.After(time.Millisecond * 10)
		assert.Equal(t, 1, nilC, "need = 1")
	})
}

type emptyPack struct {
}

func (p emptyPack) Size() int { return 100 }
