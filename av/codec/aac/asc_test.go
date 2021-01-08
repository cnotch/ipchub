// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAudioSpecificConfig_DecodeString(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		wantErr    bool
		objectType uint8
		sampleRate int
		channels   uint8
	}{
		{"case1", "121056E500", false, 2, 44100, 2},
		{"case2", "1190", false, 2, 48000, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var asc AudioSpecificConfig
			if err := asc.DecodeString(tt.config); (err != nil) != tt.wantErr {
				t.Errorf("AudioSpecificConfig.DecodeString() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, asc.ObjectType, tt.objectType)
			assert.Equal(t, asc.SampleRate, tt.sampleRate)
			assert.Equal(t, asc.Channels, tt.channels)
		})
	}
}
