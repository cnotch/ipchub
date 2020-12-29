// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bytes"
	"errors"
	"fmt"
	"net"

	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/ipchub/network"
	"github.com/cnotch/ipchub/utils"
	"github.com/cnotch/xlog"
)

var (
	errModeBehavior                = errors.New("Play mode can't send rtp pack")
	defaultStream   mediaStream    = emptyStream{}
	defaultConsumer media.Consumer = emptyConsumer{}
)

// 媒体流
type mediaStream interface {
	Close() error
	WritePacket(pack *RTPPack) error
}

// 占位流，简化判断
type emptyStream struct {
}

func (s emptyStream) Close() error               { return nil }
func (s emptyStream) WritePacket(*RTPPack) error { return errModeBehavior }

// 占位消费者，简化判断
type emptyConsumer struct {
}

func (c emptyConsumer) Consume(p Pack) {}
func (c emptyConsumer) Close() error   { return nil }

type tcpPushStream struct {
	closed bool
	stream *media.Stream
}

func (s *tcpPushStream) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true
	media.Unregist(s.stream)
	s.stream = nil
	return nil
}

func (s *tcpPushStream) WritePacket(p *RTPPack) error {
	return s.stream.WritePacket(p)
}

type tcpConsumer struct {
	*Session
	closed bool
	source *media.Stream
	cid    media.CID
}

func (c *tcpConsumer) Consume(p Pack) {
	if c.closed {
		return
	}

	p2 := p.(*RTPPack)
	var err error

	if c.wsconn != nil {
		buf := buffers.Get().(*bytes.Buffer)
		buf.Reset()
		defer buffers.Put(buf)

		p2.Write(buf, c.transport.Channels[:])

		c.lockW.Lock()
		_, err = c.wsconn.Write(buf.Bytes())
		c.lockW.Unlock()
	} else {
		c.lockW.Lock()
		err = p2.Write(c.conn, c.transport.Channels[:])
		c.lockW.Unlock()
	}

	if err != nil {
		c.logger.Errorf("send pack error = %v , close socket", err)
		c.Close()
		return
	}
}

func (c *tcpConsumer) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	c.source.StopConsume(c.cid)
	c.source = nil
	return nil
}

type udpConsumer struct {
	*Session
	closed   bool
	source   *media.Stream
	cid      media.CID
	udpConn  *net.UDPConn // 用于Player的UDP单播
	destAddr [rtpChannelCount]*net.UDPAddr
}

func (c *udpConsumer) Consume(p Pack) {
	if c.closed {
		return
	}

	p2 := p.(*RTPPack)
	addr := c.destAddr[int(p2.Channel)]
	if addr != nil {
		_, err := c.udpConn.WriteToUDP(p2.Data, addr)
		if err != nil {
			c.logger.Warn(err.Error())
			return
		}
	}
}

func (c *udpConsumer) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true

	c.source.StopConsume(c.cid)
	c.udpConn.Close()
	c.source = nil
	return nil
}

func (c *udpConsumer) prepareUDP(destIP string, destPorts [rtpChannelCount]int) error {
	// 如果还没准备 Socket
	if c.udpConn == nil {
		udpConn, err := net.ListenUDP("udp", &net.UDPAddr{})
		if err != nil {
			return err
		}
		c.udpConn = udpConn
		err = udpConn.SetWriteBuffer(config.NetBufferSize())
	}

	for i, port := range destPorts {
		if port > 0 {
			c.destAddr[i], _ = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", destIP, port))
		}
	}
	return nil
}

type multicastConsumer struct {
	*Session
	closed bool
	source *media.Stream
}

func (c *multicastConsumer) Consume(p Pack) {}
func (c *multicastConsumer) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true

	c.source.Multicastable().ReleaseMember(c.Session)
	c.source = nil
	c.Session = nil
	return nil
}

// 将Session作为Pusher角色
func (s *Session) asTCPPusher() {
	pusher := &tcpPushStream{}

	mproxy := &multicastProxy{
		path:        s.path,
		bufferSize:  config.NetBufferSize(),
		multicastIP: utils.Multicast.NextIP(), // 设置组播IP
		ttl:         config.MulticastTTL(),
		logger: s.logger.With(xlog.Fields(
			xlog.F("path", s.path),
			xlog.F("type", "multicast-proxy"))),
	}

	s.logger = s.logger.With(xlog.Fields(
		xlog.F("path", s.path),
		xlog.F("type", "pusher")))

	for i := rtpChannelMin; i < rtpChannelCount; i++ {
		mproxy.ports[i] = utils.Multicast.NextPort()
	}

	pusher.stream = media.NewStream(s.path, s.rawSdp,
		media.Attr("addr", s.conn.RemoteAddr().String()),
		media.Multicast(mproxy))

	media.Regist(pusher.stream)
	// 设置Session字段
	s.stream = pusher
}

func (s *Session) asTCPConsumer(stream *media.Stream, resp *Response) (err error) {
	if s.wsconn != nil {
		s.logger = s.logger.With(xlog.Fields(
			xlog.F("path", s.path),
			xlog.F("type", "websocket-player")))
	} else {
		s.logger = s.logger.With(xlog.Fields(
			xlog.F("path", s.path),
			xlog.F("type", "tcp-player")))
	}

	c := &tcpConsumer{
		Session: s,
		source:  stream,
	}

	err = s.response(resp)
	if err != nil {
		return err
	}
	s.timeout = 0 // play 只需发送不用接收，因此设置不超时
	s.consumer = c
	if s.wsconn != nil {
		c.cid = stream.StartConsumeNoGopCache(s, media.RTPPacket, "net=rtsp-websocket")
	} else {
		c.cid = stream.StartConsume(s, media.RTPPacket, "net=rtsp-tcp")
	}
	return
}

func (s *Session) asUDPConsumer(stream *media.Stream, resp *Response) (err error) {
	c := &udpConsumer{
		Session: s,
		source:  stream,
	}

	// 创建udp连接
	err = c.prepareUDP(network.GetIP(s.conn.RemoteAddr()), s.transport.ClientPorts)
	if err != nil {
		resp.StatusCode = StatusInternalServerError
		err = s.response(resp)
		if err != nil {
			return err
		}
		return nil
	}

	s.logger = s.logger.With(xlog.Fields(
		xlog.F("path", s.path),
		xlog.F("type", "udp-player")))
	err = s.response(resp)
	if err != nil {
		return err
	}

	s.timeout = 0 // play 只需发送不用接收，因此设置不超时
	s.consumer = c

	c.cid = stream.StartConsume(s, media.RTPPacket, "net=rtsp-udp")
	return nil
}

func (s *Session) asMulticastConsumer(stream *media.Stream, resp *Response) (err error) {
	c := &multicastConsumer{
		Session: s,
		source:  stream,
	}
	ma := stream.Multicastable()
	if ma == nil { // 不支持组播
		resp.StatusCode = StatusUnsupportedTransport
		err = s.response(resp)
		if err != nil {
			return err
		}
		return nil
	}

	s.logger = s.logger.With(xlog.Fields(
		xlog.F("path", s.path),
		xlog.F("type", "multicast-player")))
	err = s.response(resp)
	if err != nil {
		return err
	}

	c.timeout = 0 // play 只需发送不用接收，因此设置不超时
	s.consumer = c

	ma.AddMember(s)
	return nil
}
