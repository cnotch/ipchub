// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewADTSHeader(t *testing.T) {
	tests := []struct {
		name          string
		profile       byte
		sampleRateIdx byte
		channelConfig byte
		payloadSize   int
	}{
		{"case1", 1, 4, 2, 200},
		{"case1", 2, 3, 4, 5345},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewADTSHeader(tt.profile, tt.sampleRateIdx, tt.channelConfig, tt.payloadSize)
			assert.Equal(t, tt.profile, got.Profile())
			assert.Equal(t, tt.sampleRateIdx, got.SamplingIndex())
			assert.Equal(t, tt.channelConfig, got.ChannelConfig())
			assert.Equal(t, tt.payloadSize, got.PayloadSize())
		})
	}
}

func TestADTSHeader_ToAsc(t *testing.T) {
	tests := []struct {
		name          string
		profile       byte
		sampleRateIdx byte
		channelConfig byte
		payloadSize   int
	}{
		{"case1", 1, 4, 2, 200},
		{"case1", 2, 3, 4, 5345},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewADTSHeader(tt.profile, tt.sampleRateIdx, tt.channelConfig, tt.payloadSize)
			config := got.ToAsc()
			var asc AudioSpecificConfig
			asc.Decode(config)

			assert.Equal(t, tt.profile, asc.ObjectType-1)
			assert.Equal(t, tt.sampleRateIdx, asc.SamplingIndex)
			assert.Equal(t, tt.channelConfig, asc.ChannelConfig)
		})
	}
}
