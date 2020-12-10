// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bufio"
	"bytes"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestDecodeResponse(t *testing.T) {
	tests := responseTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeResponse(bufio.NewReader(bytes.NewBufferString(tt.str)))
			if err != nil {
				t.Errorf("DecodeResponse() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.resp) {
				t.Errorf("DecodeResponse() = %v, want %v", got, tt.resp)
			}
		})
	}
}

func TestResponse_EncodeTo(t *testing.T) {
	tests := responseTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(make([]byte, 0, 1024))
			err := tt.resp.EncodeTo(buf)
			if err != nil {
				t.Errorf("Response.EncodeTo() error = %v", err)
				return
			}
			got := string(buf.Bytes())

			if got != tt.str {
				t.Errorf("Response.EncodeTo() = %v, want %v", got, tt.str)
			}
		})
	}
}

type responseTestCase struct {
	name string
	str  string
	resp *Response
}

func responseTestCases() []responseTestCase {
	var testCases []responseTestCase

	var str = `RTSP/1.0 200 OK
CSeq: 1
Proxy-Require: gzipped-messages
Require: implicit-play

`
	str = strings.Replace(str, "\n", "\r\n", -1)
	var header = make(Header)
	header.Set("CSeq", "1")
	header.Set("Require", "implicit-play")
	header.Set("Proxy-Require", "gzipped-messages")
	resp := new(Response)
	resp.StatusCode = 200
	resp.Status = "200 OK"
	resp.Proto = "RTSP/1.0"
	resp.Header = header

	testCases = append(testCases, responseTestCase{"Single Value", str, resp})

	str = `RTSP/1.0 451 Invalid Parameter
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

	resp = new(Response)
	resp.StatusCode = 451
	resp.Status = "451 Invalid Parameter"
	resp.Proto = "RTSP/1.0"
	resp.Header = header

	testCases = append(testCases, responseTestCase{"Multi Value", str, resp})

	str = `RTSP/1.0 200 OK
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

	resp = new(Response)
	resp.StatusCode = 200
	resp.Status = "200 OK"
	resp.Proto = "RTSP/1.0"
	resp.Header = header
	resp.Body = "123456789012345"

	testCases = append(testCases, responseTestCase{"With Body", str, resp})

	return testCases
}

func Test_parseServerDigestAuthLine(t *testing.T) {

	tests := []struct {
		name      string
		auth      string
		wantRealm string
		wantNonce string
		wantOk    bool
	}{
		{
			name:      "digestparse",
			auth:      `Digest realm="Another Streaming Media", nonce="60a76a995a0cb012f1707abc188f60cb"`,
			wantRealm: "Another Streaming Media",
			wantNonce: "60a76a995a0cb012f1707abc188f60cb",
			wantOk:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{Header: make(Header)}
			resp.Header.set(FieldWWWAuthenticate, tt.auth)
			gotRealm, gotNonce, gotOk := resp.DigestAuth()
			if gotRealm != tt.wantRealm {
				t.Errorf("parseDigestAuthResp() gotRealm = %v, want %v", gotRealm, tt.wantRealm)
			}
			if gotNonce != tt.wantNonce {
				t.Errorf("parseDigestAuthResp() gotNonce = %v, want %v", gotNonce, tt.wantNonce)
			}
			if gotOk != tt.wantOk {
				t.Errorf("parseDigestAuthResp() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Benchmark_Response_DigestAuth(b *testing.B) {
	auth := `Digest realm="Another Streaming Media", nonce="60a76a995a0cb012f1707abc188f60cb"`
	resp := &Response{Header: make(Header)}
	resp.Header.set(FieldWWWAuthenticate, auth)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, ok := resp.DigestAuth()
			_ = ok
		}
	})
}

func Benchmark_Regexp_DigestAuth(b *testing.B) {
	auth := `Digest realm="Another Streaming Media", nonce="60a76a995a0cb012f1707abc188f60cb"`
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			realmRex := regexp.MustCompile(`realm="(.*?)"`)
			nonceRex := regexp.MustCompile(`nonce="(.*?)"`)
			realm := ""
			nonce := ""
			result1 := realmRex.FindStringSubmatch(auth)
			result2 := nonceRex.FindStringSubmatch(auth)

			if len(result1) == 2 {
				realm = result1[1]
			}
			if len(result2) == 2 {
				nonce = result2[1]
			}
			_ = realm
			_ = nonce
		}
	})
}
