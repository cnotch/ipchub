// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// TLSConfig TLS listen 配置.
type TLSConfig struct {
	ListenAddr  string `json:"listen"`
	Certificate string `json:"cert"`
	PrivateKey  string `json:"key"`
}

// Load loads the certificates from the cache or the configuration.
func (c *TLSConfig) Load() (*tls.Config, error) {
	if c.PrivateKey == "" || c.Certificate == "" {
		return &tls.Config{}, errors.New("No certificate or private key configured")
	}

	// If the certificate provided is in plain text, write to file so we can read it.
	if strings.HasPrefix(c.Certificate, "---") {
		if err := ioutil.WriteFile("broker.crt", []byte(c.Certificate), os.ModePerm); err == nil {
			c.Certificate = Name+".crt"
		}
	}

	// If the private key provided is in plain text, write to file so we can read it.
	if strings.HasPrefix(c.PrivateKey, "---") {
		if err := ioutil.WriteFile("broker.key", []byte(c.PrivateKey), os.ModePerm); err == nil {
			c.PrivateKey = Name+".key"
		}
	}

	// Make sure the paths are absolute, otherwise we won't be able to read the files.
	c.Certificate = resolvePath(c.Certificate)
	c.PrivateKey = resolvePath(c.PrivateKey)

	// Load the certificate from the cert/key files.
	cer, err := tls.LoadX509KeyPair(c.Certificate, c.PrivateKey)
	return &tls.Config{
		Certificates: []tls.Certificate{cer},
	}, err
}

func resolvePath(path string) string {
	// Make sure the path is absolute
	path, _ = filepath.Abs(path)
	return path
}
