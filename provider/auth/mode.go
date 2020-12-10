// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"bytes"
	"errors"
	"fmt"
)

// Mode 认证模式
type Mode int

// 认证模式常量
const (
	NoneAuth Mode = iota
	BasicAuth
	DigestAuth
)

var errUnmarshalNilMode = errors.New("can't unmarshal a nil *Mode")

// String 返回认证模式字串
func (m Mode) String() string {
	switch m {
	case NoneAuth:
		return "NONE"
	case BasicAuth:
		return "BASIC"
	case DigestAuth:
		return "DIGEST"
	default:
		return fmt.Sprintf("AuthMode(%d)", m)
	}
}

// MarshalText 编入认证模式到文本
func (m Mode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText 从文本编出认证模式
// 典型的用于 YAML、TOML、JSON等文件编出
func (m *Mode) UnmarshalText(text []byte) error {
	if m == nil {
		return errUnmarshalNilMode
	}
	if !m.unmarshalText(text) && !m.unmarshalText(bytes.ToLower(text)) {
		return fmt.Errorf("unrecognized Mode: %q", text)
	}
	return nil
}

func (m *Mode) unmarshalText(text []byte) bool {
	switch string(text) {
	case "none", "NONE", "": // make the zero value useful
		*m = NoneAuth
	case "basic", "BASIC":
		*m = BasicAuth
	case "digest", "DIGEST":
		*m = DigestAuth
	default:
		return false
	}
	return true
}

// Set flag.Value 接口实现.
func (m *Mode) Set(s string) error {
	return m.UnmarshalText([]byte(s))
}

// Get flag.Getter 接口实现
func (m *Mode) Get() interface{} {
	return *m
}
