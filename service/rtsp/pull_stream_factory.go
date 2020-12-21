// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"strings"

	"github.com/cnotch/ipchub/media"
)

func init() {
	// 注册拉流工厂
	media.RegistPullStreamFactory(NewPullStreamFacotry())
}

type pullStreamFactory struct {
}

// NewPullStreamFacotry 创建拉流工厂
func NewPullStreamFacotry() media.PullStreamFactory {
	return &pullStreamFactory{}
}

func (f *pullStreamFactory) Can(remoteURL string) bool {
	if len(remoteURL) >= len(rtspURLPrefix) && strings.EqualFold(remoteURL[:len(rtspURLPrefix)], rtspURLPrefix) {
		return true
	}
	return false
}

func (f *pullStreamFactory) Create(localPath, remoteURL string) (*media.Stream, error) {
	client, err := NewPullClient(localPath, remoteURL)
	if err != nil {
		return nil, err
	}
	err = client.Open()
	if err != nil {
		return nil, err
	}

	return client.stream, nil
}
