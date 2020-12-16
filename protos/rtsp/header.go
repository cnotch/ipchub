// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// RTSP 头部域定义
// type:support:methods
// type: "g"通用的请求头部；"R"请求头部；"r"响应头部；"e"实体Body头部域。
// support: opt. 可选; req. 必须
// methods: 头部域应用范围
const (
	FieldAccept            = "Accept"             // (R:opt.:entity)
	FieldAcceptEncoding    = "Accept-Encoding"    // (R:opt.:entity)
	FieldAcceptLanguage    = "Accept-Language"    // (R:opt.:all)
	FieldAllow             = "Allow"              // (R:opt.:all)
	FieldAuthorization     = "Authorization"      // (R:opt.:all)
	FieldBandwidth         = "Bandwidth"          // (R:opt.all)
	FieldBlocksize         = "Blocksize"          // (R:opt.:all but OPTIONS, TEARDOWN)
	FieldCacheControl      = "Cache-Control"      // (g:opt.:SETUP)
	FieldConference        = "Conference"         // (R:opt.:SETUP)
	FieldConnection        = "Connection"         // (g:req.:all)
	FieldContentBase       = "Content-Base"       // (e:opt.:entity)
	FieldContentEncoding   = "Content-Encoding"   // (e:req.:SET_PARAMETER ; e:req.:DESCRIBE, ANNOUNCE )
	FieldContentLanguage   = "Content-Language"   // (e:req.:DESCRIBE, ANNOUNCE)
	FieldContentLength     = "Content-Length"     // (e:req.:SET_PARAMETER, ANNOUNCE; e:req.:entity)
	FieldContentLocation   = "Content-Location"   // (e:opt.:entity)
	FieldContentType       = "Content-Type"       // (e:req.:SET_PARAMETER, ANNOUNCE; r:req.:entity )
	FieldCSeq              = "CSeq"               // (g:req.:all)
	FieldDate              = "Date"               // (g:opt.:all)
	FieldExpires           = "Expires"            // (e:opt.:DESCRIBE, ANNOUNCE)
	FieldFrom              = "From"               // (R:opt.:all)
	FieldIfModifiedSince   = "If-Modified-Since"  // (R:opt.:DESCRIBE, SETUP)
	FieldLastModified      = "Last-Modified"      // (e:opt.:entity)
	FieldProxyAuthenticate = "Proxy-Authenticate" //
	FieldProxyRequire      = "Proxy-Require"      // (R:req.:all)
	FieldPublic            = "Public"             // (r:opt.:all)
	FieldRange             = "Range"              // (R:opt.:PLAY, PAUSE, RECORD; r:opt.:PLAY, PAUSE, RECORD)
	FieldReferer           = "Referer"            // (R:opt.:all)
	FieldRequire           = "Require"            // (R:req.:all)
	FieldRetryAfter        = "Retry-After"        // (r:opt.:all)
	FieldRTPInfo           = "RTP-Info"           // (r:req.:PLAY)
	FieldScale             = "Scale"              // (Rr:opt.:PLAY, RECORD)
	FieldSession           = "Session"            // (Rr:req.:all but SETUP, OPTIONS)
	FieldServer            = "Server"             // (r:opt.:all)
	FieldSpeed             = "Speed"              // (Rr:opt.:PLAY)
	FieldTransport         = "Transport"          // (Rr:req.:SETUP)
	FieldUnsupported       = "Unsupported"        // (r:req.:all)
	FieldUserAgent         = "User-Agent"         // (R:opt.:all)
	FieldVia               = "Via"                // (g:opt.:all)
	FieldWWWAuthenticate   = "WWW-Authenticate"   // (r:opt.:all)
)

type badStringError struct {
	what string
	str  string
}

func (e *badStringError) Error() string { return fmt.Sprintf("%s %q", e.what, e.str) }

// Header 表示 RTSP 头部域键值对.
type Header map[string][]string

// Add 添加键值对到头部.
// 如果建已经存在，则对键值做append操作.
func (h Header) Add(key, value string) {
	key = canonicalKV(key)
	value = canonicalKV(value)
	h[key] = append(h[key], value)
}

