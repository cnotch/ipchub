// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"errors"
	"io"
)

// PackBuffer 数据包缓冲
type PackBuffer struct {
	buf []Pack // contents are the Packs buf[off : len(buf)]
	off int    // read at &buf[off], write at &buf[len(buf)]
}

// NewPackBuffer 创建新的缓冲
func NewPackBuffer(buf []Pack) *PackBuffer {
	return &PackBuffer{buf: buf}
}

func makeSlice(n int) []Pack {
	// If the make fails, give a known error.
	defer func() {
		if recover() != nil {
			panic(ErrTooLarge)
		}
	}()
	return make([]Pack, n)
}

// 避免内存泄露，重置指针引用
func resetSlice(packs []Pack) {
	for i := 0; i < len(packs); i++ {
		packs[i] = nil
	}
}

func (b *PackBuffer) empty() bool { return len(b.buf) <= b.off }

// Packs 获取所有的包
func (b *PackBuffer) Packs() []Pack { return b.buf[b.off:] }

// Len 缓冲包长度
func (b *PackBuffer) Len() int { return len(b.buf) - b.off }

// Cap 缓冲容量
func (b *PackBuffer) Cap() int { return cap(b.buf) }

// Reset 重置缓冲
func (b *PackBuffer) Reset() {
	resetSlice(b.buf[b.off:])

	b.buf = b.buf[:0]
	b.off = 0
}

const maxLen = int(^uint(0) >> 16)

// ErrTooLarge buf太长了
var ErrTooLarge = errors.New("stream.PackBuffer: too large")

// 尝试在cap范围内扩展buf
func (b *PackBuffer) tryGrowByReslice(n int) (int, bool) {
	if l := len(b.buf); n <= cap(b.buf)-l {
		b.buf = b.buf[:l+n]
		return l, true
	}
	return 0, false
}

// 扩展buf
func (b *PackBuffer) grow(n int) int {
	m := b.Len()
	// If PackBuffer is empty, reset to recover space.
	if m == 0 && b.off != 0 {
		b.Reset()
	}
	// Try to grow by means of a reslice.
	if i, ok := b.tryGrowByReslice(n); ok {
		return i
	}
	// // Check if we can make use of bootstrap array.
	// if b.buf == nil && n <= len(b.bootstrap) {
	// 	b.buf = b.bootstrap[:n]
	// 	return 0
	// }
	c := cap(b.buf)
	if n <= c/2-m {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= c to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		copy(b.buf, b.buf[b.off:])
		resetSlice(b.buf[m:]) // 释放移动copy后剩余的指针引用
	} else if c > maxLen-c-n {
		panic(ErrTooLarge)
	} else {
		// Not enough space anywhere, we need to allocate.
		buf := makeSlice(2*c + n)
		copy(buf, b.buf[b.off:])
		resetSlice(b.buf[b.off:]) // 重新分配了缓冲，释放旧缓冲的指针引用
		b.buf = buf
	}
	// Restore b.off and len(b.buf).
	b.off = 0
	b.buf = b.buf[:m+n]
	return m
}

// Grow 扩展包长度能容纳接下来n个包数
func (b *PackBuffer) Grow(n int) {
	if n < 0 {
		panic("stream.PackBuffer.Grow: negative count")
	}
	m := b.grow(n)
	b.buf = b.buf[:m]
}

// Write 写入包数组
func (b *PackBuffer) Write(p []Pack) (n int, err error) {
	m, ok := b.tryGrowByReslice(len(p))
	if !ok {
		m = b.grow(len(p))
	}
	return copy(b.buf[m:], p), nil
}

// WritePack 写入单个包
func (b *PackBuffer) WritePack(c Pack) error {
	m, ok := b.tryGrowByReslice(1)
	if !ok {
		m = b.grow(1)
	}
	b.buf[m] = c
	return nil
}

// Skip 跳过指定个数的包
func (b *PackBuffer) Skip(size int) int {
	if b.empty() {
		// PackBuffer is empty, reset to recover space.
		b.Reset()
		return 0
	}
	
	len := b.Len()
	if size > len {
		size = len
	}
	resetSlice(b.buf[b.off : b.off+size]) // 释放已经读取的指针引用

	b.off += size
	return size
}

// Read 读包Slice
func (b *PackBuffer) Read(p []Pack) (n int, err error) {
	if b.empty() {
		// PackBuffer is empty, reset to recover space.
		b.Reset()
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}

	n = copy(p, b.buf[b.off:])
	resetSlice(b.buf[b.off : b.off+n]) // 释放已经读取的指针引用

	b.off += n
	return n, nil
}

// ReadPack 读一个包
func (b *PackBuffer) ReadPack() (Pack, error) {
	if b.empty() {
		// PackBuffer is empty, reset to recover space.
		b.Reset()
		return nil, io.EOF
	}

	p := b.buf[b.off]
	b.buf[b.off] = nil // 释放指针引用，避免内存泄露
	b.off++
	return p, nil
}
