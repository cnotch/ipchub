// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"runtime/debug"
	"time"

	"github.com/cnotch/ipchub/av"
	"github.com/cnotch/ipchub/av/h264"
	"github.com/cnotch/ipchub/protos/amf"
	"github.com/cnotch/queue"
	"github.com/cnotch/xlog"
)

// 网络播放时 PTS（Presentation Time Stamp）的延时
// 影响视频 Tag 的 CTS 和音频的 DTS（Decoding Time Stamp）
const (
	dtsDelay = 200
	ptsDelay = 1000
)

// MuxerAvcAac flv muxer from av.Frame(H264[+AAC])
type MuxerAvcAac struct {
	videoMeta         av.VideoMeta
	audioMeta         av.AudioMeta
	typeFlags         byte
	audioDataTemplate *AudioData
	recvQueue         *queue.SyncQueue
	tagWriter         TagWriter
	closed            bool
	spsMuxed          bool
	basePts           int64
	nextDts           float64
	dtsStep           float64
	logger            *xlog.Logger // 日志对象
}

// NewMuxerAvcAac .
func NewMuxerAvcAac(videoMeta av.VideoMeta, audioMeta av.AudioMeta, tagWriter TagWriter, logger *xlog.Logger) *MuxerAvcAac {
	muxer := &MuxerAvcAac{
		recvQueue: queue.NewSyncQueue(),
		videoMeta: videoMeta,
		audioMeta: audioMeta,
		typeFlags: byte(TypeFlagsVideo),
		tagWriter: tagWriter,
		closed:    false,
		nextDts:   dtsDelay,
		logger:    logger,
	}

	if videoMeta.FrameRate > 0 {
		muxer.dtsStep = 1000.0 / videoMeta.FrameRate
	}
	if audioMeta.Codec == "AAC" {
		muxer.typeFlags |= TypeFlagsAudio
		muxer.prepareTemplate()
	}

	go muxer.process()
	return muxer
}

// WriteFrame .
func (muxer *MuxerAvcAac) WriteFrame(frame *av.Frame) error {
	muxer.recvQueue.Push(frame)
	return nil
}

// Close .
func (muxer *MuxerAvcAac) Close() error {
	if muxer.closed {
		return nil
	}

	muxer.closed = true
	muxer.recvQueue.Signal()
	return nil
}

// TypeFlags 返回 flv header 中的 TypeFlags
func (muxer *MuxerAvcAac) TypeFlags() byte {
	return muxer.typeFlags
}

func (muxer *MuxerAvcAac) process() {
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
		f := muxer.recvQueue.Pop()
		if f == nil {
			if !muxer.closed {
				muxer.logger.Warn("flvmuxer:receive nil frame")
			}
			continue
		}

		frame := f.(*av.Frame)
		if muxer.basePts == 0 {
			muxer.basePts = frame.AbsTimestamp
		}

		if frame.FrameType == av.FrameVideo {
			if err := muxer.muxVideoTag(frame); err != nil {
				muxer.logger.Errorf("flvmuxer: muxVideoTag error - %s", err.Error())
			}
		} else {
			if err := muxer.muxAudioTag(frame); err != nil {
				muxer.logger.Errorf("flvmuxer: muxAudioTag error - %s", err.Error())
			}
		}
	}
}

func (muxer *MuxerAvcAac) muxVideoTag(frame *av.Frame) error {
	if frame.Payload[0]&0x1F == h264.NalSps {
		if len(muxer.videoMeta.Sps) == 0 {
			muxer.videoMeta.Sps = frame.Payload
			var rawSps h264.RawSPS
			err := rawSps.Decode(muxer.videoMeta.Sps)
			if err != nil {
				return err
			}

			muxer.videoMeta.Width = rawSps.Width()
			muxer.videoMeta.Height = rawSps.Height()
			muxer.videoMeta.FrameRate = rawSps.FrameRate()
			muxer.dtsStep = 1000.0 / muxer.videoMeta.FrameRate
		}
		return muxer.muxSequenceHeaderTag()
	}

	if frame.Payload[0]&0x1F == h264.NalPps {
		if len(muxer.videoMeta.Pps) == 0 {
			muxer.videoMeta.Pps = frame.Payload
		}
		return muxer.muxSequenceHeaderTag()
	}

	dts := int64(muxer.nextDts)
	muxer.nextDts += muxer.dtsStep
	pts := frame.AbsTimestamp - muxer.basePts + ptsDelay
	if dts > pts {
		pts = dts
	}

	videoData := &VideoData{
		FrameType:       FrameTypeInterFrame,
		CodecID:         CodecIDAVC,
		AVCPacketType:   AVCPacketTypeNALU,
		CompositionTime: uint32(pts - dts),
		Body:            frame.Payload,
	}

	if frame.Payload[0]&0x1F == h264.NalIdrSlice {
		videoData.FrameType = FrameTypeKeyFrame
	}
	data, _ := videoData.Marshal()

	tag := &Tag{
		TagType:   TagTypeVideo,
		DataSize:  uint32(len(data)),
		Timestamp: uint32(dts),
		StreamID:  0,
		Data:      data,
	}

	return muxer.tagWriter.WriteTag(tag)
}

