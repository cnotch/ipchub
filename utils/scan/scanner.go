// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scan

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// 扫描器
var (
	// 逗号分割
	Comma = NewScanner(',', unicode.IsSpace)
	// 分号分割
	Semicolon = NewScanner(';', unicode.IsSpace)
	// 空格分割
	Space = NewScanner(' ', nil)
	// 行分割
	Line = NewScanner('\n', unicode.IsSpace)
)

// Scanner 扫描器
type Scanner struct {
	delim    rune
	delimLen int
	trimFunc func(r rune) bool
}

// NewScanner 创建扫描器
func NewScanner(delim rune, trimFunc func(r rune) bool) Scanner {
	scanner := Scanner{
		delim:    delim,
		trimFunc: trimFunc,
	}
	scanner.delimLen = utf8.RuneLen(delim)
	if trimFunc == nil {
		scanner.trimFunc = func(r rune) bool { return false }
	}
	return scanner
}

// Scan 扫描字串
func (s Scanner) Scan(str string) (advance, token string, continueScan bool) {
	i := strings.IndexRune(str, s.delim)
	if i < 0 {
		return "", strings.TrimFunc(str, s.trimFunc), false
	}

	return strings.TrimFunc(str[i+s.delimLen:], s.trimFunc), strings.TrimFunc(str[:i], s.trimFunc), true
}
