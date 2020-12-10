// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"flag"

	"github.com/cnotch/tomatox/provider/auth"
)

// RtspConfig rtsp 配置
type RtspConfig struct {
	AuthMode auth.Mode `json:"authmode"`
}

func (c *RtspConfig) initFlags() {
	// RTSP 组播
	flag.Var(&c.AuthMode, "rtsp-auth", "Set RTSP auth mode")
}
