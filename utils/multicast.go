// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package utils

import (
	"encoding/binary"
	"net"
	"sync"
)

// MulticastIPS 全局组播池
var (
	Multicast = &multicast{
		ipseed:   minIP,
		portseed: minPort,
	}
	minIP          = binary.BigEndian.Uint32([]byte{235, 0, 0, 0})
	maxIP          = binary.BigEndian.Uint32([]byte{235, 255, 255, 255})
	minPort uint16 = 16666
	maxPort uint16 = 39999
)

// multicast 组播IP地址池
type multicast struct {
	ipseed   uint32
	portseed uint16
	l        sync.Mutex
}

// NextIP 获取组播地址
func (p *multicast) NextIP() string {
	p.l.Lock()
	defer p.l.Unlock()
	var ipbytes [4]byte
	binary.BigEndian.PutUint32(ipbytes[:], p.ipseed)
	ip := net.IP(ipbytes[:]).String()
	p.ipseed++
	if p.ipseed > maxIP {
		p.ipseed = minIP
	}
	return ip
}

func (p *multicast) NextPort() int {
	p.l.Lock()
	defer p.l.Unlock()
	port := p.portseed
	p.portseed++
	if p.portseed > maxPort {
		p.portseed = minPort
	}
	return int(port)
}
