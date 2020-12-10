// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package stats

import (
	"sync/atomic"
)

// FlowSample 流统计采样
type FlowSample struct {
	InBytes  int64 `json:"inbytes"`
	OutBytes int64 `json:"outbytes"`
}

// Flow 流统计接口
type Flow interface {
	AddIn(size int64)      // 增加输入
	AddOut(size int64)     // 增加输出
	GetSample() FlowSample // 获取当前时点采样
}

func (fs *FlowSample) clone() FlowSample {
	return FlowSample{
		InBytes:  atomic.LoadInt64(&fs.InBytes),
		OutBytes: atomic.LoadInt64(&fs.OutBytes),
	}
}

// Add 采样累加
func (fs *FlowSample) Add(f FlowSample) {
	fs.InBytes = fs.InBytes + f.InBytes
	fs.OutBytes = fs.OutBytes + f.OutBytes
}

type flow struct {
	sample FlowSample
}

// NewFlow 创建流量统计
func NewFlow() Flow {
	return &flow{}
}

func (r *flow) AddIn(size int64) {
	atomic.AddInt64(&r.sample.InBytes, size)
}

func (r *flow) AddOut(size int64) {
	atomic.AddInt64(&r.sample.OutBytes, size)
}

func (r *flow) GetSample() FlowSample {
	return r.sample.clone()
}

type childFlow struct {
	parent Flow
	sample FlowSample
}

// NewChildFlow 创建子流量计数，它会把自己的计数Add到parent上
func NewChildFlow(parent Flow) Flow {
	return &childFlow{
		parent: parent,
	}
}

func (r *childFlow) AddIn(size int64) {
	atomic.AddInt64(&r.sample.InBytes, size)
	r.parent.AddIn(size)
}

func (r *childFlow) AddOut(size int64) {
	atomic.AddInt64(&r.sample.OutBytes, size)
	r.parent.AddOut(size)
}

func (r *childFlow) GetSample() FlowSample {
	return r.sample.clone()
}
