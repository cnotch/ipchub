// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cnotch/ipchub/av"
	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/media/cache"
	"github.com/cnotch/ipchub/protos/rtp"
	"github.com/cnotch/ipchub/stats"
	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/xlog"
)

// 流状态
const (
	StreamOK       int32 = iota
	StreamClosed         // 源关闭
	StreamReplaced       // 流被替换
	StreamNoConsumer
)

// 错误定义
var (
	// ErrStreamClosed 流被关闭
	ErrStreamClosed = errors.New("stream is closed")
	// ErrStreamReplaced 流被替换
	ErrStreamReplaced = errors.New("stream is replaced")
	statusErrors      = []error{nil, ErrStreamClosed, ErrStreamReplaced}
)

// Stream 媒体流
type Stream struct {
	startOn              time.Time // 启动时间
	path                 string    // 流路径
	rawsdp               string
	size                 uint64 // 流已经接收到的输入（字节）
	status               int32  // 流状态
	consumerSequenceSeed uint32
	consumptions         consumptions      // 消费者列表
	cache                cache.PackCache   // 媒体包缓存
	attrs                map[string]string // 流属性
	multicast            Multicastable
	hls                  Hlsable
	logger               *xlog.Logger // 日志对象
	Video                av.VideoMeta
	Audio                av.AudioMeta
}

// NewStream 创建新的流
func NewStream(path string, rawsdp string, options ...Option) *Stream {
	s := &Stream{
		startOn:              time.Now(),
		path:                 utils.CanonicalPath(path),
		rawsdp:               rawsdp,
		status:               StreamOK,
		consumerSequenceSeed: 0,
		attrs:                make(map[string]string, 2),
		logger:               xlog.L().With(xlog.Fields(xlog.F("path", path))),
	}

	// parseMeta
	parseMeta(rawsdp, &s.Video, &s.Audio)

	// init Cache
	switch s.Video.Codec {
	case "H264":
		s.cache = cache.NewH264Cache(config.CacheGop())
	case "H265":
		s.cache = cache.NewHevcCache(config.CacheGop())
	default:
		s.cache = cache.NewEmptyCache()
	}

	for _, option := range options {
		option.apply(s)
	}

	return s
}

// Path 流路径
func (s *Stream) Path() string {
	return s.path
}

// Sdp  sdp 字串
func (s *Stream) Sdp() string {
	return s.rawsdp
}

// Attr 流属性
func (s *Stream) Attr(key string) string {
	return s.attrs[strings.ToLower(strings.TrimSpace(key))]
}

// Close 关闭流
func (s *Stream) Close() error {
	return s.close(StreamClosed)
}
func (s *Stream) close(status int32) error {
	if atomic.LoadInt32(&s.status) != StreamOK {
		return nil
	}

	// 修改流状态
	if status != StreamReplaced {
		status = StreamClosed
	}
	atomic.StoreInt32(&s.status, status)

	s.consumptions.RemoveAndCloseAll()

	// 尽早通知GC，回收内存
	s.cache.Reset()
	return nil
}

// WritePacket 向流写入一个媒体包
func (s *Stream) WritePacket(packet *rtp.Packet) error {
	status := atomic.LoadInt32(&s.status)
	if status != StreamOK {
		return statusErrors[status]
	}

	atomic.AddUint64(&s.size, uint64(packet.Size()))

	s.cache.CachePack(packet)

	s.consumptions.SendToAll(packet)

	return nil
}

// Multicastable 返回组播支持能力，不支持返回nil
func (s *Stream) Multicastable() Multicastable {
	return s.multicast
}

// Hlsable 返回支持hls能力，不支持返回nil
func (s *Stream) Hlsable() Hlsable {
	return s.hls
}

func (s *Stream) startConsume(consumer Consumer, packetType PacketType, extra string, useGopCache bool) CID {
	c := &consumption{
		startOn:    time.Now(),
		stream:     s,
		cid:        NewCID(packetType, &s.consumerSequenceSeed),
		recvQueue:  cache.NewPackQueue(),
		consumer:   consumer,
		packetType: packetType,
		extra:      extra,
		Flow:       stats.NewFlow(),
	}

	c.logger = s.logger.With(xlog.Fields(
		xlog.F("cid", uint32(c.cid)),
		xlog.F("packettype", c.packetType.String()),
		xlog.F("extra", c.extra)))

	if useGopCache {
		c.sendGop(s.cache) // 新消费者，先发送gop缓存
	}

	s.consumptions.Add(c)
	go c.consume()

	return c.cid
}

// StartConsume 开始消费
func (s *Stream) StartConsume(consumer Consumer, packetType PacketType, extra string) CID {
	return s.startConsume(consumer, packetType, extra, true)
}

// StartConsumeNoGopCache 开始消费,不使用GopCahce
func (s *Stream) StartConsumeNoGopCache(consumer Consumer, packetType PacketType, extra string) CID {
	return s.startConsume(consumer, packetType, extra, false)
}

// StopConsume 开始消费
func (s *Stream) StopConsume(cid CID) {
	c := s.consumptions.Remove(cid)
	if c != nil {
		c.Close()
	}
}

// ConsumerCount 流消费者计数
func (s *Stream) ConsumerCount() int {
	return s.consumptions.Count()
}

// StreamInfo 流信息
type StreamInfo struct {
	StartOn          string            `json:"start_on"`
	Path             string            `json:"path"`
	Addr             string            `json:"addr"`
	Size             int               `json:"size"`
	Video            *av.VideoMeta     `json:"video,omitempty"`
	Audio            *av.AudioMeta     `json:"audio,omitempty"`
	ConsumptionCount int               `json:"cc"`
	Consumptions     []ConsumptionInfo `json:"cs,omitempty"`
}

// Info 获取流信息
func (s *Stream) Info(includeCS bool) *StreamInfo {
	si := &StreamInfo{
		StartOn:          s.startOn.Format(time.RFC3339Nano),
		Path:             s.path,
		Addr:             s.Attr("addr"),
		Size:             int(atomic.LoadUint64(&s.size) / 1024),
		ConsumptionCount: s.consumptions.Count(),
	}

	if len(s.Video.Codec) != 0 {
		si.Video = &s.Video
	}
	if len(s.Audio.Codec) != 0 {
		si.Audio = &s.Audio
	}
	if includeCS {
		si.Consumptions = s.consumptions.Infos()
	}
	return si
}

// GetConsumption 获取指定消费信息
func (s *Stream) GetConsumption(cid CID) (ConsumptionInfo, bool) {
	c, ok := s.consumptions.Load(cid)
	if ok {
		return c.(*consumption).Info(), ok
	}
	return ConsumptionInfo{}, false
}
