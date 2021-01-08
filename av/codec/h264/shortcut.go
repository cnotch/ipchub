// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package h264

import "github.com/cnotch/ipchub/av/codec"

// MetadataIsReady .
func MetadataIsReady(vm *codec.VideoMeta) bool {
	sps := vm.Sps //ParameterSet(ParameterSetSps)
	pps := vm.Pps //ParameterSet(ParameterSetPps)
	if len(sps) == 0 || len(pps) == 0 {
		return false
	}

	if vm.Width == 0 {
		// decode
		var rawsps RawSPS
		if err := rawsps.Decode(sps); err != nil {
			return false
		}
		vm.Width = rawsps.Width()
		vm.Height = rawsps.Height()
		vm.FixedFrameRate = rawsps.IsFixedFrameRate()
		vm.FrameRate = rawsps.FrameRate()
	}
	return true
}

// NulType .
func NulType(nt byte) byte {
	return nt & NalTypeBitmask
}

// IsSps .
func IsSps(nt byte) bool {
	return nt&NalTypeBitmask == NalSps
}

// IsPps .
func IsPps(nt byte) bool {
	return nt&NalTypeBitmask == NalPps
}

// IsIdrSlice .
func IsIdrSlice(nt byte) bool {
	return nt&NalTypeBitmask == NalIdrSlice
}

// IsFillerData .
func IsFillerData(nt byte) bool {
	return nt&NalTypeBitmask == NalFillerData
}
