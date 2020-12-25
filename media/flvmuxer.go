// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package media

import (
	"io"
	"runtime/debug"
	"time"

	"github.com/cnotch/ipchub/av"
	"github.com/cnotch/ipchub/av/h264"
	"github.com/cnotch/ipchub/protos/amf"
	"github.com/cnotch/ipchub/protos/flv"
	"github.com/cnotch/ipchub/protos/rtp"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// 网络播放时 PTS（Presentation Time Stamp）的延时
// 影响视频 Tag 的 CTS 和音频的 DTS（Decoding Time Stamp）
const ptsDelay = 1000

type flvMuxer struct {
	videoMeta         av.VideoMeta
	audioMeta         av.AudioMeta
	typeFlags         byte
	audioDataTemplate *flv.AudioData
	recvQueue         *queue.SyncQueue
	extractFuncs      [4]func(packet *rtp.Packet) error
	tagWriter         flv.TagWriter
	closed            bool
	spsMuxed          bool
	baseNtp           int64
	baseTs            int64
	logger            *xlog.Logger // 日志对象
}

func newFlvMuxer(videoMeta av.VideoMeta, audioMeta av.AudioMeta, tagWriter flv.TagWriter, logger *xlog.Logger) FlvMuxer {
	muxer := &flvMuxer{
		recvQueue: queue.NewSyncQueue(),
		videoMeta: videoMeta,
		audioMeta: audioMeta,
		typeFlags: byte(flv.TypeFlagsVideo),
		tagWriter: tagWriter,
		closed:    false,
		baseTs:    time.Now().UnixNano() / int64(time.Millisecond),
		logger:    logger,
	}

	h264Extractor := rtp.NewH264FrameExtractor(muxer)
	muxer.extractFuncs[rtp.ChannelVideo] = h264Extractor.Extract
	muxer.extractFuncs[rtp.ChannelVideoControl] = h264Extractor.Control
	if audioMeta.Codec == "AAC" {
		muxer.typeFlags |= flv.TypeFlagsAudio
		mpesExtractor := rtp.NewMPESFrameExtractor(muxer, audioMeta.SampleRate)
		muxer.extractFuncs[rtp.ChannelAudio] = mpesExtractor.Extract
		muxer.extractFuncs[rtp.ChannelAudioControl] = mpesExtractor.Control
		muxer.prepareTemplate()
	} else {
		muxer.extractFuncs[rtp.ChannelAudio] = func(*rtp.Packet) error { return nil }
		muxer.extractFuncs[rtp.ChannelAudioControl] = func(*rtp.Packet) error { return nil }
	}

	go muxer.consume()
	return muxer
}

func (muxer *flvMuxer) prepareTemplate() {
	audioData := &flv.AudioData{
		SoundFormat:   flv.SoundFormatAAC,
		AACPacketType: flv.AACPacketTypeRawData,
		Body:          nil,
	}

	switch muxer.audioMeta.SampleRate {
	case 5512:
		audioData.SoundRate = flv.SoundRate5512
	case 11025:
		audioData.SoundRate = flv.SoundRate11025
	case 22050:
		audioData.SoundRate = flv.SoundRate22050
	case 44100:
		audioData.SoundRate = flv.SoundRate44100
	default:
		audioData.SoundRate = flv.SoundRate44100
	}

	if muxer.audioMeta.SampleSize == 8 {
		audioData.SoundSize = flv.SoundeSize8bit
	} else {
		audioData.SoundSize = flv.SoundeSize16bit
	}

	if muxer.audioMeta.Channels > 1 {
		audioData.SoundType = flv.SoundTypeStereo
	} else {
		audioData.SoundType = flv.SoundTypeMono
	}

	muxer.audioDataTemplate = audioData
}

func (muxer *flvMuxer) WriteFrame(frame *av.Frame) error {
	if muxer.baseNtp == 0 {
		muxer.baseNtp = frame.AbsTimestamp
	}

	if frame.FrameType == av.FrameVideo {
		return muxer.muxVideoTag(frame)
	} else {
		return muxer.muxAudioTag(frame)
	}
}

func (muxer *flvMuxer) muxVideoTag(frame *av.Frame) error {
	if frame.Payload[0]&0x1F == h264.NalSps {
		if len(muxer.videoMeta.Sps) == 0 {
			muxer.videoMeta.Sps = frame.Payload
		}
		return muxer.muxSequenceHeaderTag()
	}

	if frame.Payload[0]&0x1F == h264.NalPps {
		if len(muxer.videoMeta.Pps) == 0 {
			muxer.videoMeta.Pps = frame.Payload
		}
		return muxer.muxSequenceHeaderTag()
	}

	dts := time.Now().UnixNano()/int64(time.Millisecond) - muxer.baseTs
	pts := frame.AbsTimestamp - muxer.baseNtp + ptsDelay
	if dts > pts {
		pts = dts
	}

	videoData := &flv.VideoData{
		FrameType:       flv.FrameTypeInterFrame,
		CodecID:         flv.CodecIDAVC,
		AVCPacketType:   flv.AVCPacketTypeNALU,
		CompositionTime: uint32(pts - dts),
		Body:            frame.Payload,
	}

	if frame.Payload[0]&0x1F == h264.NalIdrSlice {
		videoData.FrameType = flv.FrameTypeKeyFrame
	}
	data, _ := videoData.Marshal()

	tag := &flv.Tag{
		TagType:   flv.TagTypeVideo,
		DataSize:  uint32(len(data)),
		Timestamp: uint32(dts),
		StreamID:  0,
		Data:      data,
	}

	return muxer.tagWriter.WriteTag(tag)
}

func (muxer *flvMuxer) muxAudioTag(frame *av.Frame) error {
	audioData := *muxer.audioDataTemplate
	audioData.Body = frame.Payload
	data, _ := audioData.Marshal()

	tag := &flv.Tag{
		TagType:   flv.TagTypeAudio,
		DataSize:  uint32(len(data)),
		Timestamp: uint32(frame.AbsTimestamp-muxer.baseNtp) + ptsDelay,
		StreamID:  0,
		Data:      data,
	}
	return muxer.tagWriter.WriteTag(tag)
}

func (muxer *flvMuxer) muxMetadataTag() error {
	properties := make(amf.EcmaArray, 0, 12)

	properties = append(properties,
		amf.ObjectProperty{
			Name:  "creator",
			Value: "ipchub stream media server"})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  flv.MetaDataCreationDate,
			Value: time.Now().Format(time.RFC3339)})

	if muxer.typeFlags&flv.TypeFlagsAudio > 0 {
		properties = append(properties,
			amf.ObjectProperty{
				Name:  flv.MetaDataAudioCodecID,
				Value: flv.SoundFormatAAC})
		properties = append(properties,
			amf.ObjectProperty{
				Name:  flv.MetaDataAudioDateRate,
				Value: muxer.audioMeta.DataRate})
		properties = append(properties,
			amf.ObjectProperty{
				Name:  flv.MetaDataAudioSampleRate,
				Value: muxer.audioMeta.SampleRate})
		properties = append(properties,
			amf.ObjectProperty{
				Name:  flv.MetaDataAudioSampleSize,
				Value: muxer.audioMeta.SampleSize})
		properties = append(properties,
			amf.ObjectProperty{
				Name:  flv.MetaDataStereo,
				Value: muxer.audioMeta.Channels > 1})
	}

	properties = append(properties,
		amf.ObjectProperty{
			Name:  flv.MetaDataVideoCodecID,
			Value: flv.CodecIDAVC})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  flv.MetaDataVideoDataRate,
			Value: muxer.videoMeta.DataRate})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  flv.MetaDataFrameRate,
			Value: muxer.videoMeta.FrameRate})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  flv.MetaDataWidth,
			Value: muxer.videoMeta.Width})
	properties = append(properties,
		amf.ObjectProperty{
			Name:  flv.MetaDataHeight,
			Value: muxer.videoMeta.Height})

	scriptData := flv.ScriptData{
		Name:  flv.ScriptOnMetaData,
		Value: properties,
	}
	data, _ := scriptData.Marshal()

	tag := &flv.Tag{
		TagType:   flv.TagTypeAmf0Data,
		DataSize:  uint32(len(data)),
		Timestamp: 0,
		StreamID:  0,
		Data:      data,
	}

	return muxer.tagWriter.WriteTag(tag)
}

