// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"errors"
	"strconv"
	"strings"

	"github.com/cnotch/ipchub/utils/scan"
)

// RTPTransportType RTP 传输模式
type RTPTransportType int

// SessionMode 会话模式
type SessionMode int

// 通讯类型
const (
	RTPUnknownTrans RTPTransportType = iota
	RTPTCPUnicast                    // TCP
	RTPUDPUnicast                    // UDP
	RTPMulticast                     // 组播
)

// 会话类型
const (
	UnknownSession SessionMode = iota
	PlaySession                // 播放
	RecordSession              // 录像
)

// RTPTransport RTP传输设置
type RTPTransport struct {
	Mode        SessionMode
	Append      bool
	Type        RTPTransportType
	Channels    [rtpChannelCount]int
	ClientPorts [rtpChannelCount]int
	ServerPorts [rtpChannelCount]int

	// 组播相关设置
	Ports       [rtpChannelCount]int // 组播端口
	MulticastIP string            // 组播地址 224.0.1.0～238.255.255.255
	TTL         int
	Source      string // 组播源地址
}

func parseRange(p string) (begin int, end int) {
	begin = -1
	end = -1

	var s1, s2 string
	index := strings.IndexByte(p, '-')
	if index < 0 {
		s1 = p
	} else {
		s1 = strings.TrimSpace(p[:index])
		s2 = strings.TrimSpace(p[index+1:])
	}
	var err error
	if len(s1) > 0 {
		begin, err = strconv.Atoi(s1)
		if err != nil {
			begin = -1
		}
	}
	if len(s2) > 0 {
		end, err = strconv.Atoi(s2)
		if err != nil {
			end = -1
		}
	}
	return
}

// ParseTransport 解析Setup中的传输配置
func (t *RTPTransport) ParseTransport(rtpType int, ts string) (err error) {
	if t.Mode == UnknownSession {
		t.Mode = PlaySession
	}

	// 确定传输类型
	index := strings.IndexByte(ts, ';')
	if index < 0 {
		return errors.New("malformed trannsport")
	}
	transportSpec := strings.TrimSpace(ts[:index])
	ts = ts[index+1:]

	if transportSpec == "RTP/AVP/TCP" {
		t.Type = RTPTCPUnicast
	} else if transportSpec == "RTP/AVP" || transportSpec == "RTP/AVP/UDP" {
		t.Type = RTPMulticast // 默认组播
	} else {
		return errors.New("malformed trannsport")
	}

	// 扫描参数
	advance := ts
	token := ""
	continueScan := true
	for continueScan {
		advance, token, continueScan = scan.Semicolon.Scan(advance)
		if token == "unicast" && t.Type == RTPMulticast {
			t.Type = RTPUDPUnicast
			continue
		}
		if token == "multicast" && t.Type == RTPTCPUnicast {
			err = errors.New("malformed trannsport")
			continue
		}
		if token == "append" {
			t.Append = true
			continue
		}

		k, v, _ := scan.EqualPair.Scan(token)
		switch k {
		case "mode":
			if v == "record" {
				t.Mode = RecordSession
			} else {
				t.Mode = PlaySession
			}
		case "interleaved":
			begin, end := parseRange(v)
			if begin >= 0 {
				t.Channels[rtpType] = begin
			}
			if end >= 0 {
				t.Channels[rtpType+1] = end
			}
			if begin < 0 {
				err = errors.New("malformed trannsport")
			}
		case "client_port":
			t.ClientPorts[rtpType], t.ClientPorts[rtpType+1] = parseRange(v)
			if t.ClientPorts[rtpType] < 0 {
				err = errors.New("malformed trannsport")
			}
		case "server_port":
			t.ServerPorts[rtpType], t.ServerPorts[rtpType+1] = parseRange(v)
			if t.ServerPorts[rtpType] < 0 {
				err = errors.New("malformed trannsport")
			}
		case "port":
			t.Ports[rtpType], t.Ports[rtpType+1] = parseRange(v)
			if t.Ports[rtpType] < 0 {
				err = errors.New("malformed trannsport")
			}
		case "destination":
			t.MulticastIP = v
		case "source":
			t.Source = v
		case "ttl":
			t.TTL, _ = strconv.Atoi(v)
		}
	}
	return
}
