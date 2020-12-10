/**********************************************************************************
* Copyright (c) 2009-2017 Misakai Ltd.
* This program is free software: you can redistribute it and/or modify it under the
* terms of the GNU Affero General Public License as published by the  Free Software
* Foundation, either version 3 of the License, or(at your option) any later version.
*
* This program is distributed  in the hope that it  will be useful, but WITHOUT ANY
* WARRANTY;  without even  the implied warranty of MERCHANTABILITY or FITNESS FOR A
* PARTICULAR PURPOSE.  See the GNU Affero General Public License  for  more details.
*
* You should have  received a copy  of the  GNU Affero General Public License along
* with this program. If not, see<http://www.gnu.org/licenses/>.
************************************************************************************/
//
// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package websocket

import (
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Conn websocket连接
type Conn interface {
	net.Conn
	Subprotocol() string // 获取子协议
	TextTransport() Conn // 获取文本传输通道
	Path() string        // 接入时的ws后的路径
	Username() string    // 接入是http验证后的用户名称
}

type websocketConn interface {
	NextReader() (messageType int, r io.Reader, err error)
	NextWriter(messageType int) (io.WriteCloser, error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Subprotocol() string
}

// websocketConn represents a websocket connection.
type websocketTransport struct {
	sync.Mutex
	socket   websocketConn
	reader   io.Reader
	closing  chan bool
	path     string
	username string
}

const (
	writeWait        = 10 * time.Second    // Time allowed to write a message to the peer.
	pongWait         = 60 * time.Second    // Time allowed to read the next pong message from the peer.
	pingPeriod       = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.
	closeGracePeriod = 10 * time.Second    // Time to wait before force close on connection.
)

// The default upgrader to use
var upgrader = &websocket.Upgrader{
	Subprotocols: []string{"rtsp", "control", "data"},
	CheckOrigin:  func(r *http.Request) bool { return true },
	// ReadBufferSize: 64 * 1024, WriteBufferSize: 64 * 1024,
}

// TryUpgrade attempts to upgrade an HTTP request to rtsp/wsp over websocket.
func TryUpgrade(w http.ResponseWriter, r *http.Request, path, username string) (Conn, bool) {
	if w == nil || r == nil {
		return nil, false
	}

	if ws, err := upgrader.Upgrade(w, r, nil); err == nil {
		return newConn(ws, path, username), true
	}

	return nil, false
}

// newConn creates a new transport from websocket.
func newConn(ws websocketConn, path, username string) Conn {
	conn := &websocketTransport{
		socket:   ws,
		closing:  make(chan bool),
		path:     path,
		username: username,
	}

	/*ws.SetReadLimit(maxMessageSize)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	ws.SetCloseHandler(func(code int, text string) error {
		return conn.Close()
	})

	utils.Repeat(func() {
		log.Println("ping")
		if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
			log.Println("ping:", err)
		}
	}, pingPeriod, conn.closing)*/

	return conn
}

// Read reads data from the connection. It is possible to allow reader to time
// out and return a Error with Timeout() == true after a fixed time limit by
// using SetDeadline and SetReadDeadline on the websocket.
func (c *websocketTransport) Read(b []byte) (n int, err error) {
	var opCode int
	if c.reader == nil {
		// New message
		var r io.Reader
		for {
			if opCode, r, err = c.socket.NextReader(); err != nil {
				return
			}

			if opCode != websocket.BinaryMessage && opCode != websocket.TextMessage {
				continue
			}

			c.reader = r
			break
		}
	}

	// Read from the reader
	n, err = c.reader.Read(b)
	if err != nil {
		if err == io.EOF {
			c.reader = nil
			err = nil
		}
	}
	return
}

// Write writes data to the connection. It is possible to allow writer to time
// out and return a Error with Timeout() == true after a fixed time limit by
// using SetDeadline and SetWriteDeadline on the websocket.
func (c *websocketTransport) Write(b []byte) (n int, err error) {
	// Serialize write to avoid concurrent write
	c.Lock()
	defer c.Unlock()

	var w io.WriteCloser
	if w, err = c.socket.NextWriter(websocket.BinaryMessage); err == nil {
		if n, err = w.Write(b); err == nil {
			err = w.Close()
		}
	}
	return
}

// Close terminates the connection.
func (c *websocketTransport) Close() error {
	return c.socket.Close()
}

// LocalAddr returns the local network address.
func (c *websocketTransport) LocalAddr() net.Addr {
	return c.socket.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *websocketTransport) RemoteAddr() net.Addr {
	return c.socket.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
func (c *websocketTransport) SetDeadline(t time.Time) (err error) {
	if err = c.socket.SetReadDeadline(t); err == nil {
		err = c.socket.SetWriteDeadline(t)
	}
	return
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
func (c *websocketTransport) SetReadDeadline(t time.Time) error {
	return c.socket.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
func (c *websocketTransport) SetWriteDeadline(t time.Time) error {
	return c.socket.SetWriteDeadline(t)
}

// Subprotocol 获取子协议名称
func (c *websocketTransport) Subprotocol() string {
	return c.socket.Subprotocol()
}

// TextTransport 获取文本传输Conn
func (c *websocketTransport) TextTransport() Conn {
	return &websocketTextTransport{c}
}

func (c *websocketTransport) Path() string {
	return c.path
}

func (c *websocketTransport) Username() string {
	return c.username
}

type websocketTextTransport struct {
	*websocketTransport
}

// Write writes data to the connection. It is possible to allow writer to time
// out and return a Error with Timeout() == true after a fixed time limit by
// using SetDeadline and SetWriteDeadline on the websocket.
func (c *websocketTextTransport) Write(b []byte) (n int, err error) {
	// Serialize write to avoid concurrent write
	c.Lock()
	defer c.Unlock()

	var w io.WriteCloser
	if w, err = c.socket.NextWriter(websocket.TextMessage); err == nil {
		if n, err = w.Write(b); err == nil {
			err = w.Close()
		}
	}
	return
}
