// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package buffered

import (
	"bufio"
	"bytes"
	"net"
	"time"

	"github.com/kelindar/rate"
)

const (
	defaultRate       = 50
	defaultBufferSize = 64 * 1024
	minBufferSize     = 8 * 1024
)

// Conn wraps a net.Conn and provides buffered ability.
type Conn struct {
	socket     net.Conn      // The underlying network connection.
	reader     *bufio.Reader // The buffered reader
	writer     *bytes.Buffer // The buffered write queue.
	limit      *rate.Limiter // The write rate limiter.
	bufferSize int           // The read and write max buffer size
}

// NewConn creates a new sniffed connection.
func NewConn(c net.Conn, options ...Option) *Conn {
	conn, ok := c.(*Conn)
	if !ok {
		conn = &Conn{
			socket: c,
		}
	}

	for _, option := range options {
		option.apply(conn)
	}

	// 设置默认值刷新频率
	if conn.limit == nil {
		conn.limit = rate.New(defaultRate, time.Second)
	}

	if conn.bufferSize <= 0 {
		conn.bufferSize = defaultBufferSize
	}

	// 设置IO缓冲对象
	conn.reader = bufio.NewReaderSize(conn.socket, conn.bufferSize)
	conn.writer = bytes.NewBuffer(make([]byte, 0, conn.bufferSize))
	return conn
}

// Buffered returns the pending buffer size.
func (m *Conn) Buffered() (n int) {
	return m.writer.Len()
}

// Reader 返回内部的 bufio.Reader
func (m *Conn) Reader() *bufio.Reader {
	return m.reader
}

// Flush flushes the underlying buffer by writing into the underlying connection.
func (m *Conn) Flush() (n int, err error) {
	if m.Buffered() == 0 {
		return 0, nil
	}

	// Flush everything and reset the buffer
	n, err = m.writeFull(m.writer.Bytes())
	m.writer.Reset()
	return
}

// Read reads the block of data from the underlying buffer.
func (m *Conn) Read(p []byte) (int, error) {
	return m.reader.Read(p)
}

// Write writes the block of data into the underlying buffer.
func (m *Conn) Write(p []byte) (nn int, err error) {
	var n int
	// 没有足够的空间容纳 p
	for len(p) > m.bufferSize-m.Buffered() && err == nil {
		if m.Buffered() == 0 {
			// Large write, empty buffer.
			// Write directly from p to avoid copy.
			n, err = m.socket.Write(p)
		} else {
			// write buffer to full state，and flush
			n, err = m.writer.Write(p[:m.bufferSize-m.writer.Len()])
			_, err = m.Flush()
		}
		nn += n
		p = p[n:]
	}

	if err != nil {
		return nn, err
	}

	// 未到达时间频率的间隔，直接写到缓存
	if m.limit.Limit() {
		n, err = m.writer.Write(p)
		return nn + n, err
	}

	// 缓存中有数据，flush
	if m.Buffered() > 0 {
		n, err = m.writer.Write(p)
		_, err = m.Flush()
		return nn + n, err
	}

	// 缓存中无数据，直接写避免内存拷贝
	n, err = m.writeFull(p)
	return nn + n, err

}

func (m *Conn) writeFull(p []byte) (nn int, err error) {
	var n int
	for len(p) > 0 && err == nil {
		n, err = m.socket.Write(p)
		nn += n
		p = p[n:]
	}
	return nn, err
}

// Close closes the connection. Any blocked Read or Write operations will be unblocked
// and return errors.
func (m *Conn) Close() error {
	return m.socket.Close()
}

// LocalAddr returns the local network address.
func (m *Conn) LocalAddr() net.Addr {
	return m.socket.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (m *Conn) RemoteAddr() net.Addr {
	return m.socket.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
func (m *Conn) SetDeadline(t time.Time) error {
	return m.socket.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
func (m *Conn) SetReadDeadline(t time.Time) error {
	return m.socket.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
func (m *Conn) SetWriteDeadline(t time.Time) error {
	return m.socket.SetWriteDeadline(t)
}

// Option 配置 Conn 的选项接口
type Option interface {
	apply(*Conn)
}

// OptionFunc 包装函数以便它满足 Option 接口
type optionFunc func(*Conn)

func (f optionFunc) apply(c *Conn) {
	f(c)
}

// FlushRate Conn 写操作的每秒刷新频率
func FlushRate(r int) Option {
	return optionFunc(func(c *Conn) {
		if r < 1 { // 如果不合规，设置成默认值
			r = defaultRate
		}
		c.limit = rate.New(r, time.Second)
	})
}

// BufferSize Conn 缓冲大小
func BufferSize(bufferSize int) Option {
	return optionFunc(func(c *Conn) {
		if bufferSize < minBufferSize { // 如果不合规，设置成最小值
			bufferSize = minBufferSize
		}
		c.bufferSize = bufferSize
	})
}
