// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package aac

import "github.com/cnotch/ipchub/av/codec"

// MetadataIsReady .
func MetadataIsReady(am *codec.AudioMeta) bool {
	config := am.Sps //ParameterSet(ParameterSetConfig)
	if len(config) == 0 {
		return false
	}
	if am.SampleRate == 0 {
		// decode
		var asc AudioSpecificConfig
		if err := asc.Decode(config); err != nil {
			return false
		}
		am.Channels = int(asc.Channels)
		am.SampleRate = asc.SampleRate
		if asc.ExtSampleRate > 0 {
			am.SampleRate = asc.ExtSampleRate
		}
		am.SampleSize = 16
	}
	return true
}
