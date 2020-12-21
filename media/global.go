// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cnotch/ipchub/provider/route"
	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/scheduler"
	"github.com/cnotch/xlog"
)

// 全局变量
var (
	streams     sync.Map            // 流媒体集合 string->*Stream
	psFactories []PullStreamFactory // 拉流工厂
)

// PullStreamFactory 拉流工程流
type PullStreamFactory interface {
	Can(remoteURL string) bool
	Create(localPath, remoteURL string) (*Stream, error)
}

// RegistPullStreamFactory 注册拉流工厂
func RegistPullStreamFactory(f PullStreamFactory) {
	for _, factroy := range psFactories {
		if factroy == f {
			return
		}
	}
	psFactories = append(psFactories, f)
}

// Regist 注册流
func Regist(s *Stream) {
	// 获取同 path 的现有流
	oldSI, ok := streams.Load(s.path)
	if s == oldSI { // 如果是同一个源
		return
	}

	// 设置新流
	streams.Store(s.path, s)

	// 如果存在旧流
	if ok {
		oldS := oldSI.(*Stream)
		if oldS.ConsumerCount() <= 0 { // 没有消费者直接关闭
			oldS.close(StreamReplaced)
		} else { // 有消费者个5分钟检查一次，直到没有消费者就关闭
			runZeroConsumersCloseTask(oldS, StreamReplaced)
		}
	}
}

// Unregist 取消注册
func Unregist(s *Stream) {
	si, ok := streams.Load(s.path)
	if ok {
		s2 := si.(*Stream)
		if s2 == s {
			streams.Delete(s.path)
		}
	}
	s.Close()
}

// UnregistAll 取消全部注册的流
func UnregistAll() {
	streams.Range(func(key, value interface{}) bool {
		streams.Delete(key)
		s := value.(*Stream)
		s.Close()
		return true
	})
}

// Get 获取路径为 path 已存在的流。
func Get(path string) *Stream {
	path = utils.CanonicalPath(path)

	si, ok := streams.Load(path)
	if ok {
		return si.(*Stream)
	}
	return nil
}

// GetOrCreate 获取路径为 path 的流媒体,如果不存在尝试自动创建拉流。
func GetOrCreate(path string) *Stream {
	if s:=Get(path);s!=nil{
		return s
	}

	// 检查路由
	path = utils.CanonicalPath(path)
	r := route.Match(path)
	if r != nil {
		var s *Stream
		var err error
		for _, psf := range psFactories {
			if psf.Can(r.URL) {
				s, err = psf.Create(r.Pattern, r.URL)
				if err == nil {
					if !r.KeepAlive {
						// 启动没有消费时自动关闭任务
						runZeroConsumersCloseTask(s, StreamNoConsumer)
					}
				} else {
					xlog.Errorf("open pull stream from `%s` failed; %s.", r.URL, err.Error())
				}

				break
			}
		}

		if s != nil {
			return s
		}
	}

	return nil
}

// Count 流媒体数量和消费者数量
func Count() (sc, cc int) {
	streams.Range(func(key, value interface{}) bool {
		s := value.(*Stream)
		sc++
		cc += s.ConsumerCount()
		return true
	})
	return
}

// Infos 返回所有的流媒体信息
func Infos(pagetoken string, pagesize int, includeCS bool) (int, []*StreamInfo) {
	rtp := make(map[string]*StreamInfo)

	streams.Range(func(key, value interface{}) bool {
		s := value.(*Stream)
		rtp[s.Path()] = s.Info(includeCS)
		return true
	})

	count := len(rtp)
	ss := make([]*StreamInfo, 0, count)
	for _, v := range rtp {
		if v.Path > pagetoken {
			ss = append(ss, v)
		}
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Path < ss[j].Path
	})

	if pagesize > len(ss) {
		return count, ss
	}
	return count, ss[:pagesize]
}

func runZeroConsumersCloseTask(s *Stream, closedStatus int32) {
	timing := &runZeroConsumersClose{
		s:           s,
		d:           time.Minute * 5,
		closedStats: closedStatus,
	}
	scheduler.PostFunc(timing, timing.run,
		fmt.Sprintf("%s: The close task when the stream exceeds a certain amount of time without a consumer.", s.path))
}

// 不处于管理状态的流的监测计划
type runZeroConsumersClose struct {
	s           *Stream
	d           time.Duration
	closed      bool
	closedStats int32
}

func (r *runZeroConsumersClose) Next(t time.Time) time.Time {
	if r.closed {
		return time.Time{}
	}
	return t.Add(r.d)
}

func (r *runZeroConsumersClose) run() {
	if r.s.consumptions.Count() <= 0 {
		hlsable := r.s.Hlsable()
		if hlsable == nil || time.Now().Sub(hlsable.LastAccessTime()) >= r.d {
			r.closed = true
			r.s.close(r.closedStats)
		}
	}
}