func (muxer *flvMuxer) muxSequenceHeaderTag() error {
	if muxer.spsMuxed {
		return nil
	}

	if len(muxer.videoMeta.Sps) == 0 || len(muxer.videoMeta.Pps) == 0 {
		// not enough
		return nil
	}

	muxer.spsMuxed = true

	record := flv.NewAVCDecoderConfigurationRecord(muxer.videoMeta.Sps, muxer.videoMeta.Pps)
	body, _ := record.Marshal()

	videoData := &flv.VideoData{
		FrameType:       flv.FrameTypeKeyFrame,
		CodecID:         flv.CodecIDAVC,
		AVCPacketType:   flv.AVCPacketTypeSequenceHeader,
		CompositionTime: 0,
		Body:            body,
	}
	data, _ := videoData.Marshal()

	tag := &flv.Tag{
		TagType:   flv.TagTypeVideo,
		DataSize:  uint32(len(data)),
		Timestamp: 0,
		StreamID:  0,
		Data:      data,
	}

	if err := muxer.tagWriter.WriteTag(tag); err != nil {
		return err
	}

	return muxer.muxAudioSequenceHeaderTag()
}

func (muxer *flvMuxer) muxAudioSequenceHeaderTag() error {
	if muxer.typeFlags&flv.TypeFlagsAudio == 0 {
		return nil
	}

	audioData := *muxer.audioDataTemplate
	audioData.AACPacketType = flv.AACPacketTypeSequenceHeader
	audioData.Body = muxer.audioMeta.Sps
	data, _ := audioData.Marshal()

	tag := &flv.Tag{
		TagType:   flv.TagTypeAudio,
		DataSize:  uint32(len(data)),
		Timestamp: 0,
		StreamID:  0,
		Data:      data,
	}
	return muxer.tagWriter.WriteTag(tag)
}

