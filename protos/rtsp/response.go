// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"strings"

	"github.com/cnotch/tomatox/utils/scan"
)

// RTSP 响应状态码.
// See: https://tools.ietf.org/html/rfc2326#page-19
const (
	StatusContinue = 100

	//======Success 2xx
	StatusOK                = 200
	StatusCreated           = 201 // only for RECORD
	StatusLowOnStorageSpace = 250 //only for RECORD

	//======Redirection 3xx
	StatusMultipleChoices  = 300
	StatusMovedPermanently = 301
	StatusMovedTemporarily = 302 // 和http不同
	StatusSeeOther         = 303
	StatusNotModified      = 304
	StatusUseProxy         = 305

	//======Client Error 4xx
	StatusBadRequest                = 400
	StatusUnauthorized              = 401
	StatusPaymentRequired           = 402
	StatusForbidden                 = 403
	StatusNotFound                  = 404
	StatusMethodNotAllowed          = 405
	StatusNotAcceptable             = 406
	StatusProxyAuthRequired         = 407
	StatusRequestTimeout            = 408
	StatusGone                      = 410
	StatusLengthRequired            = 411
	StatusPreconditionFailed        = 412 // only for DESCRIBE, SETUP
	StatusRequestEntityTooLarge     = 413
	StatusRequestURITooLong         = 414
	StatusUnsupportedMediaType      = 415
	StatusInvalidParameter          = 451 // only for SETUP
	StatusConferenceNotFound        = 452 // only for SETUP
	StatusNotEnoughBandwidth        = 453 // only for SETUP
	StatusSessionNotFound           = 454
	StatusMethodNotValidInThisState = 455
	StatusHeaderFieldNotValid       = 456
	StatusInvalidRange              = 457 // only for PLAY
	StatusParameterIsReadOnly       = 458 // only for SET_PARAMETER
	StatusAggregateOpNotAllowed     = 459
	StatusOnlyAggregateOpAllowed    = 460
	StatusUnsupportedTransport      = 461
	StatusDestinationUnreachable    = 462

	StatusInternalServerError     = 500
	StatusNotImplemented          = 501
	StatusBadGateway              = 502
	StatusServiceUnavailable      = 503
	StatusGatewayTimeout          = 504
	StatusRTSPVersionNotSupported = 505
	StatusOptionNotSupported      = 551 // 和 http 不同
)

var statusText = map[int]string{
	StatusContinue: "Continue",

	StatusOK:                "OK",
	StatusCreated:           "Created",
	StatusLowOnStorageSpace: "Low on Storage Space",

	StatusMultipleChoices:  "Multiple Choices",
	StatusMovedPermanently: "Moved Permanently",
	StatusMovedTemporarily: "Moved Temporarily",
	StatusSeeOther:         "See Other",
	StatusNotModified:      "Not Modified",
	StatusUseProxy:         "Use Proxy",

	StatusBadRequest:                "Bad Request",
	StatusUnauthorized:              "Unauthorized",
	StatusPaymentRequired:           "Payment Required",
	StatusForbidden:                 "Forbidden",
	StatusNotFound:                  "Not Found",
	StatusMethodNotAllowed:          "Method Not Allowed",
	StatusNotAcceptable:             "Not Acceptable",
	StatusProxyAuthRequired:         "Proxy Authentication Required",
	StatusRequestTimeout:            "Request Timeout",
	StatusGone:                      "Gone",
	StatusLengthRequired:            "Length Required",
	StatusPreconditionFailed:        "Precondition Failed",
	StatusRequestEntityTooLarge:     "Request Entity Too Large",
	StatusRequestURITooLong:         "Request URI Too Long",
	StatusUnsupportedMediaType:      "Unsupported Media Type",
	StatusInvalidParameter:          "Invalid parameter", // 451~462 和 http 不同
	StatusConferenceNotFound:        "Illegal Conference Identifier",
	StatusNotEnoughBandwidth:        "Not Enough Bandwidth",
	StatusSessionNotFound:           "Session Not Found",
	StatusMethodNotValidInThisState: "Method Not Valid In This State",
	StatusHeaderFieldNotValid:       "Header Field Not Valid",
	StatusInvalidRange:              "Invalid Range",
	StatusParameterIsReadOnly:       "Parameter Is Read-Only",
	StatusAggregateOpNotAllowed:     "Aggregate Operation Not Allowed",
	StatusOnlyAggregateOpAllowed:    "Only Aggregate Operation Allowed",
	StatusUnsupportedTransport:      "Unsupported Transport",
	StatusDestinationUnreachable:    "Destination Unreachable",

	StatusInternalServerError:     "Internal Server Error",
	StatusNotImplemented:          "Not Implemented",
	StatusBadGateway:              "Bad Gateway",
	StatusServiceUnavailable:      "Service Unavailable",
	StatusGatewayTimeout:          "Gateway Timeout",
	StatusRTSPVersionNotSupported: "RTSP Version Not Supported",
	StatusOptionNotSupported:      "Option not support",
}

// StatusText 返回 RTSP 状态码的文本。如果 code 未知返回空字串。
func StatusText(code int) string {
	return statusText[code]
}

// Response 表示 RTSP 请求的响应
type Response struct {
	Proto      string // e.g. "RTSP/1.0"
	StatusCode int    // e.g. 200
	Status     string // e.g. "200 OK"
	Header     Header
	Body       string
	Request    *Request
}

