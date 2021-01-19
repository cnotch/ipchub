// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/format/flv"
	"github.com/cnotch/ipchub/av/format/hls"
	"github.com/cnotch/ipchub/av/format/mpegts"
	"github.com/cnotch/ipchub/av/format/rtp"
	"github.com/cnotch/ipchub/av/format/sdp"
	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/media/cache"
	"github.com/cnotch/ipchub/stats"
	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/queue"
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
	consumptions         consumptions // 消费者列表
	cache                packCache    // 媒体包缓存
	rtpDemuxer           rtpDemuxer
	flvMuxer             flvMuxer
	flvConsumptions      consumptions
	flvCache             packCache
	tsMuxer              *mpegts.Muxer
	hlsSG                *hls.SegmentGenerator
	hlsPlaylist          *hls.Playlist
	attrs                map[string]string // 流属性
	multicast            Multicastable
	hls                  Hlsable
	logger               *xlog.Logger // 日志对象
	Video                codec.VideoMeta
	Audio                codec.AudioMeta
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
	sdp.ParseMetadata(rawsdp, &s.Video, &s.Audio)

	// init Cache
	switch s.Video.Codec {
	case "H264":
		s.cache = cache.NewH264Cache(config.CacheGop())
	case "H265":
		s.cache = cache.NewHevcCache(config.CacheGop())
	default:
		s.cache = emptyCache{}
	}

	for _, option := range options {
		option.apply(s)
	}

	// prepareOtherStream
	s.prepareOtherStream()
	return s
}

func (s *Stream) prepareOtherStream() {
	// steam(rtp)->rtpdemuxer->stream(frame)->flvmuxer->stream(tag)

	s.flvCache = emptyCache{}
	s.flvMuxer = emptyFlvMuxer{}

	// prepare rtp.Packet -> codec.Frame
	var err error
	if s.rtpDemuxer, err = rtp.NewDemuxer(&s.Video, &s.Audio,
		s, s.logger.With(xlog.Fields(xlog.F("extra", "rtp2frame")))); err != nil {
		s.rtpDemuxer = emptyRtpDemuxer{}
		return
	}

	// prepare codec.Frame -> flv.Tag
	var flvMuxer *flv.Muxer
	if flvMuxer, err = flv.NewMuxer(&s.Video, &s.Audio,
		s, s.logger.With(xlog.Fields(xlog.F("extra", "frame2flv")))); err == nil {
		s.flvCache = cache.NewFlvCache(config.CacheGop())
		s.flvMuxer = flvMuxer
	}

	// prepare codec.Frame -> mpegts.Frame
	if s.Video.Codec == "H264" {
		hlsPlaylist := hls.NewPlaylist()
		sg, err := hls.NewSegmentGenerator(hlsPlaylist, s.path,
			config.HlsFragment(),
			config.HlsPath(), s.Audio.SampleRate,
			s.logger.With(xlog.Fields(xlog.F("extra", "hls.Muxer"))))
		if err != nil {
			return
		}
		tsMuxer, err2 := mpegts.NewMuxer(&s.Video, &s.Audio, sg,
			s.logger.With(xlog.Fields(xlog.F("extra", "ts.Muxer"))))
		if err2 != nil {
			return
		}
		s.tsMuxer = tsMuxer
		s.hlsSG = sg
		s.hlsPlaylist = hlsPlaylist
	}
}

// Path 流路径
func (s *Stream) Path() string {
	return s.path
}

// Sdp  sdp 字串
func (s *Stream) Sdp() string {
	return s.rawsdp
}

// FlvTypeFlags 支持的 flv TypeFlags
func (s *Stream) FlvTypeFlags() byte {
	return s.flvMuxer.TypeFlags()
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

	// 关闭 hls
	if s.tsMuxer != nil {
		s.tsMuxer.Close()
		s.hlsSG.Close()
		s.hlsPlaylist.Close()
	}

	// 关闭 flv 消费者和 Muxer
	s.flvConsumptions.RemoveAndCloseAll()
	s.flvCache.Reset()
	s.flvMuxer.Close()

	// 关闭 av.Frame 转换器
	s.rtpDemuxer.Close()

	s.consumptions.RemoveAndCloseAll()
	s.cache.Reset()
	return nil
}

