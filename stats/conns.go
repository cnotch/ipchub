// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package stats

import (
	"sync/atomic"
)

// 全局变量
var (
	RtspConns = NewConns() // RTSP连接统计
	WspConns  = NewConns() // WSP连接统计
	FlvConns  = NewConns() // flv连接统计
)

// ConnsSample 连接计数采样
type ConnsSample struct {
	Total  int64 `json:"total"`
	Active int64 `json:"active"`
}

// Conns 连接统计
type Conns interface {
	Add() int64
	Release() int64
	GetSample() ConnsSample
}

func (s *ConnsSample) clone() ConnsSample {
	return ConnsSample{
		Total:  atomic.LoadInt64(&s.Total),
		Active: atomic.LoadInt64(&s.Active),
	}
}

type conns struct {
	sample ConnsSample
}

// NewConns 新建连接计数
func NewConns() Conns {
	return &conns{}
}

func (c *conns) Add() int64 {
	atomic.AddInt64(&c.sample.Total, 1)
	return atomic.AddInt64(&c.sample.Active, 1)
}

func (c *conns) Release() int64 {
	return atomic.AddInt64(&c.sample.Active, -1)
}

func (c *conns) GetSample() ConnsSample {
	return c.sample.clone()
}
