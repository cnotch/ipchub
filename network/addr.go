// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/emitter-io/address"
)

// GetIP 获取IP信息
func GetIP(addr net.Addr) string {
	s := addr.String()
	i := strings.LastIndex(s, ":")
	return s[:i]
}

// GetLocalIP 获取本地IP
func GetLocalIP() []string {
	addrs, _ := net.InterfaceAddrs()
	ips := []string{}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

// IsLocalhostIP 判断是否为本机IP
func IsLocalhostIP(ip net.IP) bool {
	for _, localhost := range loopbackBlocks {
		if localhost.Contains(ip) {
			return true
		}
	}
	privs, err := address.GetPrivate()
	if err != nil {
		return false
	}

	for _, priv := range privs {
		if priv.IP.Equal(ip) {
			return true
		}
	}

	return false
}

var loopbackBlocks = []*net.IPNet{
	parseCIDR("0.0.0.0/8"),   // RFC 1918 IPv4 loopback address
	parseCIDR("127.0.0.0/8"), // RFC 1122 IPv4 loopback address
	parseCIDR("::1/128"),     // RFC 1884 IPv6 loopback address
}

func parseCIDR(s string) *net.IPNet {
	_, block, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("Bad CIDR %s: %s", s, err))
	}
	return block
}
