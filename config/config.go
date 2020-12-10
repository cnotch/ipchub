// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"flag"
)

// config 服务配置
type config struct {
	ListenAddr string          `json:"listen"`               // 服务侦听地址和端口
	Auth       bool            `json:"auth"`                 // 启用安全验证
	CacheGop   bool            `json:"cache_gop"`            // 缓存图像组，以便提高播放端打开速度，但内存需求大
	HlsPath    string          `json:"hlspath,omitempty"`    // Hls临时缓存目录
	Profile    bool            `json:"profile"`              // 是否启动Profile
	TLS        *TLSConfig      `json:"tls,omitempty"`        // https安全端口交互
	Routetable *ProviderConfig `json:"routetable,omitempty"` // 路由表
	Users      *ProviderConfig `json:"users,omitempty"`      // 用户
	Log        LogConfig       `json:"log"`                  // 日志配置
}

func (c *config) initFlags() {
	// 服务的端口
	flag.StringVar(&c.ListenAddr, "listen", ":554", "Set server listen address")
	flag.BoolVar(&c.Auth, "auth", false,
		"Determines if requires permission verification to access stream media")
	flag.BoolVar(&c.CacheGop, "cachegop", false,
		"Determines if Gop should be cached to memory")
	flag.StringVar(&c.HlsPath, "hlspath", "", "Set HLS live dir")
	flag.BoolVar(&c.Profile, "pprof", false,
		"Determines if profile enabled")

	// 初始化日志配置
	c.Log.initFlags()
}