// WriteRtpPacket 向流写入一个媒体包
func (s *Stream) WriteRtpPacket(packet *rtp.Packet) error {
	status := atomic.LoadInt32(&s.status)
	if status != StreamOK {
		return statusErrors[status]
	}

	atomic.AddUint64(&s.size, uint64(packet.Size()))

	keyframe := s.cache.CachePack(packet)
	s.consumptions.SendToAll(packet, keyframe)

	s.rtpDemuxer.WriteRtpPacket(packet)
	return nil
}

// WriteFrame .
func (s *Stream) WriteFrame(frame *codec.Frame) error {
	if err := s.flvMuxer.WriteFrame(frame); err != nil {
		s.logger.Error(err.Error())
	}
	if s.tsMuxer != nil {
		if err := s.tsMuxer.WriteFrame(frame); err != nil {
			s.logger.Error(err.Error())
		}
	}
	return nil
}

// WriteTag .
func (s *Stream) WriteFlvTag(tag *flv.Tag) error {
	status := atomic.LoadInt32(&s.status)
	if status != StreamOK {
		return statusErrors[status]
	}

	keyframe := s.flvCache.CachePack(tag)
	s.flvConsumptions.SendToAll(tag, keyframe)
	return nil
}

// Multicastable 返回组播支持能力，不支持返回nil
func (s *Stream) Multicastable() Multicastable {
	return s.multicast
}

// Hlsable 返回支持hls能力，不支持返回nil
func (s *Stream) Hlsable() Hlsable {
	return s.hlsPlaylist
}

func (s *Stream) startConsume(consumer Consumer, packetType PacketType, extra string, useGopCache bool) CID {
	if packetType == FLVPacket && s.flvMuxer == nil {
		return CID(0) // 不支持
	}

	c := &consumption{
		startOn:    time.Now(),
		stream:     s,
		cid:        NewCID(packetType, &s.consumerSequenceSeed),
		recvQueue:  queue.NewSyncQueue(),
		consumer:   consumer,
		packetType: packetType,
		extra:      extra,
		Flow:       stats.NewFlow(),
		maxQLen:    1000,
	}

	c.logger = s.logger.With(xlog.Fields(
		xlog.F("cid", uint32(c.cid)),
		xlog.F("packettype", c.packetType.String()),
		xlog.F("extra", c.extra)))

	cs := &s.consumptions
	cache := s.cache
	if packetType == FLVPacket {
		cs = &s.flvConsumptions
		cache = s.flvCache
	}

	if useGopCache {
		c.sendGop(cache) // 新消费者，先发送gop缓存
	}
	cs.Add(c)

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
	cs := &s.consumptions
	if cid.Type() == FLVPacket {
		cs = &s.flvConsumptions
	}

	c := cs.Remove(cid)
	if c != nil {
		c.Close()
	}
}

// ConsumerCount 流消费者计数
func (s *Stream) ConsumerCount() int {
	return s.consumptions.Count() + s.flvConsumptions.Count()
}

// StreamInfo 流信息
type StreamInfo struct {
	StartOn          string            `json:"start_on"`
	Path             string            `json:"path"`
	Addr             string            `json:"addr"`
	Size             int               `json:"size"`
	Video            *codec.VideoMeta  `json:"video,omitempty"`
	Audio            *codec.AudioMeta  `json:"audio,omitempty"`
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
		ConsumptionCount: s.ConsumerCount(),
	}

	if len(s.Video.Codec) != 0 {
		si.Video = &s.Video
	}
	if len(s.Audio.Codec) != 0 {
		si.Audio = &s.Audio
	}
	if includeCS {
		si.Consumptions = s.consumptions.Infos()
		si.Consumptions = append(si.Consumptions, s.flvConsumptions.Infos()...)
	}
	return si
}

// GetConsumption 获取指定消费信息
func (s *Stream) GetConsumption(cid CID) (ConsumptionInfo, bool) {
	cs := &s.consumptions
	if cid.Type() == FLVPacket {
		cs = &s.flvConsumptions
	}

	c, ok := cs.Load(cid)
	if ok {
		return c.(*consumption).Info(), ok
	}
	return ConsumptionInfo{}, false
}
