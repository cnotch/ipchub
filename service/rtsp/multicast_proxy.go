// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/xlog"
	"github.com/emitter-io/address"
)

// 组播代理
type multicastProxy struct {
	// 创建时设置
	logger     *xlog.Logger
	path       string
	bufferSize int

	multicastIP string
	ports       [rtpChannelCount]int
	ttl         int
	sourceIP    string

	closed   bool
	udpConn  *net.UDPConn
	destAddr [rtpChannelCount]*net.UDPAddr
	cid      media.CID

	multicastLock sync.Mutex
	members       []io.Closer
}

func (proxy *multicastProxy) AddMember(m io.Closer) {
	proxy.multicastLock.Lock()
	defer proxy.multicastLock.Unlock()

	if len(proxy.members) == 0 {
		stream := media.Get(proxy.path)
		if stream == nil {
			proxy.logger.Error("start multicast proxy failed.")
			return
		}

		udpConn, err := net.ListenUDP("udp", &net.UDPAddr{})
		if err != nil {
			proxy.logger.Errorf("start multicast proxy failed. %s", err.Error())
			return
		}

		proxy.udpConn = udpConn
		err = udpConn.SetWriteBuffer(proxy.bufferSize)

		for i, port := range proxy.ports {
			if port > 0 {
				proxy.destAddr[i], _ = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", proxy.multicastIP, proxy.ports[i]))
			}
		}

		proxy.members = append(proxy.members, m)
		proxy.cid = stream.StartConsume(proxy, media.RTPPacket,
			"net = rtsp-multicast, "+proxy.multicastIP)
		proxy.closed = false

		proxy.logger.Info("multicast proxy started.")
	}
}

func (proxy *multicastProxy) ReleaseMember(m io.Closer) {
	proxy.multicastLock.Lock()
	defer proxy.multicastLock.Unlock()
	for i, m2 := range proxy.members {
		if m == m2 {
			proxy.members = append(proxy.members[:i], proxy.members[i+1:]...)
			break
		}
	}

	if len(proxy.members) == 0 {
		// 停止组播代理
		proxy.close()
	}
}

func (proxy *multicastProxy) MulticastIP() string {
	return proxy.multicastIP
}

func (proxy *multicastProxy) Port(index int) int {
	if index < 0 || index > len(proxy.ports) {
		return 0
	}
	return proxy.ports[index]
}

func (proxy *multicastProxy) TTL() int {
	return proxy.ttl
}

func (proxy *multicastProxy) SourceIP() string {
	if len(proxy.sourceIP) == 0 {
		addrs, err := address.GetPublic()
		if err != nil {
			proxy.sourceIP = "Unknown"
		} else {
			proxy.sourceIP = addrs[0].IP.String()
		}
	}
	return proxy.sourceIP
}

func (proxy *multicastProxy) Consume(p Pack) {
	if proxy.closed {
		return
	}

	p2 := p.(*RTPPack)
	addr := proxy.destAddr[int(p2.Channel)]
	if addr != nil {
		_, err := proxy.udpConn.WriteToUDP(p2.Data, addr)
		if err != nil {
			proxy.logger.Error(err.Error())
			return
		}
	}
}

func (proxy *multicastProxy) Close() error {
	proxy.multicastLock.Lock()
	defer proxy.multicastLock.Unlock()

	proxy.close()
	return nil
}

func (proxy *multicastProxy) close() {
	if proxy.closed {
		return
	}
	proxy.closed = true

	stream := media.Get(proxy.path)
	if stream != nil {
		stream.StopConsume(proxy.cid)
	}

	if proxy.udpConn != nil {
		proxy.udpConn.Close()
		proxy.udpConn = nil
	}

	// 关闭所有的组播客户端
	for _, m := range proxy.members {
		m.Close()
	}
	proxy.members = nil

	for i := range proxy.destAddr {
		proxy.destAddr[i] = nil
	}

	proxy.logger.Info("multicast proxy stopped.")
}