func (muxer *flvMuxer) consume() {
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

	muxer.muxMetadataTag()
	muxer.muxSequenceHeaderTag()

	for !muxer.closed {
		pack := muxer.recvQueue.Pop()
		if pack == nil {
			if !muxer.closed {
				muxer.logger.Warn("flvmuxer:receive nil packet")
			}
			continue
		}

		packet := pack.(*rtp.Packet)
		if err := muxer.extractFuncs[int(packet.Channel)](packet); err != nil {
			muxer.logger.Errorf("flvmuxer: extract rtp frame error :%s", err.Error())
			// break
		}
	}
}

func (muxer *flvMuxer) Close() error {
	if muxer.closed {
		return nil
	}

	muxer.closed = true
	muxer.recvQueue.Signal()
	return nil
}

func (muxer *flvMuxer) WritePacket(packet *rtp.Packet) error {
	muxer.recvQueue.Push(packet)
	return nil
}

func (muxer *flvMuxer) TypeFlags() byte {
	return muxer.typeFlags
}

type FlvMuxer interface {
	TypeFlags() byte
	rtp.PacketWriter
	io.Closer
}

type emptyFlvMuxer struct{}

func (emptyFlvMuxer) TypeFlags() byte                      { return 0 }
func (emptyFlvMuxer) WritePacket(packet *rtp.Packet) error { return nil }
func (emptyFlvMuxer) Close() error                         { return nil }
