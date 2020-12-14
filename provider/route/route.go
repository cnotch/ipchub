// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package route

import (
	"net/url"

	"github.com/cnotch/ipchub/utils"
)

// Route 路由
type Route struct {
	Pattern   string `json:"pattern"`             // 路由模式字串
	URL       string `json:"url"`                 // 目标url
	KeepAlive bool   `json:"keepalive,omitempty"` // 是否一直保持连接，直到对方断开；默认 false，会在没有人使用时关闭
}

func (r *Route) init() error {
	r.Pattern = utils.CanonicalPath(r.Pattern)
	_, err := url.Parse(r.URL)
	if err != nil {
		return err
	}
	return nil
}

// CopyFrom 从源拷贝
func (r *Route) CopyFrom(src *Route) {
	r.URL = src.URL
	r.KeepAlive = src.KeepAlive
}

// Provider 路由提供者
type Provider interface {
	LoadAll() ([]*Route, error)
	Flush(full []*Route, saves []*Route, removes []*Route) error
}
