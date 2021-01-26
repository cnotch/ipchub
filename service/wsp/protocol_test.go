// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package wsp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/cnotch/xlog"
)

func TestDecodeRequest(t *testing.T) {
	var reqStr = "WSP/1.1 GET_INFO\r\nproto: rtsp\r\nhost: 192.168.1.1\r\nport: 554\r\nclient: \r\nseq: 1\r\n\r\n"
	t.Run("decode", func(t *testing.T) {
		r := bytes.NewBufferString(reqStr)
		got, err := DecodeRequest(r, xlog.L())
		if err != nil {
			t.Errorf("DecodeRequest() error = %v", err)
			return
		}
		assert.Equal(t, CmdGetInfo, got.Cmd)
		assert.Equal(t, "1", got.Header[FieldSeq])
	})

}

func TestRequest_ResponseOK(t *testing.T) {
	respStr1 := "WSP/1.1 200 OK\r\nchannel: 334\r\nseq: 1\r\n\r\n"
	respStr2 := "WSP/1.1 404 NOT FOUND\r\nchannel: 334\r\nseq: 1\r\n\r\n123"
	t.Run("no payload", func(t *testing.T) {
		req := &Request{
			Header: make(map[string]string),
		}
		req.Header[FieldSeq] = "1"
		buf := &bytes.Buffer{}
		header := make(map[string]string)
		header[FieldChannel] = "334"
		req.ResponseOK(buf, header, "")
		resp := buf.String()
		assert.Equal(t, respStr1, resp)
	})
	t.Run("payload", func(t *testing.T) {
		req := &Request{
			Header: make(map[string]string),
		}
		req.Header[FieldSeq] = "1"
		buf := &bytes.Buffer{}
		header := make(map[string]string)
		header[FieldChannel] = "334"
		req.ResponseTo(buf, 404, "NOT FOUND", header, "123")
		resp := buf.String()
		assert.Equal(t, respStr2, resp)
	})

}

func Benchmark_DecodeRequest(b *testing.B) {
	var reqStr = "WSP/1.1 GET_INFO\r\nproto: rtsp\r\nhost: 192.168.1.1\r\nport: 554\r\nclient: \r\nseq: 1\r\n\r\n"
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			got, _ := DecodeStringRequest(reqStr)
			_ = got
		}
	})
}
