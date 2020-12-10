// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"flag"
)

// RtmpConfig rtsp 配置
type RtmpConfig struct {
	ChunkSize int `json:"chunksize"`
}

func (c *RtmpConfig) initFlags() {
	// RTSP 组播
	flag.IntVar(&c.ChunkSize, "rtmp-chunksize", 16*1024, "Set RTMP ChunkSize")
}
