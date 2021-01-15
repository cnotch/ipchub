// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"time"
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

	// Attempt to load a plugin provider
	p, err := plugin.Open(resolvePath(c.Provider))
	if err != nil {
		return nil, errors.New("The provider plugin '" + c.Provider + "' could not be opened. " + err.Error())
	}

	// Get the symbol
	sym, err := p.Lookup("New")
	if err != nil {
		return nil, errors.New("The provider '" + c.Provider + "' does not contain 'func New() interface{}' symbol")
	}

	// Resolve the
	pFactory, validFunc := sym.(*func() interface{})
	if !validFunc {
		return nil, errors.New("The provider '" + c.Provider + "' does not contain 'func New() interface{}' symbol")
	}

	// Construct the provider
	provider, validProv := ((*pFactory)()).(Provider)
	if !validProv {
		return nil, errors.New("The provider '" + c.Provider + "' does not implement 'Provider'")
	}

	// Configure the provider
	err = provider.Configure(c.Config)
	if err != nil {
		return nil, errors.New("The provider '" + c.Provider + "' could not be configured")
	}

	// Succesfully opened and configured a provider
	return provider, nil
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

func resolvePath(path string) string {
	// If it's an url, download the file
	if strings.HasPrefix(path, "http") {
		f, err := httpFile(path)
		if err != nil {
			panic(err)
		}

		// Get the downloaded file path
		path = f.Name()
	}

	// Make sure the path is absolute
	path, _ = filepath.Abs(path)
	return path
}

// DefaultClient used for http with a shorter timeout.
var defaultClient = &http.Client{
	Timeout: 5 * time.Second,
}

// httpFile downloads a file from HTTP
var httpFile = func(url string) (*os.File, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	output, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if _, err := io.Copy(output, response.Body); err != nil {
		return nil, err
	}

	return output, nil
}
