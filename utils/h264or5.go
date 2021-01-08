// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package utils

// RemoveH264or5EmulationBytes A general routine for making a copy of a (H.264 or H.265) NAL unit, removing 'emulation' bytes from the copy
// copy from live555
func RemoveH264or5EmulationBytes(from []byte) []byte {
	to := make([]byte, len(from))
	toMaxSize := len(to)
	fromSize := len(from)
	toSize := 0
	i := 0
	for i < fromSize && toSize+1 < toMaxSize {
		if i+2 < fromSize && from[i] == 0 && from[i+1] == 0 && from[i+2] == 3 {
			to[toSize] = 0
			to[toSize+1] = 0
			toSize += 2
			i += 3
		} else {
			to[toSize] = from[i]
			toSize++
			i++
		}
	}

	// 如果剩余最后一个字节，拷贝它
	if i < fromSize && toSize < toMaxSize {
		to[toSize] = from[i]
		toSize++
		i++
	}

	return to[:toSize]
	// return bytes.Replace(from, []byte{0, 0, 3}, []byte{0, 0}, -1)
}
