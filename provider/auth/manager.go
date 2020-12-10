// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"strings"
	"sync"

	"github.com/cnotch/xlog"
)

var globalM = &manager{
	m: make(map[string]*User),
}

func init() {
	// 默认为内存提供者，避免没有初始化全局函数调用问题
	globalM.Reset(&memProvider{})
}

// Reset 重置用户提供者
func Reset(provider UserProvider) {
	globalM.Reset(provider)
}

// All 获取所有的用户
func All() []*User {
	return globalM.All()
}

// Get 获取取指定名称的用户
func Get(userName string) *User {
	return globalM.Get(userName)
}

// Del 删除指定名称的用户
func Del(userName string) error {
	return globalM.Del(userName)
}

// Save 保存用户
func Save(src *User, updatePassword bool) error {
	return globalM.Save(src, updatePassword)
}

// Flush 刷新用户
func Flush() error {
	return globalM.Flush()
}

type manager struct {
	lock sync.RWMutex
	m    map[string]*User // 用户map
	l    []*User          // 用户list

	saves   []*User // 自上次Flush后新的保存和删除的用户
	removes []*User

	provider UserProvider
}

func (m *manager) Reset(provider UserProvider) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.m = make(map[string]*User)
	m.l = m.l[:0]
	m.saves = m.saves[:0]
	m.removes = m.removes[:0]
	m.provider = provider

	users, err := provider.LoadAll()
	if err != nil {
		panic("Load user fail")
	}

	if cap(m.l) < len(users) {
		m.l = make([]*User, 0, len(users))
	}

	// 加入缓存
	for _, u := range users {
		if err := u.init(); err != nil {
			xlog.Warnf("user table init failed: `%v`", err)
			continue // 忽略错误的配置
		}
		m.m[u.Name] = u
		m.l = append(m.l, u)
	}
}

func (m *manager) Get(userName string) *User {
	m.lock.RLock()
	defer m.lock.RUnlock()

	userName = strings.ToLower(userName)
	u, ok := m.m[userName]
	if ok {
		return u
	}
	return nil
}

func (m *manager) Del(userName string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	userName = strings.ToLower(userName)
	u, ok := m.m[userName]

	if ok {
		delete(m.m, userName)

		// 从完整列表中删除
		for i, u2 := range m.l {
			if u.Name == u2.Name {
				m.l = append(m.l[:i], m.l[i+1:]...)
				break
			}
		}

		// 从保存列表中删除
		for i, u2 := range m.saves {
			if u.Name == u2.Name {
				m.saves = append(m.saves[:i], m.saves[i+1:]...)
				break
			}
		}

		m.removes = append(m.removes, u)
	}
	return nil
}

func (m *manager) Save(newu *User, updatePassword bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	err := newu.init()
	if err != nil {
		return err
	}

	u, ok := m.m[newu.Name]

	if ok { // 更新
		u.CopyFrom(newu, updatePassword)

		save := true
		// 如果保存列表存在，不新增
		for _, u2 := range m.saves {
			if u.Name == u2.Name {
				save = false
				break
			}
		}

		if save {
			m.saves = append(m.saves, u)
		}
	} else { // 新增
		u = newu
		m.m[u.Name] = u

		m.l = append(m.l, u)
		m.saves = append(m.saves, u)

		for i, u2 := range m.removes {
			if u.Name == u2.Name {
				m.removes = append(m.removes[:i], m.removes[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (m *manager) Flush() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.saves)+len(m.removes) == 0 {
		return nil
	}

	err := m.provider.Flush(m.l, m.saves, m.removes)
	if err != nil {
		return err
	}

	m.saves = m.saves[:0]
	m.removes = m.removes[:0]
	return nil
}

func (m *manager) All() []*User {
	m.lock.RLock()
	defer m.lock.RUnlock()

	users := make([]*User, len(m.l))
	copy(users, m.l)
	return users
}
