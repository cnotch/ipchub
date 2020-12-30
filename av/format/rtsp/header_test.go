// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bufio"
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestReadHeader(t *testing.T) {
	tests := headerTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadHeader(bufio.NewReader(bytes.NewBufferString(tt.str)))
			if err != nil {
				t.Errorf("ReadHeader() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.header) {
				t.Errorf("ReadHeader() = %v, want %v", got, tt.header)
			}
		})
	}
}

func TestHeader_Write(t *testing.T) {
	tests := headerTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(make([]byte, 0, 1024))

			err := tt.header.Write(buf)
			got := string(buf.Bytes())
			if err != nil {
				t.Errorf("Header.Write() error = %v", err)
				return
			}
			if got != tt.str {
				t.Errorf("Header.Write() = %v, want %v", got, tt.str)
			}
		})
	}
}

type headerTestCase struct {
	name   string
	str    string
	header Header
}

func headerTestCases() []headerTestCase {
	var testCases []headerTestCase

	var str = `CSeq: 1
Proxy-Require: gzipped-messages
Require: implicit-play

`
	str = strings.Replace(str, "\n", "\r\n", -1)
	var header = make(Header)
	header.Set("CSeq", "1")
	header.Set("Require", "implicit-play")
	header.Set("Proxy-Require", "gzipped-messages")

	testCases = append(testCases, headerTestCase{"Single Value", str, header})

	str = `CSeq: 1
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

	testCases = append(testCases, headerTestCase{"Multi Value", str, header})

	return testCases
}
