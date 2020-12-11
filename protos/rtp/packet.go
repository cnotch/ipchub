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

// Channel RTP 通道类型
type Channel byte

// 预定义 RTP 通道类型
const (
	ChannelVideo        = Channel(iota) // 视频通道
	ChannelVideoControl                 // 视频控制通道
	ChannelAudio                        // 音频通道
	ChannelAudioControl                 // 音频控制通道
	ChannelCount                        // 支持的 RTP 通道类型数量
	ChannelMin          = ChannelVideo  // 支持的 RTP 通道类型最小值
)

func (ct Channel) String() string {
	switch ct {
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
	Channel    Channel // 通道
	Data       []byte  // 数据
	rtp.Header         // Video 、Audio Channel'Header
}

func (p *Packet) decode(prefix [4]byte, r *bufio.Reader, channels []int) error {
	var err error
	if prefix[0] != TransferPrefix {
		return errors.New("RTP Pack must start with `$`")
	}
	channel := int(prefix[1])
	rtpLen := int(binary.BigEndian.Uint16(prefix[2:]))

	// 读取包数据
	rtpBytes := make([]byte, rtpLen)
	if _, err = io.ReadFull(r, rtpBytes); err != nil {
		return err
	}

	p.Data = rtpBytes
	for i, v := range channels {
		if v == channel {
			p.Channel = Channel(i)
			if p.Channel == ChannelVideo || p.Channel == ChannelAudio {
				if err = p.Header.Unmarshal(p.Data); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return errors.New("RTP Packet illegal channel")

}

// Decode 根据规范从 r 中解码.
// channels 包含包类型对应的通道信息，以便提取通道类型
func (p *Packet) Decode(r *bufio.Reader, channels []int) error {
	var err error

	var prefix [4]byte
	// 读前缀4字节
	if _, err = io.ReadFull(r, prefix[:]); err != nil {
		return err
	}
	return p.decode(prefix, r, channels)
}

// Len 返回包在 RTP 中的传输长度
func (p *Packet) Len() int {
	return len(p.Data) + 4
}

// EncodeTo 根据规范将 RTP 包编码到 w
func (p *Packet) EncodeTo(w io.Writer, channels []int) error {
	if p.Channel >= ChannelCount {
		return errors.New("unknow pack type")
	}

	ch := channels[p.Channel]
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

// Payload 数据包中实际的载荷
// 如果是控制通道，返回nil
func (p *Packet) Payload() []byte {
	if p.Channel == ChannelVideo || p.Channel == ChannelAudio {
		return p.Data[p.PayloadOffset:]
	}
	return nil
}
