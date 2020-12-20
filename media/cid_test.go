// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"testing"
)

func TestCID(t *testing.T) {
	var consumerSequenceSeed uint32

	tests := []struct {
		name string
		typ  ConsumerType
	}{
		{
			"NewRTPConsumer",
			RTPConsumer,
		},
		{
			"NewFLVConsumer",
			FLVConsumer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cid := NewCID(tt.typ, &consumerSequenceSeed)
			if cid.Type() != tt.typ {
				t.Errorf("cid.Type() = %v, want %v", cid.Type(), tt.typ)
			}
			if cid.Sequence() != consumerSequenceSeed {
				t.Errorf("cid.Sequence() = %v, want %v", cid.Sequence(), consumerSequenceSeed)
			}
		})
	}
}
