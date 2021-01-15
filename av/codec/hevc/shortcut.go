// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hevc

import "github.com/cnotch/ipchub/av/codec"

// MetadataIsReady .
func MetadataIsReady(vm *codec.VideoMeta) bool {
	vps := vm.Vps
	sps := vm.Sps //ParameterSet(ParameterSetSps)
	pps := vm.Pps //ParameterSet(ParameterSetPps)
	if len(vps) == 0 || len(sps) == 0 || len(pps) == 0 {
		return false
	}

	if vm.Width == 0 {
		// decode
		var rawsps H265RawSPS
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
	return (nt >> 1) & 0x3f
}