// Set 设置头部指定键的值，操作后该键只有单值.
// 如果键已经存在，则覆盖其值.
func (h Header) Set(key, value string) {
	key = canonicalKV(key)
	value = canonicalKV(value)
	h[key] = []string{value}
}

// Get 获取指定键的值，方法对键会做规范化处理（textproto.CanonicalMIMEHeaderKey）
// 如果没有值返回“”，如果存在返回第一个值
// 想访问多值，直接使用map方法.
func (h Header) Get(key string) string {
	key = canonicalKV(key)
	v := h[key]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// set 和 Set方法类似，不对 key 进行规范化处理
func (h Header) set(key, value string) {
	h[key] = []string{value}
}

// get 和 Get方法类似，不对 key 进行规范化处理.
func (h Header) get(key string) string {
	if v := h[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// Del 删除指定 key 的值.
func (h Header) Del(key string) {
	delete(h, canonicalKV(key))
}

// SetInt 设置头部域整数值
func (h Header) SetInt(key string, value int) {
	h[key] = []string{strconv.Itoa(value)}
}

// Int 获取头部整数域值
func (h Header) Int(key string) int {
	fv := h.get(key)
	if len(fv) < 1 {
		return 0
	}

	n, err := strconv.ParseInt(fv, 10, 32)
	if err != nil || n < 0 {
		return 0
	}

	return int(n)
}

// Setf 格式化的设置头部域
func (h Header) Setf(key, format string, a ...interface{}) string {
	value := fmt.Sprintf(format, a...)
	h.set(key, value)
	return value
}

// clone 克隆头部
func (h Header) clone() Header {
	h2 := make(Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}

// ReadHeader 根据规范的格式从 r 中读取 Header
func ReadHeader(r *bufio.Reader) (Header, error) {
	h := make(Header, 6) // 多数情况够了
	for {
		var kv string
		kv, err := readLine(r)
		// 返回错误
		if err != nil {
			return nil, err
		}

		// 空行，Header读取完成退出循环;
		// 根据 Content-Length 的值来决定是否需要读取Body
		if len(kv) == 0 {
			break
		}

		i := strings.Index(kv, ":")
		// 没有找到分割符，格式错误
		if i < 0 {
			return nil, &badStringError{"malformed header line: ", kv}
		}

		key := canonicalKV(kv[:i])
		// 忽略，跳过
		if key == "" {
			continue
		}
		// 忽略key的大小写
		if canonicalKey, ok := canonicalKeys[strings.ToUpper(key)]; ok {
			key = canonicalKey
		}

		value := canonicalKV(kv[i+1:])
		h[key] = append(h[key], value)

		// // 可能存在多个值
		// values := strings.Split(kv[i+1:], ",")
		// for _, value := range values {
		// 	value = strings.TrimSpace(value)
		// 	if value == "" { // 忽略空 Value
		// 		continue
		// 	}
		// 	h[key] = append(h[key], value)
		// }
	}
	return h, nil
}

// Write 根据规范将 Header 输出到 w
func (h Header) Write(w io.Writer) error {
	ws, ok := w.(writeStringer)
	if !ok {
		ws = stringWriter{w}
	}

	kvs, sorter := h.sortedKeyValues()
	defer headerSorterPool.Put(sorter)

	for _, kv := range kvs {
		value := strings.Join(kv.values, ", ")

		for _, s := range []string{kv.key, ": ", value, "\r\n"} {
			if _, err := ws.WriteString(s); err != nil {
				return err
			}
		}
	}

	// 写 Header 结束行
	ws.WriteString("\r\n")
	return nil
}

type keyValues struct {
	key    string
	values []string
}

// A headerSorter implements sort.Interface by sorting a []keyValues
// by key. It's used as a pointer, so it can fit in a sort.Interface
// interface value without allocation.
type headerSorter struct {
	kvs []keyValues
}

func (s *headerSorter) Len() int           { return len(s.kvs) }
func (s *headerSorter) Swap(i, j int)      { s.kvs[i], s.kvs[j] = s.kvs[j], s.kvs[i] }
func (s *headerSorter) Less(i, j int) bool { return s.kvs[i].key < s.kvs[j].key }

var headerSorterPool = sync.Pool{
	New: func() interface{} { return new(headerSorter) },
}

// sortedKeyValues returns h's keys sorted in the returned kvs
// slice. The headerSorter used to sort is also returned, for possible
// return to headerSorterCache.
func (h Header) sortedKeyValues() (kvs []keyValues, hs *headerSorter) {
	hs = headerSorterPool.Get().(*headerSorter)
	if cap(hs.kvs) < len(h) {
		hs.kvs = make([]keyValues, 0, len(h))
	}
	kvs = hs.kvs[:0]
	for k, vv := range h {
		kvs = append(kvs, keyValues{k, vv})
	}
	hs.kvs = kvs
	sort.Sort(hs)
	return kvs, hs
}

// readLine 读取一行
func readLine(r *bufio.Reader) (string, error) {
	const maxLineLenght = 16 * 1024

	var line []byte
	for {
		l, more, err := r.ReadLine()
		if err != nil {
			return "", err
		}
		// Avoid the copy if the first call produced a full line.
		if line == nil && !more {
			return string(l), nil
		}
		line = append(line, l...)
		if !more {
			break
		}
		// if len(line) >maxLineLenght {
		// 	return string(line),errors.New("line over the maximum length")
		// }
	}
	return string(line), nil
}

var headerNewlineToSpace = strings.NewReplacer("\n", " ", "\r", " ")

func canonicalKV(s string) string {
	return strings.TrimSpace(headerNewlineToSpace.Replace(s))
}

var canonicalKeys = map[string]string{
	strings.ToUpper(FieldAccept):            FieldAccept,
	strings.ToUpper(FieldAcceptEncoding):    FieldAcceptEncoding,
	strings.ToUpper(FieldAcceptLanguage):    FieldAcceptLanguage,
	strings.ToUpper(FieldAllow):             FieldAllow,
	strings.ToUpper(FieldAuthorization):     FieldAuthorization,
	strings.ToUpper(FieldBandwidth):         FieldBandwidth,
	strings.ToUpper(FieldBlocksize):         FieldBlocksize,
	strings.ToUpper(FieldCacheControl):      FieldCacheControl,
	strings.ToUpper(FieldConference):        FieldConference,
	strings.ToUpper(FieldConnection):        FieldConnection,
	strings.ToUpper(FieldContentBase):       FieldContentBase,
	strings.ToUpper(FieldContentEncoding):   FieldContentEncoding,
	strings.ToUpper(FieldContentLanguage):   FieldContentLanguage,
	strings.ToUpper(FieldContentLength):     FieldContentLength,
	strings.ToUpper(FieldContentLocation):   FieldContentLocation,
	strings.ToUpper(FieldContentType):       FieldContentType,
	strings.ToUpper(FieldCSeq):              FieldCSeq,
	strings.ToUpper(FieldDate):              FieldDate,
	strings.ToUpper(FieldExpires):           FieldExpires,
	strings.ToUpper(FieldFrom):              FieldFrom,
	strings.ToUpper(FieldIfModifiedSince):   FieldIfModifiedSince,
	strings.ToUpper(FieldLastModified):      FieldLastModified,
	strings.ToUpper(FieldProxyAuthenticate): FieldProxyAuthenticate,
	strings.ToUpper(FieldProxyRequire):      FieldProxyRequire,
	strings.ToUpper(FieldPublic):            FieldPublic,
	strings.ToUpper(FieldRange):             FieldRange,
	strings.ToUpper(FieldReferer):           FieldReferer,
	strings.ToUpper(FieldRequire):           FieldRequire,
	strings.ToUpper(FieldRetryAfter):        FieldRetryAfter,
	strings.ToUpper(FieldRTPInfo):           FieldRTPInfo,
	strings.ToUpper(FieldScale):             FieldScale,
	strings.ToUpper(FieldSession):           FieldSession,
	strings.ToUpper(FieldServer):            FieldServer,
	strings.ToUpper(FieldSpeed):             FieldSpeed,
	strings.ToUpper(FieldTransport):         FieldTransport,
	strings.ToUpper(FieldUnsupported):       FieldUnsupported,
	strings.ToUpper(FieldUserAgent):         FieldUserAgent,
	strings.ToUpper(FieldVia):               FieldVia,
	strings.ToUpper(FieldWWWAuthenticate):   FieldWWWAuthenticate,
}
