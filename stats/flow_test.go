// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package stats

import (
	"testing"
)

func TestFlow(t *testing.T) {
	totalFlow := NewFlow()
	sub1 := NewChildFlow(totalFlow)
	sub2 := NewChildFlow(totalFlow)

	t.Run("", func(t *testing.T) {
		sub1.AddIn(100)
		sample := sub1.GetSample()
		if sample.InBytes != 100 {
			t.Error("InBytes not is 100")
		}
		sub2.AddIn(200)
		sample = totalFlow.GetSample()
		if sample.InBytes != 300 {
			t.Error("InBytes not is 300")
		}
	})

}
