// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"sync"
	"time"

	"github.com/cnotch/ipchub/provider/security"
)

// Token 用户登录后的Token
type Token struct {
	Username string `json:"-"`
	AToken   string `json:"access_token"`
	AExp     int64  `json:"-"`
	RToken   string `json:"refresh_token"`
	RExp     int64  `json:"-"`
}

// TokenManager token管理
type TokenManager struct {
	tokens sync.Map // token->Token
}

// NewToken 给用户新建Token
func (tm *TokenManager) NewToken(username string) *Token {
	token := &Token{
		Username: username,
		AToken:   security.NewID().MD5(),
		AExp:     time.Now().Add(time.Hour * time.Duration(2)).Unix(),
		RToken:   security.NewID().MD5(),
		RExp:     time.Now().Add(time.Hour * time.Duration(7*24)).Unix(),
	}

	tm.tokens.Store(token.AToken, token)
	tm.tokens.Store(token.RToken, token)
	return token
}

// Refresh 刷新指定的Token
func (tm *TokenManager) Refresh(rtoken string) *Token {
	ti, ok := tm.tokens.Load(rtoken)
	if ok {
		oldToken := ti.(*Token)
		username := oldToken.Username
		if rtoken == oldToken.RToken { // 是refresh token
			tm.tokens.Delete(oldToken.AToken)
			tm.tokens.Delete(oldToken.RToken)
			if oldToken.RExp > time.Now().Unix() {
				return tm.NewToken(username)
			}
		}
	}
	return nil
}

// AccessCheck 访问检测
func (tm *TokenManager) AccessCheck(atoken string) string {
	ti, ok := tm.tokens.Load(atoken)
	if ok {
		token := ti.(*Token)
		if token.AToken == atoken { // 访问token
			if token.AExp > time.Now().Unix() {
				return token.Username
			}
			tm.tokens.Delete(token.AToken)
		}
	}
	return ""
}

// ExpCheck 过期检测
func (tm *TokenManager) ExpCheck() {
	tm.tokens.Range(func(k, v interface{}) bool {
		token := v.(*Token)
		if time.Now().Unix() > token.AExp {
			tm.tokens.Delete(token.AToken)
		}
		if time.Now().Unix() > token.RExp {
			tm.tokens.Delete(token.RToken)
		}
		return true
	})
}
