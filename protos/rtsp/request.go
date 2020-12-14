// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/url"
	"strings"

	"github.com/cnotch/ipchub/utils/scan"
)

const (
	rtspProto        = "RTSP/1.0" // RTSP协议版本
	basicAuthPrefix  = "Basic "   // 用户基础验证前缀
	digestAuthPrefix = "Digest "  // 摘要认证前缀
	rtspURLPrefix    = "rtsp://"  // RTSP地址前缀
)

// 通用的 RTSP 方法。
//
// 除非特别说明，这些定义在 RFC2326 规范的 10 章中。
// 未实现的方法需要返回 "501 Not Implemented"
const (
	MethodOptions      = "OPTIONS"       // 查询命令支持情况(C->S, S->C)
	MethodDescribe     = "DESCRIBE"      // 获取媒体信息(C->S)
	MethodAnnounce     = "ANNOUNCE"      // 声明要push的媒体信息(方向：C->S, S->C)
	MethodSetup        = "SETUP"         // 构建传输会话，也可以调整传输参数(C->S);如果不允许调整，可以返回 455 错误
	MethodPlay         = "PLAY"          // 开始发送媒体数据(C->S)
	MethodPause        = "PAUSE"         // 暂停发送媒体数据(C->S)
	MethodTeardown     = "TEARDOWN"      // 关闭发送通道；关闭后需要重新执行 Setup 方法(C->S)
	MethodGetParameter = "GET_PARAMETER" // 获取参数；空body可作为心跳ping(C->S, S->C)
	MethodSetParameter = "SET_PARAMETER" // 设置参数，应该每次只设置一个参数(C->S, S->C)
	MethodRecord       = "RECORD"        // 启动录像(C->S)
	MethodRedirect     = "REDIRECT"      // 跳转(S->C)
)

// Request 表示一个 RTSP 请求;
// 它可以是 sever 接收或发送的，也可以是 client 接收或发送的。
type Request struct {
	Method string   // RTSP 的方法（OPTIONS、DESCRIBE...）
	URL    *url.URL // 请求的 URI。
	Proto  string   // 协议版本，默认 "RTSP/1.0"
	Header Header   // 包含请求的头字段。
	Body   string   // 请求的消息体。
}

func decodeRequest(initialLine string, r *bufio.Reader) (*Request, error) {
	var err error
	var req = new(Request)

	line := initialLine
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return nil, &badStringError{"malformed RTSP request", line}
	}
	s2 += s1 + 1

	// 解析结束
	method := strings.TrimSpace(line[:s1])
	rurl := strings.TrimSpace(line[s1+1 : s2])
	proto := strings.TrimSpace(line[s2+1:])

	// 检查请求的首行内容
	// 检查方法命令名
	if len(method) == 0 || method[0] == '$' { // RTP 包的第一个字节是 `$`
		return nil, &badStringError{"invalid method", method}
	}
	// 只有 "OPTIONS" 命令的请求的URL为 *
	if method != MethodOptions && rurl == "*" {
		return nil, &badStringError{"invalid Request-URI", rurl}
	}
	if req.URL, err = url.ParseRequestURI(rurl); err != nil {
		return nil, err
	}
	req.Method = method
	req.Proto = proto
	// 去除结尾的 :
	if strings.LastIndex(req.URL.Host, ":") > strings.LastIndex(req.URL.Host, "]") {
		req.URL.Host = strings.TrimSuffix(req.URL.Host, ":")
	}

	// 读取 Header
	if req.Header, err = DecodeHeader(r); err != nil {
		return nil, err
	}

	// 读取Body
	cl := req.Header.Int(FieldContentLength)
	if cl > 0 {
		// 读取 n 字节的字串Body
		body := make([]byte, cl)
		_, err = io.ReadFull(r, body)
		req.Body = string(body)
	}
	return req, nil
}

// DecodeRequest 根据规范从 r 中解码 Request
func DecodeRequest(r *bufio.Reader) (*Request, error) {
	var err error

	// 读取并解析首行
	var line string
	line, err = readLine(r)
	if err != nil {
		return nil, err
	}

	return decodeRequest(line, r)
}

func (req *Request) String() string {
	buf := bytes.Buffer{}
	req.EncodeTo(&buf)
	return buf.String()
}

