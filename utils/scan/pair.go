// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scan

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// 预定义Pair扫描对象
var (
	// EqualPair 扫描 K=V这类形式的Pair字串
	EqualPair = NewPair('=',
		func(r rune) bool {
			return unicode.IsSpace(r) || r == '"'
		})

	// ColonPair 扫描 K:V 这类形式的Pair字串
	ColonPair = NewPair(':',
		func(r rune) bool {
			return unicode.IsSpace(r) || r == '"'
		})
)

// Pair 从字串扫描Key Value 值
type Pair struct {
	delim    rune              // Key Value 间的分割
	delimLen int               // 分割符长度
	trimFunc func(r rune) bool // 返回前 Trim使用的函数
}

// NewPair 新建 Pair 扫描器
func NewPair(delim rune, trimFunc func(r rune) bool) Pair {
	pair := Pair{
		delim:    delim,
		trimFunc: trimFunc,
	}
	pair.delimLen = utf8.RuneLen(delim)
	if trimFunc == nil {
		pair.trimFunc = func(r rune) bool { return false }
	}
	return pair
}

// Scan 提取 K V
func (p Pair) Scan(s string) (key, value string, found bool) {
	if p.delim == 0 {
		return s, "", false
	}

	i := strings.IndexRune(s, p.delim)
	if i < 0 {
		return s, "", false
	}

	return strings.TrimFunc(s[:i], p.trimFunc),
		strings.TrimFunc(s[i+p.delimLen:], p.trimFunc), true
}
