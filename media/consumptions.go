// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"sort"
	"sync"
	"sync/atomic"

	"github.com/cnotch/ipchub/media/cache"
)

type consumptions struct {
	sync.Map
	count int32
}

func (m *consumptions) SendToAll(p cache.Pack) {
	m.Range(func(key, value interface{}) bool {
		c := value.(*consumption)
		c.send(p)
		return true
	})
}

func (m *consumptions) RemoveAndCloseAll() {
	m.Range(func(key, value interface{}) bool {
		c := value.(*consumption)
		m.Delete(key)
		c.Close()
		return true
	})

	atomic.StoreInt32(&m.count, 0)
}

func (m *consumptions) Add(c *consumption) {
	m.Store(c.cid, c)
	atomic.AddInt32(&m.count, 1)
}

func (m *consumptions) Remove(cid CID) *consumption {
	ci, ok := m.Load(cid)
	if ok {
		m.Delete(cid)
		atomic.AddInt32(&m.count, -1)
		return ci.(*consumption)
	}
	return nil
}

func (m *consumptions) Count() int {
	return int(atomic.LoadInt32(&m.count))
}

func (m *consumptions) Infos() []ConsumptionInfo {
	cs := make([]ConsumptionInfo, 0, 10)
	m.Range(func(key, value interface{}) bool {
		cs = append(cs, value.(*consumption).Info())
		return true
	})

	sort.Slice(cs, func(i, j int) bool {
		return cs[i].ID < cs[j].ID
	})

	return cs
}
