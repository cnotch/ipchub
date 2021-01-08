// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package bits

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var bitsDatas = [][]byte{
	{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09},
	{
		0x47, 0x40, 0x00, 0x10, 0x00,
		0x00, 0xb0, 0x0d, 0x00, 0x01, 0xc1, 0x00, 0x00,
		0x00, 0x01, 0xf0, 0x01,
		0x2e, 0x70, 0x19, 0x05,
	},
}

func TestBitsReader_ReadBit(t *testing.T) {
	r := NewReader(bitsDatas[0])
	gotRet := r.ReadBit()
	wantRet := uint8(0)
	assert.Equal(t, wantRet, gotRet)

	gotRet = r.ReadBit()
	wantRet = 1
	assert.Equal(t, wantRet, gotRet)

	r.Skip(3)
	gotRet = r.ReadBit()
	wantRet = 1
	assert.Equal(t, wantRet, gotRet)

	gotRet = r.ReadBit()
	wantRet = 1
	assert.Equal(t, wantRet, gotRet)

	r.Skip(5)
	gotRet = r.ReadBit()
	wantRet = 1
	assert.Equal(t, wantRet, gotRet)

	gotRet = r.ReadBit()
	wantRet = 1
	assert.Equal(t, wantRet, gotRet)

	gotRet = r.ReadBit()
	wantRet = 0
	assert.Equal(t, wantRet, gotRet)

	gotRet = r.ReadUint8(8)
	wantRet = 0x2b
	assert.Equal(t, wantRet, gotRet)

}

func TestBitsReader_ReadUint16(t *testing.T) {
	r := NewReader(bitsDatas[0])
	gotRet := r.ReadUint16(16)
	wantRet := uint16(0x464c)
	assert.Equal(t, wantRet, gotRet)

	r.Skip(4)
	gotRet = r.ReadUint16(16)
	wantRet = uint16(0x6010)
	assert.Equal(t, wantRet, gotRet)

	r.Skip(1)
	gotRet = r.ReadUint16(2)
	wantRet = uint16(0x2)
	assert.Equal(t, wantRet, gotRet)
}

func TestBitsReader_ReadUint32(t *testing.T) {
	r := NewReader(bitsDatas[1])
	gotRet := r.ReadUint32(32)
	wantRet := uint32(0x47400010)
	assert.Equal(t, wantRet, gotRet)

	r.Skip(4)
	gotRet = r.ReadUint32(32)
	wantRet = uint32(0x000b00d0)
	assert.Equal(t, wantRet, gotRet)

	r.Skip(8)
	gotRet = r.ReadUint32(12)
	wantRet = uint32(0x1c1)
	assert.Equal(t, wantRet, gotRet)
}

func TestBitsReader_ReadUint64(t *testing.T) {
	r := NewReader(bitsDatas[1])
	gotRet := r.ReadUint64(36)
	wantRet := uint64(0x474000100)
	assert.Equal(t, wantRet, gotRet)

	gotRet = r.ReadUint64(32)
	wantRet = uint64(0x000b00d0)
	assert.Equal(t, wantRet, gotRet)

	r.Skip(8)
	gotRet = r.ReadUint64(12)
	wantRet = uint64(0x1c1)
	assert.Equal(t, wantRet, gotRet)
}

func BenchmarkReadBit(b *testing.B) {
	r := NewReader(bitsDatas[1])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.offset = 2
		ret := r.ReadBit()
		_ = ret
	}
}

func BenchmarkReadUint8(b *testing.B) {
	r := NewReader(bitsDatas[1])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.offset = 2
		ret := r.ReadUint8(7)
		_ = ret
	}
}

func BenchmarkReadUint16(b *testing.B) {
	r := NewReader(bitsDatas[1])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.offset = 2
		ret := r.ReadUint16(13)
		_ = ret
	}
}

func BenchmarkReadUint32(b *testing.B) {
	r := NewReader(bitsDatas[1])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.offset = 2
		ret := r.ReadUint32(29)
		_ = ret
	}
}

func BenchmarkReadUint64(b *testing.B) {
	r := NewReader(bitsDatas[1])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.offset = 2
		ret := r.ReadUint64(61)
		_ = ret
	}
}
