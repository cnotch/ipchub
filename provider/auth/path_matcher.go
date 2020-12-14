// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"strings"
	"unicode"

	"github.com/cnotch/ipchub/utils/scan"
)

const (
	sectionWildcard = "+" // 单段通配符
	endWildcard     = "*" // 0-n段通配符，必须位于结尾
)

// 行分割
var pathScanner = scan.NewScanner('/', unicode.IsSpace)

// PathMatcher 路径匹配接口
type PathMatcher interface {
	Match(path string) bool
}

// NewPathMatcher 创建匹配器
func NewPathMatcher(pathMask string) PathMatcher {
	if strings.TrimSpace(pathMask) == endWildcard {
		return alwaysMatcher{}
	}

	parts := strings.Split(strings.ToLower(strings.Trim(pathMask, "/")), "/")
	wildcard := parts[len(parts)-1] == endWildcard
	if wildcard {
		parts = parts[0 : len(parts)-1]
	}
	return &pathMacher{parts: parts, wildcardEnd: wildcard}
}

type alwaysMatcher struct {
}

func (m alwaysMatcher) Match(path string) bool {
	return true
}

type pathMacher struct {
	parts       []string
	wildcardEnd bool
}

func (m *pathMacher) Match(path string) bool {
	path = strings.ToLower(strings.Trim(path, "/"))
	count := partCount(path) + 1

	if count < len(m.parts) {
		return false
	}

	if count > len(m.parts) && !m.wildcardEnd {
		return false
	}

	ok := true
	advance := path
	token := ""
	for i := 0; i < len(m.parts) && ok; i++ {
		advance, token, ok = pathScanner.Scan(advance)
		if sectionWildcard == m.parts[i] {
			continue // 跳过
		}
		if token != m.parts[i] {
			return false
		}
	}

	return true
}

func partCount(s string) int {
	n := 0
	for {
		i := strings.IndexByte(s, '/')
		if i == -1 {
			return n
		}
		n++
		s = s[i+1:]
	}
}
