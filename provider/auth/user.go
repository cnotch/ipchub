// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/cnotch/ipchub/utils/scan"
)

// AccessRight 访问权限类型
type AccessRight int

// 权限常量
const (
	PullRight AccessRight = 1 << iota // 拉流权限
	PushRight                         // 推流权限
)

// UserProvider 用户提供者
type UserProvider interface {
	LoadAll() ([]*User, error)
	Flush(full []*User, saves []*User, removes []*User) error
}

// User 用户
type User struct {
	Name       string `json:"name"`
	Password   string `json:"password,omitempty"`
	Admin      bool   `json:"admin,omitempty"`
	PushAccess string `json:"push,omitempty"`
	PullAccess string `json:"pull,omitempty"`

	pushMatchers []PathMatcher
	pullMatchers []PathMatcher
}

func initMatchers(access string, destMatcher *[]PathMatcher) {
	advance := access
	pathMask := ""
	continueScan := true
	for continueScan {
		advance, pathMask, continueScan = scan.Semicolon.Scan(advance)
		if len(pathMask) == 0 {
			continue
		}
		*destMatcher = append(*destMatcher, NewPathMatcher(pathMask))
	}
}

func (u *User) init() error {
	u.Name = strings.ToLower(u.Name)
	if u.Admin {
		if len(u.PullAccess) == 0 {
			u.PullAccess = "*"
		}
		if len(u.PushAccess) == 0 {
			u.PushAccess = "*"
		}
	}

	initMatchers(u.PushAccess, &u.pushMatchers)
	initMatchers(u.PullAccess, &u.pullMatchers)
	return nil
}

// PasswordMD5 返回口令的MD5字串
func (u *User) PasswordMD5() string {
	if passwordNeedMD5(u.Password) {
		pw := md5.Sum([]byte(u.Password))
		return hex.EncodeToString(pw[:])
	}
	return u.Password
}

// ValidatePassword 验证密码
func (u *User) ValidatePassword(password string) error {
	if passwordNeedMD5(password) {
		pw := md5.Sum([]byte(password))
		password = hex.EncodeToString(pw[:])
	}

	if strings.EqualFold(u.PasswordMD5(), password) {
		return nil
	}
	return errors.New("password error")
}

// ValidatePermission 验证权限
func (u *User) ValidatePermission(path string, right AccessRight) bool {
	var matchers []PathMatcher
	switch right {
	case PushRight:
		matchers = u.pushMatchers
	case PullRight:
		matchers = u.pullMatchers
	}

	if matchers == nil {
		return false
	}

	path = strings.TrimSpace(path)
	for _, matcher := range matchers {
		if matcher.Match(path) {
			return true
		}
	}

	return false
}

// CopyFrom 从源属性并初始化
func (u *User) CopyFrom(src *User, withPassword bool) {
	if withPassword {
		u.Password = src.Password
	}
	u.Admin = src.Admin
	u.PushAccess = src.PushAccess
	u.PullAccess = src.PullAccess
	u.init()
}

// 密码是否需要进行md5处理，如果已经是md5则不处理
func passwordNeedMD5(password string) bool {
	if len(password) != 32 {
		return true
	}

	_, err := hex.DecodeString(password)
	if err != nil {
		return true
	}

	return false
}