func (muxer *MuxerAvcAac) muxAudioTag(frame *av.Frame) error {
	audioData := *muxer.audioDataTemplate
	audioData.Body = frame.Payload
	data, _ := audioData.Marshal()

	tag := &Tag{
		TagType:   TagTypeAudio,
		DataSize:  uint32(len(data)),
		Timestamp: uint32(frame.AbsTimestamp-muxer.basePts) + ptsDelay,
		StreamID:  0,
		Data:      data,
	}
	return muxer.tagWriter.WriteTag(tag)
}

func (muxer *MuxerAvcAac) muxMetadataTag() error {
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

	properties = append(properties,
		amf.ObjectProperty{
			Name:  MetaDataVideoCodecID,
			Value: CodecIDAVC})
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

	return muxer.tagWriter.WriteTag(tag)
}

func (muxer *MuxerAvcAac) muxSequenceHeaderTag() error {
	if muxer.spsMuxed {
		return nil
	}

	if len(muxer.videoMeta.Sps) == 0 || len(muxer.videoMeta.Pps) == 0 {
		// not enough
		return nil
	}

	muxer.spsMuxed = true

	record := NewAVCDecoderConfigurationRecord(muxer.videoMeta.Sps, muxer.videoMeta.Pps)
	body, _ := record.Marshal()

	videoData := &VideoData{
		FrameType:       FrameTypeKeyFrame,
		CodecID:         CodecIDAVC,
		AVCPacketType:   AVCPacketTypeSequenceHeader,
		CompositionTime: 0,
		Body:            body,
	}
	data, _ := videoData.Marshal()

	tag := &Tag{
		TagType:   TagTypeVideo,
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

func (muxer *MuxerAvcAac) muxAudioSequenceHeaderTag() error {
	if muxer.typeFlags&TypeFlagsAudio == 0 {
		return nil
	}

	audioData := *muxer.audioDataTemplate
	audioData.AACPacketType = AACPacketTypeSequenceHeader
	audioData.Body = muxer.audioMeta.Sps
	data, _ := audioData.Marshal()

	tag := &Tag{
		TagType:   TagTypeAudio,
		DataSize:  uint32(len(data)),
		Timestamp: 0,
		StreamID:  0,
		Data:      data,
	}
	return muxer.tagWriter.WriteTag(tag)
}

func (muxer *MuxerAvcAac) prepareTemplate() {
	audioData := &AudioData{
		SoundFormat:   SoundFormatAAC,
		AACPacketType: AACPacketTypeRawData,
		Body:          nil,
	}

	switch muxer.audioMeta.SampleRate {
	case 5512:
		audioData.SoundRate = SoundRate5512
	case 11025:
		audioData.SoundRate = SoundRate11025
	case 22050:
		audioData.SoundRate = SoundRate22050
	case 44100:
		audioData.SoundRate = SoundRate44100
	default:
		audioData.SoundRate = SoundRate44100
	}

	if muxer.audioMeta.SampleSize == 8 {
		audioData.SoundSize = SoundeSize8bit
	} else {
		audioData.SoundSize = SoundeSize16bit
	}

	if muxer.audioMeta.Channels > 1 {
		audioData.SoundType = SoundTypeStereo
	} else {
		audioData.SoundType = SoundTypeMono
	}

	muxer.audioDataTemplate = audioData
}