// EncodeTo 根据规范将 Request 编码到 w
func (req *Request) EncodeTo(w io.Writer) error {
	ws, ok := w.(writeStringer)
	if !ok {
		ws = stringWriter{w}
	}

	// TODO：如果存在用户名密码，去掉；
	ruri := req.URL.String()

	// 写请求方法行
	ws.WriteString(req.Method)
	ws.WriteString(" ")
	ws.WriteString(ruri)
	ws.WriteString(" RTSP/1.0\r\n")

	// 写 Header
	if len(req.Body) > 0 {
		req.Header.SetInt(FieldContentLength, len(req.Body))
	} else {
		delete(req.Header, FieldContentLength)
	}
	if err := req.Header.EncodeTo(w); err != nil {
		return err
	}

	// 写 Content
	if len(req.Body) > 0 {
		if _, err := ws.WriteString(req.Body); err != nil {
			return err
		}
	}

	return nil
}

// BasicAuth returns the username and password provided in the request's
// Authorization header, if the request uses HTTP Basic Authentication.
// See RFC 2617, Section 2.
func (req *Request) BasicAuth() (username, password string, ok bool) {
	auth := req.Header.get(FieldAuthorization)
	if auth == "" {
		return
	}
	return parseBasicAuth(auth)
}

// SetBasicAuth sets the request's Authorization header to use HTTP
// Basic Authentication with the provided username and password.
//
// With HTTP Basic Authentication the provided username and password
// are not encrypted.
func (req *Request) SetBasicAuth(username, password string) {
	req.Header.set(FieldAuthorization, formatBasicAuth(username, password))
}

func formatBasicAuth(username, password string) string {
	auth := username + ":" + password
	return basicAuthPrefix + base64.StdEncoding.EncodeToString([]byte(auth))
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(basicAuthPrefix) || !strings.EqualFold(auth[:len(basicAuthPrefix)], basicAuthPrefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(basicAuthPrefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

// DigestAuth 获取摘要认证信息
func (req *Request) DigestAuth() (username, response string, ok bool) {
	auth := req.Header.get(FieldAuthorization)

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
		case "username":
			username = v
		case "response":
			response = v
		}
	}

	return username, response, len(username) > 0 && len(response) > 0
}

// SetDigestAuth 为请求设置数字认证
func (req *Request) SetDigestAuth(url *url.URL, realm, nonce, username, password string) {
	req.Header.set(FieldAuthorization, formatDigestAuth(realm, nonce, req.Method, url.String(), username, password))
}

func formatDigestAuth(realm, nonce, method, url string, username, password string) string {
	response := formatDigestAuthResponse(realm, nonce, method, url, username, password)

	buf := bytes.Buffer{}

	buf.WriteString(`Digest username="`)
	buf.WriteString(username)
	buf.WriteString(`", realm="`)
	buf.WriteString(realm)
	buf.WriteString(`", nonce="`)
	buf.WriteString(nonce)
	buf.WriteString(`", uri="`)
	buf.WriteString(url)
	buf.WriteString(`", response="`)
	buf.WriteString(response)
	buf.WriteByte('"')
	return buf.String()
}

// response= md5(md5(username:realm:password):nonce:md5(public_method:url));
func formatDigestAuthResponse(realm, nonce, method, url string, username, password string) string {
	buf := bytes.Buffer{}

	// response= md5(md5(username:realm:password):nonce:md5(public_method:url));
	buf.WriteString(username)
	buf.WriteByte(':')
	buf.WriteString(realm)
	buf.WriteByte(':')
	buf.WriteString(password)

	md5Digest := md5.Sum(buf.Bytes())
	md5UserRealmPwd := hex.EncodeToString(md5Digest[:])

	buf.Reset()
	buf.WriteString(method)
	buf.WriteByte(':')
	buf.WriteString(url)
	md5Digest = md5.Sum(buf.Bytes())
	md5MethodURL := hex.EncodeToString(md5Digest[:])

	buf.Reset()
	buf.WriteString(md5UserRealmPwd)
	buf.WriteByte(':')
	buf.WriteString(nonce)
	buf.WriteByte(':')
	buf.WriteString(md5MethodURL)

	md5Digest = md5.Sum(buf.Bytes())
	return hex.EncodeToString(md5Digest[:])
}
