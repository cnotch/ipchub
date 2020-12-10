// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"errors"
	"strings"
)

// Provider 提供者接口
type Provider interface {
	Name() string
	Configure(config map[string]interface{}) error
}

// ProviderConfig 可扩展提供者配置
type ProviderConfig struct {
	Provider string                 `json:"provider"`         // 提供者类型
	Config   map[string]interface{} `json:"config,omitempty"` // 提供者配置
}

// Load 加载Provider
func (c *ProviderConfig) Load(builtins ...Provider) (Provider, error) {
	for _, builtin := range builtins {
		if strings.ToLower(builtin.Name()) == strings.ToLower(c.Provider) {
			if err := builtin.Configure(c.Config); err != nil {
				return nil, errors.New("The provider '" + c.Provider + "' could not be loaded. " + err.Error())
			}

			return builtin, nil
		}
	}

	// TODO: load a plugin provider
	return nil, errors.New("The provider '" + c.Provider + "' could not be loaded. ")
}

// LoadOrPanic 加载 Provider 如果失败直接 panics.
func (c *ProviderConfig) LoadOrPanic(builtins ...Provider) Provider {
	provider, err := c.Load(builtins...)
	if err != nil {
		panic(err)
	}

	return provider
}

// LoadProvider 加载Provider或Panic，默认值为第一个provider
func LoadProvider(config *ProviderConfig, providers ...Provider) Provider {
	if config == nil || config.Provider == "" {
		config = &ProviderConfig{
			Provider: providers[0].Name(),
		}
	}

	// Load the provider according to the configuration
	return config.LoadOrPanic(providers...)
}
