// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package bits

const uintBitsCount = int(32 << (^uint(0) >> 63))

// Reader .
type Reader struct {
	buf    []byte
	offset int // bit base
}

// NewReader retruns a new Reader.
func NewReader(buf []byte) *Reader {
	return &Reader{
		buf: buf,
	}
}

// Skip skip n bits.
func (r *Reader) Skip(n int) {
	if n <= 0 {
		return
	}
	_ = r.buf[(r.offset+n-1)>>3] // bounds check hint to compiler; see golang.org/issue/14808
	r.offset += n
}

// Peek peek the uint32 of n bits.
func (r *Reader) Peek(n int) uint64 {
	clone := *r
	return clone.readUint64(n, 64)
}

// Read read the uint32 of n bits.
func (r *Reader) Read(n int) uint32 {
	return uint32(r.readUint64(n, 32))
}

// ReadBit read a bit.
func (r *Reader) ReadBit() uint8 {
	_ = r.buf[(r.offset+1-1)>>3] // bounds check hint to compiler; see golang.org/issue/14808

	tmp := (r.buf[r.offset>>3] >> (7 - r.offset&0x7)) & 1
	r.offset++
	return tmp
}

// ReadUe .
func (r *Reader) ReadUe() (res uint32) {
	i := 0
	for {
		if bit := r.ReadBit(); !(bit == 0 && i < 32) {
			break
		}
		i++
	}

	res = r.Read(i)
	res += (1 << uint(i)) - 1
	return
}

// ReadSe .
func (r *Reader) ReadSe() (res int32) {
	ui32 := r.ReadUe()
	if ui32&0x01 != 0 {
		res = (int32(res) + 1) / 2
	} else {
		res = -int32(res) / 2
	}
	return
}

// ==== shortcut methods

// ReadBool read one bit bool.
func (r *Reader) ReadBool() bool { return bool(r.ReadBit() == 1) }

// ReadUint read the uint of n bits.
func (r *Reader) ReadUint(n int) uint { return uint(r.readUint64(n, uintBitsCount)) }

// ReadUint8 read the uint8 of n bits.
func (r *Reader) ReadUint8(n int) uint8 { return uint8(r.readUint64(n, 8)) }

// ReadUint16 read the uint16 of n bits.
func (r *Reader) ReadUint16(n int) uint16 { return uint16(r.readUint64(n, 16)) }

// ReadUint32 read the uint32 of n bits.
func (r *Reader) ReadUint32(n int) uint32 { return uint32(r.readUint64(n, 32)) }

// ReadUint64 read the uint64 of n bits.
func (r *Reader) ReadUint64(n int) uint64 { return r.readUint64(n, 64) }

// ReadInt read the int of n bits.
func (r *Reader) ReadInt(n int) int { return int(r.readUint64(n, uintBitsCount)) }

// ReadInt8 read the int8 of n bits.
func (r *Reader) ReadInt8(n int) int8 { return int8(r.readUint64(n, 8)) }

// ReadInt16 read the int16 of n bits.
func (r *Reader) ReadInt16(n int) int16 { return int16(r.readUint64(n, 16)) }

// ReadInt32 read the int32 of n bits.
func (r *Reader) ReadInt32(n int) int32 { return int32(r.readUint64(n, 32)) }

// ReadInt64 read the int64 of n bits.
func (r *Reader) ReadInt64(n int) int64 { return int64(r.readUint64(n, 64)) }

// ReadUe8 read the UE GolombCode of uint8.
func (r *Reader) ReadUe8() uint8 { return uint8(r.ReadUe()) }

// ReadUe16 read the UE GolombCode of uint16.
func (r *Reader) ReadUe16() uint16 { return uint16(r.ReadUe()) }

// ReadSe8 read the SE of int8.
func (r *Reader) ReadSe8() int8 { return int8(r.ReadSe()) }

// ReadSe16 read the SE of int16.
func (r *Reader) ReadSe16() int16 { return int16(r.ReadSe()) }

// Offset returns the offset of bits.
func (r *Reader) Offset() int {
	return r.offset
}

// BitsLeft returns the number of left bits.
func (r *Reader) BitsLeft() int {
	return len(r.buf)<<3 - r.offset
}

// BytesLeft returns the left byte slice.
func (r *Reader) BytesLeft() []byte {
	return r.buf[r.offset>>3:]
}

var bitsMask = [9]byte{
	0x00,
	0x01, 0x03, 0x07, 0x0f,
	0x1f, 0x3f, 0x7f, 0xff,
}

// readUint64 read the uint64 of n bits.
func (r *Reader) readUint64(n, max int) uint64 {
	if n <= 0 || n > max {
		return 0
	}

	_ = r.buf[(r.offset+n-1)>>3] // bounds check hint to compiler; see golang.org/issue/14808

	idx := r.offset >> 3
	validBits := 8 - r.offset&0x7
	r.offset += n

	var tmp uint64
	for n >= validBits {
		n -= validBits
		tmp |= uint64(r.buf[idx]&bitsMask[validBits]) << n
		idx++
		validBits = 8
	}

	if n > 0 {
		tmp |= uint64((r.buf[idx] >> (validBits - n)) & bitsMask[n])
	}
	return tmp
}
