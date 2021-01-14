// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/cnotch/ipchub/av/codec"
	"github.com/cnotch/ipchub/av/format/amf"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// Packetizer 封包器
type Packetizer interface {
	PacketizeSequenceHeader() error
	Packetize(basePts int64, frame *codec.Frame) error
}

type emptyPacketizer struct{}

func (emptyPacketizer) PacketizeSequenceHeader() error                    { return nil }
func (emptyPacketizer) Packetize(basePts int64, frame *codec.Frame) error { return nil }

// 网络播放时 PTS（Presentation Time Stamp）的延时
// 影响视频 Tag 的 CTS 和音频的 DTS（Decoding Time Stamp）
const (
	ptsDelay = 1000
)

// Muxer flv muxer from av.Frame(H264[+AAC])
type Muxer struct {
	videoMeta *codec.VideoMeta
	audioMeta *codec.AudioMeta
	vp        Packetizer
	ap        Packetizer
	typeFlags byte
	recvQueue *queue.SyncQueue
	tagWriter TagWriter
	closed    bool

	logger *xlog.Logger // 日志对象
}

// NewMuxer .
func NewMuxer(videoMeta *codec.VideoMeta, audioMeta *codec.AudioMeta, tagWriter TagWriter, logger *xlog.Logger) (*Muxer, error) {
	muxer := &Muxer{
		recvQueue: queue.NewSyncQueue(),
		videoMeta: videoMeta,
		audioMeta: audioMeta,
		vp:        emptyPacketizer{},
		ap:        emptyPacketizer{},
		typeFlags: byte(TypeFlagsVideo),
		tagWriter: tagWriter,
		closed:    false,
		logger:    logger,
	}
	switch videoMeta.Codec {
	case "H264":
		muxer.vp = NewH264Packetizer(videoMeta, tagWriter)
	default:
		return nil, fmt.Errorf("flv muxer unsupport video codec type:%s", videoMeta.Codec)
	}

	if audioMeta.Codec == "AAC" {
		muxer.typeFlags |= TypeFlagsAudio
		muxer.ap = NewAacPacketizer(audioMeta, tagWriter)
	}

	go muxer.process()
	return muxer, nil
}

// WriteFrame .
func (muxer *Muxer) WriteFrame(frame *codec.Frame) error {
	muxer.recvQueue.Push(frame)
	return nil
}

// Close .
func (muxer *Muxer) Close() error {
	if muxer.closed {
		return nil
	}

	muxer.closed = true
	muxer.recvQueue.Signal()
	return nil
}

// TypeFlags 返回 flv header 中的 TypeFlags
func (muxer *Muxer) TypeFlags() byte {
	return muxer.typeFlags
}

func (muxer *Muxer) process() {
	defer func() {
		defer func() { // 避免 handler 再 panic
			recover()
		}()

		if r := recover(); r != nil {
			muxer.logger.Errorf("flvmuxer routine panic；r = %v \n %s", r, debug.Stack())
		}

		// 尽早通知GC，回收内存
		muxer.recvQueue.Reset()
	}()

	var basePts int64
	for !muxer.closed {
		f := muxer.recvQueue.Pop()
		if f == nil {
			if !muxer.closed {
				muxer.logger.Warn("flvmuxer:receive nil frame")
			}
			continue
		}

		frame := f.(*codec.Frame)
		if basePts == 0 {
			basePts = frame.AbsTimestamp
			muxer.muxMetadataTag()
			muxer.vp.PacketizeSequenceHeader()
			muxer.ap.PacketizeSequenceHeader()
		}

		switch frame.MediaType {
		case codec.MediaTypeVideo:
			if err := muxer.vp.Packetize(basePts, frame); err != nil {
				muxer.logger.Errorf("flvmuxer: muxVideoTag error - %s", err.Error())
			}
		case codec.MediaTypeAudio:
			if err := muxer.ap.Packetize(basePts, frame); err != nil {
				muxer.logger.Errorf("flvmuxer: muxAudioTag error - %s", err.Error())
			}
		default:
		}
	}
}

func (muxer *Muxer) muxMetadataTag() error {
	properties := make(amf.EcmaArray, 0, 12)

	properties = append(properties,
		amf.ObjectProperty{
			Name:  "creator",
			Value: "ipchub stream media server"})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  MetaDataCreationDate,
			Value: time.Now().Format(time.RFC3339)})

	if muxer.typeFlags&TypeFlagsAudio > 0 {
		properties = append(properties,
			amf.ObjectProperty{
				Name:  MetaDataAudioCodecID,
				Value: SoundFormatAAC})
		properties = append(properties,
			amf.ObjectProperty{
				Name:  MetaDataAudioDateRate,
				Value: muxer.audioMeta.DataRate})
		properties = append(properties,
			amf.ObjectProperty{
				Name:  MetaDataAudioSampleRate,
				Value: muxer.audioMeta.SampleRate})
		properties = append(properties,
			amf.ObjectProperty{
				Name:  MetaDataAudioSampleSize,
				Value: muxer.audioMeta.SampleSize})
		properties = append(properties,
			amf.ObjectProperty{
				Name:  MetaDataStereo,
				Value: muxer.audioMeta.Channels > 1})
	}

	vcodecID := CodecIDAVC
	if muxer.videoMeta.Codec == "H265" {
		vcodecID = CodecIDHEVC
	}

	properties = append(properties,
		amf.ObjectProperty{
			Name:  MetaDataVideoCodecID,
			Value: vcodecID})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  MetaDataVideoDataRate,
			Value: muxer.videoMeta.DataRate})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  MetaDataFrameRate,
			Value: muxer.videoMeta.FrameRate})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  MetaDataWidth,
			Value: muxer.videoMeta.Width})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  MetaDataHeight,
			Value: muxer.videoMeta.Height})

	scriptData := ScriptData{
		Name:  ScriptOnMetaData,
		Value: properties,
	}
	data, _ := scriptData.Marshal()

	tag := &Tag{
		TagType:   TagTypeAmf0Data,
		DataSize:  uint32(len(data)),
		Timestamp: 0,
		StreamID:  0,
		Data:      data,
	}

	return muxer.tagWriter.WriteFlvTag(tag)
}
