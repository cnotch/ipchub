// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"io"
	"strings"
	"time"
)

// Multicastable 支持组播模式的源
type Multicastable interface {
	AddMember(io.Closer)
	ReleaseMember(io.Closer)
	MulticastIP() string
	Port(index int) int
	TTL() int
	SourceIP() string
}

// Hlsable 支持Hls访问
type Hlsable interface {
	M3u8(token string) ([]byte, error)
	Segment(seq int) (io.Reader, int, error)
	LastAccessTime() time.Time
}

// Option 配置 Stream 的选项接口
type Option interface {
	apply(*Stream)
}

// optionFunc 包装函数以便它满足 Option 接口
type optionFunc func(*Stream)

func (f optionFunc) apply(s *Stream) {
	f(s)
}

// Attr 流属性选项
func Attr(k, v string) Option {
	return optionFunc(func(s *Stream) {
		k := strings.ToLower(strings.TrimSpace(k))
		s.attrs[k] = v
	})
}

// Multicast 流组播选项
func Multicast(multicast Multicastable) Option {
	return optionFunc(func(s *Stream) {
		s.multicast = multicast
	})
}

// Hls Hls选项
func Hls(hls Hlsable) Option {
	return optionFunc(func(s *Stream) {
		s.hls = hls
	})
}
