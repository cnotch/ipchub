// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package h264

import (
	bits "github.com/cnotch/bitutil"
)

func u8(r *bits.Reader, w int, target *uint8) (err error) {
	*target, err = r.ReadUint8(w)
	return
}

func u16(r *bits.Reader, w int, target *uint16) (err error) {
	*target, err = r.ReadUint16(w)
	return
}

func u32(r *bits.Reader, w int, target *uint32) (err error) {
	*target, err = r.ReadUint32(w)
	return
}

func flag(r *bits.Reader, target *uint8) (err error) {
	*target, err = r.ReadBit()
	return
}

func ue(r *bits.Reader) (res uint32, err error) {
	i := 0
	for {
		var bit uint8
		if bit, err = r.ReadBit(); err != nil {
			return
		}
		if !(bit == 0 && i < 32) {
			break
		}
		i++
	}

	if res, err = r.ReadUint32(i); err != nil {
		return
	}
	res += (1 << uint(i)) - 1
	return
}
func ue8(r *bits.Reader, target *uint8) error {
	temp, err := ue(r)
	*target = uint8(temp)
	return err
}

func ue16(r *bits.Reader, target *uint16) error {
	temp, err := ue(r)
	*target = uint16(temp)
	return err
}

func ue32(r *bits.Reader, target *uint32) (err error) {
	*target, err = ue(r)
	return
}

func se(r *bits.Reader) (res int32, err error) {
	var ui32 uint32
	if ui32, err = ue(r); err != nil {
		return
	}
	res = int32(ui32)

	if res&0x01 != 0 {
		res = (res + 1) / 2
	} else {
		res = -res / 2
	}
	return
}
func se8(r *bits.Reader, target *int8) error {
	temp, err := se(r)
	*target = int8(temp)
	return err
}
func se16(r *bits.Reader, target *int16) error {
	temp, err := se(r)
	*target = int16(temp)
	return err
}
func se32(r *bits.Reader, target *int32) (err error) {
	*target, err = se(r)
	return
}

// copy from live555
// A general routine for making a copy of a (H.264 or H.265) NAL unit, removing 'emulation' bytes from the copy
func removeH264or5EmulationBytes(from []byte) []byte {
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
