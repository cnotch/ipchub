// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package flv

import (
	"bytes"

	"github.com/cnotch/ipchub/protos/amf"
)

// 数据名称常量，如元数据
const (
	ScriptOnMetaData = "onMetaData"
)

// MetaData 常见属性名
const (
	MetaDataAudioCodecID    = "audiocodecid"    // Number	音频编解码器 ID
	MetaDataAudioDateRate   = "audiodatarate"   // Number	音频码率，单位 kbps
	MetaDataAudioDelay      = "audiodelay"      // Number	由音频编解码器引入的延时，单位秒
	MetaDataAudioSampleRate = "audiosamplerate" // Number	音频采样率
	MetaDataAudioSampleSize = "audiosamplesize" // Number	音频采样点尺寸
	MetaDataStereo          = "stereo"          // Boolean	音频立体声标志
	MetaDataCanSeekToEnd    = "canSeekToEnd"    // Boolean	指示最后一个视频帧是否是关键帧
	MetaDataCreationDate    = "creationdate"    // String	创建日期与时间
	MetaDataDuration        = "duration"        // Number	文件总时长，单位秒
	MetaDataFileSize        = "filesize"        // Number	文件总长度，单位字节
	MetaDataFrameRate       = "framerate"       // Number	视频帧率
	MetaDataHeight          = "height"          // Number	视频高度，单位像素
	MetaDataVideoCodecID    = "videocodecid"    // Number	视频编解码器 ID
	MetaDataVideoDataRate   = "videodatarate"   // Number	视频码率，单位 kbps
	MetaDataWidth           = "width"           // Number	视频宽度，单位像素
)

// ScriptData .
type ScriptData struct {
	Name  string
	Value interface{}
}

// Unmarshal .
func (scriptData *ScriptData) Unmarshal(data []byte) (err error) {
	buff := bytes.NewBuffer(data)

	if scriptData.Name, err = amf.ReadString(buff); err != nil {
		return
	}

	scriptData.Value, err = amf.ReadAny(buff)
	return
}

// Marshal .
func (scriptData *ScriptData) Marshal() ([]byte, error) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))

	if err := amf.WriteString(buff, scriptData.Name); err != nil {
		return nil, err
	}

	if err := amf.WriteAny(buff, scriptData.Value); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}
