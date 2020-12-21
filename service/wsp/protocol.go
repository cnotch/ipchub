// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package wsp

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"

	"github.com/cnotch/ipchub/utils/scan"
	"github.com/cnotch/xlog"
)

const (
	wspProto   = "WSP/1.1"  // WSP协议版本
	prefixBody = "\r\n\r\n" // Header和Body分割符
)

// WSP 协议命令
const (
	CmdInit    = "INIT"     // 初始化建立通道
	CmdJoin    = "JOIN"     // 数据通道使用
	CmdWrap    = "WRAP"     // 包装其他协议的命令
	CmdGetInfo = "GET_INFO" // 获取客户及license信息
)

// WSP 协议字段
const (
	FieldProto   = "proto"   // 初始化的协议 如：rtsp
	FieldSeq     = "seq"     // 命令序列
	FieldHost    = "host"    // 需要代理服务访问的远端host
	FieldPort    = "port"    // 需要代理服务访问的远端port
	FieldClient  = "client"  // 客户信息
	FieldChannel = "channel" // 数据通道编号，相当于一个session
	FieldSocket  = "socket"  // 代替上面的host和port
)

type badStringError struct {
	what string
	str  string
}

func (e *badStringError) Error() string { return fmt.Sprintf("%s %q", e.what, e.str) }

// Request WSP 协议请求
type Request struct {
	Cmd    string
	Header map[string]string
	Body   string
}

var (
	spacePair = scan.NewPair(' ',
		func(r rune) bool {
			return unicode.IsSpace(r)
		})

	validCmds = map[string]bool{
		CmdGetInfo: true,
		CmdInit:    true,
		CmdJoin:    true,
		CmdWrap:    true,
	}

	bspool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 8*1024)
		},
	}
	buffers = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024*2))
		},
	}
)

// DecodeStringRequest 解码字串请求
func DecodeStringRequest(input string) (*Request, error) {
	index := strings.Index(input, prefixBody)
	if index < 0 {
		return nil, &badStringError{"malformed WSP request,missing '\\r\\n\\r\\n'", input}
	}

	req := &Request{
		Body:   input[index+4:],
		Header: make(map[string]string, 4),
	}

	scanner := scan.Line

	// 先取首行
	tailing, substr, ok := scanner.Scan(input[:index])
	if !ok {
		return nil, &badStringError{"malformed WSP request first line", substr}
	}
	proto, cmd, ok := spacePair.Scan(substr)
	if proto != wspProto {
		return nil, &badStringError{"malformed WSP request proto ", proto}
	}
	if _, ok := validCmds[cmd]; !ok {
		return nil, &badStringError{"malformed WSP request command ", cmd}
	}
	req.Cmd = cmd

	// 循环取header
	for ok {
		tailing, substr, ok = scanner.Scan(tailing)
		k, v, found := scan.ColonPair.Scan(substr)
		if found {
			req.Header[k] = v
		}
	}

	return req, nil
}

// DecodeRequest 解码请求
func DecodeRequest(r io.Reader, logger *xlog.Logger) (*Request, error) {
	buf := bspool.Get().([]byte)
	defer bspool.Put(buf)

	n, err := r.Read(buf)
	if n == 0 && err == nil { // 上一个报文结束，再读一次
		n, err = r.Read(buf)
	}

	if err != nil {
		return nil, err
	}

	input := string(buf[:n])
	logger.Debugf("wsp <<<=== \r\n%s", input)

	return DecodeStringRequest(input)
}

// IsWrap 是否是包装协议，如果是，可以从Body提取被包装的协议
func (req *Request) IsWrap() bool {
	return req.Cmd == CmdWrap
}

// ResponseOK 响应请求成功
func (req *Request) ResponseOK(buf *bytes.Buffer, header map[string]string, payload string) {
	req.ResponseTo(buf, 200, "OK", header, payload)
}

// ResponseTo 响应请求到buf
func (req *Request) ResponseTo(buf *bytes.Buffer, statusCode int, statusText string, header map[string]string, payload string) {
	// 写首行
	buf.WriteString(fmt.Sprintf("%s %d %s\r\n", wspProto, statusCode, statusText))
	// 写头
	header[FieldSeq] = req.Header[FieldSeq]
	for k, v := range header {
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(v)
		buf.WriteString("\r\n")
	}
	// 写header和body分割
	buf.WriteString("\r\n")
	// 写payload
	if len(payload) > 0 {
		buf.WriteString(payload)
	}
}
