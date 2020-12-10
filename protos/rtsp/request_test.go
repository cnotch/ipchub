// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bufio"
	"bytes"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestDecodeRequest(t *testing.T) {
	tests := requestTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeRequest(bufio.NewReader(bytes.NewBufferString(tt.str)))
			if err != nil {
				t.Errorf("DecodeRequest() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.req) {
				t.Errorf("DecodeRequest() = %v, want %v", got, tt.req)
			}
		})
	}
}

func TestRequest_EncodeTo(t *testing.T) {
	tests := requestTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(make([]byte, 0, 1024))
			err := tt.req.EncodeTo(buf)
			if err != nil {
				t.Errorf("Request.EncodeTo() error = %v", err)
				return
			}
			got := string(buf.Bytes())

			if got != tt.str {
				t.Errorf("Request.EncodeTo() = %v, want %v", got, tt.str)
			}
		})
	}
}

type requestTestCase struct {
	name string
	str  string
	req  *Request
}

func requestTestCases() []requestTestCase {
	var testCases []requestTestCase

	var str = `OPTIONS * RTSP/1.0
CSeq: 1
Proxy-Require: gzipped-messages
Require: implicit-play

`
	str = strings.Replace(str, "\n", "\r\n", -1)
	var header = make(Header)
	header.Set("CSeq", "1")
	header.Set("Require", "implicit-play")
	header.Set("Proxy-Require", "gzipped-messages")
	req := new(Request)
	req.Method = "OPTIONS"
	req.URL, _ = url.Parse("*")
	req.Proto = "RTSP/1.0"
	req.Header = header

	testCases = append(testCases, requestTestCase{"Single Value", str, req})

	str = `DESCRIBE rtsp://server.example.com/fizzle/foo RTSP/1.0
CSeq: 1
Public: DESCRIBE, SETUP, TEARDOWN, PLAY, PAUSE

`
	str = strings.Replace(str, "\n", "\r\n", -1)
	header = make(Header)
	header.Set("CSeq", "1")
	header.Set("Public", "DESCRIBE, SETUP, TEARDOWN, PLAY, PAUSE")
	// header.Add("Public", "DESCRIBE")
	// header.Add("Public", "SETUP")
	// header.Add("Public", "TEARDOWN")
	// header.Add("Public", "PLAY")
	// header.Add("Public", "PAUSE")

	req = new(Request)
	req.Method = "DESCRIBE"
	req.URL, _ = url.Parse("rtsp://server.example.com/fizzle/foo")
	req.Proto = "RTSP/1.0"
	req.Header = header

	testCases = append(testCases, requestTestCase{"Multi Value", str, req})

	str = `GET_PARAMETER rtsp://example.com/fizzle/foo RTSP/1.0
CSeq: 431
Content-Length: 15
Content-Type: text/parameters
Session: 12345678

123456789012345`
	str = strings.Replace(str, "\n", "\r\n", -1)
	header = make(Header)
	header.Set("CSeq", "431")
	header.Set("Content-Type", "text/parameters")
	header.Set("Session", "12345678")
	header.Set("Content-Length", "15")

	req = new(Request)
	req.Method = "GET_PARAMETER"
	req.URL, _ = url.Parse("rtsp://example.com/fizzle/foo")
	req.Proto = "RTSP/1.0"
	req.Header = header
	req.Body = "123456789012345"

	testCases = append(testCases, requestTestCase{"With Body", str, req})

	return testCases
}