func decodeResponse(initialLine string, r *bufio.Reader) (*Response, error) {
	var err error
	var resp = new(Response)

	defer func() {
		if err == io.EOF { // 结尾，返回非期望的结尾
			err = io.ErrUnexpectedEOF
		}
	}()

	line := initialLine
	var i int
	if i = strings.IndexByte(line, ' '); i < 0 {
		return nil, &badStringError{"malformed RTSP response", line}
	}

	resp.Proto = line[:i]
	resp.Status = strings.TrimLeft(line[i+1:], " ")

	statusCodeStr := resp.Status
	if i := strings.IndexByte(statusCodeStr, ' '); i != -1 {
		statusCodeStr = statusCodeStr[:i]
	}
	if len(statusCodeStr) != 3 {
		return nil, &badStringError{"malformed RTSP status code", statusCodeStr}
	}
	resp.StatusCode, err = strconv.Atoi(statusCodeStr)
	if err != nil || resp.StatusCode < 0 {
		return nil, &badStringError{"malformed RTSP status code", statusCodeStr}
	}

	// 读取 Header
	if resp.Header, err = DecodeHeader(r); err != nil {
		return nil, err
	}

	// 读取Body
	cl := resp.Header.Int(FieldContentLength)
	if cl > 0 {
		// 读取 n 字节的字串Body
		body := make([]byte, cl)
		_, err = io.ReadFull(r, body)
		resp.Body = string(body)
	}
	return resp, nil
}

// DecodeResponse 根据规范从 r 中解码 Response
func DecodeResponse(r *bufio.Reader) (*Response, error) {
	var err error

	// 读取并解析首行
	var line string
	if line, err = readLine(r); err != nil {
		return nil, err
	}
	return decodeResponse(line, r)
}

func (resp *Response) String() string {
	buf := bytes.Buffer{}
	resp.EncodeTo(&buf)
	return buf.String()
}

// EncodeTo 根据规范将 Response 编码到 w
func (resp *Response) EncodeTo(w io.Writer) error {
	ws, ok := w.(writeStringer)
	if !ok {
		ws = stringWriter{w}
	}

	// Status line
	text := resp.Status
	if text == "" {
		var ok bool
		text, ok = statusText[resp.StatusCode]
		if !ok {
			text = "status code " + strconv.Itoa(resp.StatusCode)
		}
	} else {
		// Just to reduce stutter, if user set r.Status to "200 OK" and StatusCode to 200.
		// Not important.
		text = strings.TrimPrefix(text, strconv.Itoa(resp.StatusCode)+" ")
	}

	ws.WriteString("RTSP/1.0 ")
	ws.WriteString(strconv.Itoa(resp.StatusCode))
	ws.WriteString(" ")
	ws.WriteString(text)
	ws.WriteString("\r\n")

	// 写 Header
	if len(resp.Body) > 0 {
		resp.Header.SetInt(FieldContentLength, len(resp.Body))
	} else {
		delete(resp.Header, FieldContentLength)
	}
	if err := resp.Header.EncodeTo(w); err != nil {
		return err
	}

	// 写 Content
	if len(resp.Body) > 0 {
		if _, err := ws.WriteString(resp.Body); err != nil {
			return err
		}
	}

	return nil
}

// BasicAuth 获取基本认证信息
func (resp *Response) BasicAuth() (realm string, ok bool) {
	auth := resp.Header.get(FieldWWWAuthenticate)

	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(basicAuthPrefix) || !strings.EqualFold(auth[:len(basicAuthPrefix)], basicAuthPrefix) {
		return
	}

	auth = auth[len(basicAuthPrefix):]

	tailing := auth
	substr := ""
	continueScan := true
	for continueScan {
		tailing, substr, continueScan = scan.Comma.Scan(tailing)
		k, v, _ := scan.EqualPair.Scan(substr)
		switch k {
		case "realm":
			realm = v
		}
	}

	return realm, len(realm) > 0
}

// SetBasicAuth 设置摘要认证安全请求
func (resp *Response) SetBasicAuth(realm string) {
	buf := bytes.Buffer{}
	buf.WriteString(basicAuthPrefix)
	buf.WriteString("realm=\"")
	buf.WriteString(realm)
	buf.WriteByte('"')
	resp.Header.set(FieldWWWAuthenticate, buf.String())
}

// DigestAuth 获取摘要认证信息
func (resp *Response) DigestAuth() (realm, nonce string, ok bool) {
	auth := resp.Header.get(FieldWWWAuthenticate)

	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(digestAuthPrefix) || !strings.EqualFold(auth[:len(digestAuthPrefix)], digestAuthPrefix) {
		return
	}

	auth = auth[len(digestAuthPrefix):]
	tailing := auth
	substr := ""
	continueScan := true
	for continueScan {
		tailing, substr, continueScan = scan.Comma.Scan(tailing)
		k, v, _ := scan.EqualPair.Scan(substr)
		switch k {
		case "realm":
			realm = v
		case "nonce":
			nonce = v
		}
	}

	return realm, nonce, len(realm) > 0 && len(nonce) > 0
}

// SetDigestAuth 设置摘要认证安全请求
func (resp *Response) SetDigestAuth(realm, nonce string) {
	buf := bytes.Buffer{}
	buf.WriteString(digestAuthPrefix)
	buf.WriteString("realm=\"")
	buf.WriteString(realm)
	buf.WriteString("\",nonce=\"")
	buf.WriteString(nonce)
	buf.WriteByte('"')
	resp.Header.set(FieldWWWAuthenticate, buf.String())
}
