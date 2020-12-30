// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"

	"github.com/pion/rtp"
)

const (
	// TransferPrefix RTP 包网络传输时的前缀
	TransferPrefix = byte(0x24) // $
)

// 预定义 RTP 通道类型
const (
	ChannelVideo        = iota         // 视频通道
	ChannelVideoControl                // 视频控制通道
	ChannelAudio                       // 音频通道
	ChannelAudioControl                // 音频控制通道
	ChannelCount                       // 支持的 RTP 通道类型数量
	ChannelMin          = ChannelVideo // 支持的 RTP 通道类型最小值
)

// DefaultChannelConfig 默认的通道配置
var DefaultChannelConfig = []int{
	ChannelVideo,
	ChannelVideoControl,
	ChannelAudio,
	ChannelAudioControl,
}

// ChannelName 通道名
func ChannelName(channel int) string {
	switch channel {
	case ChannelAudio:
		return "audio"
	case ChannelVideo:
		return "video"
	case ChannelAudioControl:
		return "audio control"
	case ChannelVideoControl:
		return "video control"
	}
	return "unknow"
}

// Packet RTP 数据包
type Packet struct {
	Channel    byte   // 通道
	Data       []byte // 数据
	rtp.Header        // Video 、Audio Channel'Header
}

// PacketWriter 包装 WritePacket 方法的接口
type PacketWriter interface {
	WriteRtpPacket(packet *Packet) error
}

// ReadPacket 根据规范从 r 中读取 rtp 包.
// channelConfig 提供通道类型所在通道的配置信息
func ReadPacket(r *bufio.Reader, channelConfig []int) (*Packet, error) {
	var err error

	var prefix [4]byte
	// 读前缀4字节
	if _, err = io.ReadFull(r, prefix[:]); err != nil {
		return nil, err
	}

	if prefix[0] != TransferPrefix {
		return nil, errors.New("RTP Pack must start with `$`")
	}

	channel := int(prefix[1])
	rtpLen := int(binary.BigEndian.Uint16(prefix[2:]))

	// 读取包数据
	rtpBytes := make([]byte, rtpLen)
	if _, err = io.ReadFull(r, rtpBytes); err != nil {
		return nil, err
	}

	var p = new(Packet)
	p.Data = rtpBytes
	for i, v := range channelConfig {
		if v == channel {
			p.Channel = byte(i)
			if p.Channel == ChannelVideo || p.Channel == ChannelAudio {
				if err = p.Header.Unmarshal(p.Data); err != nil {
					return nil, err
				}
			}
			return p, nil
		}
	}
	return nil, errors.New("RTP Packet illegal channel")
}

// Write 根据规范将 RTP 包输出到 w
// channelConfig 提供通道类型所在通道的配置信息
func (p *Packet) Write(w io.Writer, channelConfig []int) error {
	if p.Channel >= ChannelCount {
		return errors.New("unknow pack type")
	}

	ch := channelConfig[p.Channel]
	if ch < 0 || ch > 255 { // 可能是未订阅，忽略
		return nil
	}

	var prefix [4]byte
	prefix[0] = TransferPrefix // 起始字节
	prefix[1] = byte(ch)       // channel
	binary.BigEndian.PutUint16(prefix[2:], uint16(len(p.Data)))

	// 写前4个字节
	if _, err := w.Write(prefix[:]); err != nil {
		return err
	}

	// 写包数据部分
	if _, err := w.Write(p.Data); err != nil {
		return err
	}

	return nil
}

// Size 包在 RTP 中的传输总大小
func (p *Packet) Size() int {
	return len(p.Data) + 4
}

// Payload 数据包中实际的载荷
// 如果是控制通道，返回nil
func (p *Packet) Payload() []byte {
	if p.Channel == ChannelVideo || p.Channel == ChannelAudio {
		return p.Data[p.PayloadOffset:]
	}
	return nil
}
