// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package route

import (
	"sync"

	"github.com/cnotch/tomatox/utils"
	"github.com/cnotch/xlog"
)

var globalT = &routetable{
	m: make(map[string]*Route),
}

func init() {
	// 默认为内存提供者，避免没有初始化全局函数调用问题
	globalT.Reset(&memProvider{})
}

// Reset 重置路由表提供者
func Reset(provider Provider) {
	globalT.Reset(provider)
}

// Match 从路由表中获取和路径匹配的路由实例
func Match(path string) *Route {
	return globalT.Match(path)
}

// All 获取所有的路由
func All() []*Route {
	return globalT.All()
}

// Get 获取取指定模式的路由
func Get(pattern string) *Route {
	return globalT.Get(pattern)
}

// Del 删除指定模式的路由
func Del(pattern string) error {
	return globalT.Del(pattern)
}

// Save 保存路由
func Save(src *Route) error {
	return globalT.Save(src)
}

// Flush 刷新路由
func Flush() error {
	return globalT.Flush()
}

type routetable struct {
	lock sync.RWMutex
	m    map[string]*Route // 路由map
	l    []*Route          // 路由list

	saves   []*Route // 自上次Flush后新的保存和删除的路由
	removes []*Route

	provider Provider
}

func (t *routetable) Reset(provider Provider) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.m = make(map[string]*Route)
	t.l = t.l[:0]
	t.saves = t.saves[:0]
	t.removes = t.removes[:0]
	t.provider = provider

	routes, err := provider.LoadAll()
	if err != nil {
		panic("Load route table fail")
	}

	if cap(t.l) < len(routes) {
		t.l = make([]*Route, 0, len(routes))
	}

	// 加入缓存
	for _, r := range routes {
		if err := r.init(); err != nil {
			xlog.Warnf("route table init failed: `%v`", err)
			continue // 忽略错误的配置
		}
		t.m[r.Pattern] = r
		t.l = append(t.l, r)
	}
}

func (t *routetable) Match(path string) *Route {
	t.lock.RLock()
	defer t.lock.RUnlock()

	path = utils.CanonicalPath(path)
	if path[len(path)-1] == '/' { // 必须有具体的子路径
		return nil
	}

	r, ok := t.m[path]
	if ok { // 精确匹配
		ret := *r
		return &ret
	}

	// 获取最长有效匹配的路由
	var n = 0
	for k, v := range t.m {
		if !pathMatch(k, path) {
			continue
		}

		if r == nil || len(k) > n {
			n = len(k)
			r = v
		}
	}

	if r != nil {
		ret := *r
		r = &ret
		if r.URL[len(r.URL)-1] == '/' {
			r.URL = r.URL + path[len(r.Pattern):]
		} else {
			r.URL = r.URL + path[len(r.Pattern)-1:]
		}
		r.Pattern = path
	}
	return r
}

func (t *routetable) Get(pattern string) *Route {
	t.lock.RLock()
	defer t.lock.RUnlock()

	pattern = utils.CanonicalPath(pattern)
	r, _ := t.m[pattern]
	return r
}

func (t *routetable) Del(pattern string) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	pattern = utils.CanonicalPath(pattern)
	r, ok := t.m[pattern]

	if ok {
		delete(t.m, pattern)

		// 从完整列表中删除
		for i, r2 := range t.l {
			if r.Pattern == r2.Pattern {
				t.l = append(t.l[:i], t.l[i+1:]...)
				break
			}
		}

		// 从保存列表中删除
		for i, r2 := range t.saves {
			if r.Pattern == r2.Pattern {
				t.saves = append(t.saves[:i], t.saves[i+1:]...)
				break
			}
		}

		t.removes = append(t.removes, r)
	}
	return nil
}

func (t *routetable) Save(newr *Route) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	err := newr.init()
	if err != nil {
		return err
	}

	r, ok := t.m[newr.Pattern]

	if ok { // 更新
		r.CopyFrom(newr)

		save := true
		// 如果保存列表存在，不新增
		for _, r2 := range t.saves {
			if r.Pattern == r2.Pattern {
				save = false
				break
			}
		}

		if save {
			t.saves = append(t.saves, r)
		}
	} else { // 新增
		r = newr
		t.m[r.Pattern] = r

		t.l = append(t.l, r)
		t.saves = append(t.saves, r)

		for i, r2 := range t.removes {
			if r.Pattern == r2.Pattern {
				t.removes = append(t.removes[:i], t.removes[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (t *routetable) Flush() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if len(t.saves)+len(t.removes) == 0 {
		return nil
	}

	err := t.provider.Flush(t.l, t.saves, t.removes)
	if err != nil {
		return err
	}

	t.saves = t.saves[:0]
	t.removes = t.removes[:0]
	return nil
}

func (t *routetable) All() []*Route {
	t.lock.RLock()
	defer t.lock.RUnlock()

	routes := make([]*Route, len(t.l))
	copy(routes, t.l)
	return routes
}

// Does path match pattern?
func pathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		// should not happen
		return false
	}
	n := len(pattern)
	if pattern[n-1] != '/' {
		return pattern == path
	}
	return len(path) >= n && path[0:n] == pattern
}
