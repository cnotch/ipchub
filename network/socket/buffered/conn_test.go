// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package buffered

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/kelindar/rate"
)

func TestConn(t *testing.T) {
	conn := NewConn(new(fakeConn))
	defer conn.Close()

	assert.Equal(t, 0, conn.Buffered())
	assert.Nil(t, conn.LocalAddr())
	assert.Nil(t, conn.RemoteAddr())
	assert.Nil(t, conn.SetDeadline(time.Now()))
	assert.Nil(t, conn.SetReadDeadline(time.Now()))
	assert.Nil(t, conn.SetWriteDeadline(time.Now()))

	conn.limit = rate.New(1, time.Millisecond)
	for i := 0; i < 100; i++ {
		_, err := conn.Write([]byte{1, 2, 3})
		assert.NoError(t, err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err := conn.Write([]byte{1, 2, 3})
	assert.NoError(t, err)
	conn.Write(make([]byte, 122*1024))
	assert.Equal(t, defaultBufferSize, conn.writer.Cap(), "buffer can't extend")
}

// ------------------------------------------------------------------------------------

type fakeConn struct{}

func (m *fakeConn) Read(p []byte) (int, error) {
	return 0, nil
}

func (m *fakeConn) Write(p []byte) (int, error) {
	if len(p) > minBufferSize {
		return minBufferSize, nil
	}
	return len(p), nil
}

func (m *fakeConn) Close() error {
	return nil
}

func (m *fakeConn) LocalAddr() net.Addr {
	return nil
}

func (m *fakeConn) RemoteAddr() net.Addr {
	return nil
}

func (m *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}
