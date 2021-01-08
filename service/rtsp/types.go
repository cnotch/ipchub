// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"github.com/cnotch/ipchub/av/format"
	"github.com/cnotch/ipchub/av/format/rtp"
	"github.com/cnotch/ipchub/av/format/rtsp"
)

// Pack .
type Pack = format.Packet

// .
var (
	ReadPacket   = rtp.ReadPacket
	ReadResponse = rtsp.ReadResponse
	ReadRequest  = rtsp.ReadRequest
)

// Request .
type Request = rtsp.Request

//Response .
type Response = rtsp.Response

// Header .
type Header = rtsp.Header

// RTPPack .
type RTPPack = rtp.Packet

const (
	rtpPackPrefix    = rtp.TransferPrefix
	rtspProto        = "RTSP/1.0" // RTSP协议版本
	rtspURLPrefix    = "rtsp://"  // RTSP地址前缀
	basicAuthPrefix  = "Basic "   // 用户基础验证前缀
	digestAuthPrefix = "Digest "  // 摘要认证前缀
)

// 预定义 RTP 包类型
const (
	ChannelVideo        = rtp.ChannelVideo
	ChannelVideoControl = rtp.ChannelVideoControl
	ChannelAudio        = rtp.ChannelAudio
	ChannelAudioControl = rtp.ChannelAudioControl
	rtpChannelCount     = rtp.ChannelCount
	rtpChannelMin       = rtp.ChannelMin
)

// 通用的 RTSP 方法。
//
// 除非特别说明，这些定义在 RFC2326 规范的 10 章中。
// 未实现的方法需要返回 "501 Not Implemented"
const (
	MethodOptions      = rtsp.MethodOptions      // 查询命令支持情况(C->S, S->C)
	MethodDescribe     = rtsp.MethodDescribe     // 获取媒体信息(C->S)
	MethodAnnounce     = rtsp.MethodAnnounce     // 声明要push的媒体信息(方向：C->S, S->C)
	MethodSetup        = rtsp.MethodSetup        // 构建传输会话，也可以调整传输参数(C->S);如果不允许调整，可以返回 455 错误
	MethodPlay         = rtsp.MethodPlay         // 开始发送媒体数据(C->S)
	MethodPause        = rtsp.MethodPause        // 暂停发送媒体数据(C->S)
	MethodTeardown     = rtsp.MethodTeardown     // 关闭发送通道；关闭后需要重新执行 Setup 方法(C->S)
	MethodGetParameter = rtsp.MethodGetParameter // 获取参数；空body可作为心跳ping(C->S, S->C)
	MethodSetParameter = rtsp.MethodSetParameter // 设置参数，应该每次只设置一个参数(C->S, S->C)
	MethodRecord       = rtsp.MethodRecord       // 启动录像(C->S)
	MethodRedirect     = rtsp.MethodRedirect     // 跳转(S->C)
)

// RTSP 头部域定义
// type:support:methods
// type: "g"通用的请求头部；"R"请求头部；"r"响应头部；"e"实体Body头部域。
// support: opt. 可选; req. 必须
// methods: 头部域应用范围
const (
	FieldAccept            = rtsp.FieldAccept            // (R:opt.:entity)
	FieldAcceptEncoding    = rtsp.FieldAcceptEncoding    // (R:opt.:entity)
	FieldAcceptLanguage    = rtsp.FieldAcceptLanguage    // (R:opt.:all)
	FieldAllow             = rtsp.FieldAllow             // (R:opt.:all)
	FieldAuthorization     = rtsp.FieldAuthorization     // (R:opt.:all)
	FieldBandwidth         = rtsp.FieldBandwidth         // (R:opt.all)
	FieldBlocksize         = rtsp.FieldBlocksize         // (R:opt.:all but OPTIONS, TEARDOWN)
	FieldCacheControl      = rtsp.FieldCacheControl      // (g:opt.:SETUP)
	FieldConference        = rtsp.FieldConference        // (R:opt.:SETUP)
	FieldConnection        = rtsp.FieldConnection        // (g:req.:all)
	FieldContentBase       = rtsp.FieldContentBase       // (e:opt.:entity)
	FieldContentEncoding   = rtsp.FieldContentEncoding   // (e:req.:SET_PARAMETER ; e:req.:DESCRIBE, ANNOUNCE )
	FieldContentLanguage   = rtsp.FieldContentLanguage   // (e:req.:DESCRIBE, ANNOUNCE)
	FieldContentLength     = rtsp.FieldContentLength     // (e:req.:SET_PARAMETER, ANNOUNCE; e:req.:entity)
	FieldContentLocation   = rtsp.FieldContentLocation   // (e:opt.:entity)
	FieldContentType       = rtsp.FieldContentType       // (e:req.:SET_PARAMETER, ANNOUNCE; r:req.:entity )
	FieldCSeq              = rtsp.FieldCSeq              // (g:req.:all)
	FieldDate              = rtsp.FieldDate              // (g:opt.:all)
	FieldExpires           = rtsp.FieldExpires           // (e:opt.:DESCRIBE, ANNOUNCE)
	FieldFrom              = rtsp.FieldFrom              // (R:opt.:all)
	FieldIfModifiedSince   = rtsp.FieldIfModifiedSince   // (R:opt.:DESCRIBE, SETUP)
	FieldLastModified      = rtsp.FieldLastModified      // (e:opt.:entity)
	FieldProxyAuthenticate = rtsp.FieldProxyAuthenticate //
	FieldProxyRequire      = rtsp.FieldProxyRequire      // (R:req.:all)
	FieldPublic            = rtsp.FieldPublic            // (r:opt.:all)
	FieldRange             = rtsp.FieldRange             // (R:opt.:PLAY, PAUSE, RECORD; r:opt.:PLAY, PAUSE, RECORD)
	FieldReferer           = rtsp.FieldReferer           // (R:opt.:all)
	FieldRequire           = rtsp.FieldRequire           // (R:req.:all)
	FieldRetryAfter        = rtsp.FieldRetryAfter        // (r:opt.:all)
	FieldRTPInfo           = rtsp.FieldRTPInfo           // (r:req.:PLAY)
	FieldScale             = rtsp.FieldScale             // (Rr:opt.:PLAY, RECORD)
	FieldSession           = rtsp.FieldSession           // (Rr:req.:all but SETUP, OPTIONS)
	FieldServer            = rtsp.FieldServer            // (r:opt.:all)
	FieldSpeed             = rtsp.FieldSpeed             // (Rr:opt.:PLAY)
	FieldTransport         = rtsp.FieldTransport         // (Rr:req.:SETUP)
	FieldUnsupported       = rtsp.FieldUnsupported       // (r:req.:all)
	FieldUserAgent         = rtsp.FieldUserAgent         // (R:opt.:all)
	FieldVia               = rtsp.FieldVia               // (g:opt.:all)
	FieldWWWAuthenticate   = rtsp.FieldWWWAuthenticate   // (r:opt.:all)
)

// RTSP 响应状态码.
// See: https://tools.ietf.org/html/rfc2326#page-19
const (
	StatusContinue = rtsp.StatusContinue

	//======Success 2xx
	StatusOK                = rtsp.StatusOK
	StatusCreated           = rtsp.StatusCreated           // only for RECORD
	StatusLowOnStorageSpace = rtsp.StatusLowOnStorageSpace //only for RECORD

	//======Redirection 3xx
	StatusMultipleChoices  = rtsp.StatusMultipleChoices
	StatusMovedPermanently = rtsp.StatusMovedPermanently
	StatusMovedTemporarily = rtsp.StatusMovedTemporarily // 和http不同
	StatusSeeOther         = rtsp.StatusSeeOther
	StatusNotModified      = rtsp.StatusNotModified
	StatusUseProxy         = rtsp.StatusUseProxy

	//======Client Error 4xx
	StatusBadRequest                = rtsp.StatusBadRequest
	StatusUnauthorized              = rtsp.StatusUnauthorized
	StatusPaymentRequired           = rtsp.StatusPaymentRequired
	StatusForbidden                 = rtsp.StatusForbidden
	StatusNotFound                  = rtsp.StatusNotFound
	StatusMethodNotAllowed          = rtsp.StatusMethodNotAllowed
	StatusNotAcceptable             = rtsp.StatusNotAcceptable
	StatusProxyAuthRequired         = rtsp.StatusProxyAuthRequired
	StatusRequestTimeout            = rtsp.StatusRequestTimeout
	StatusGone                      = rtsp.StatusGone
	StatusLengthRequired            = rtsp.StatusLengthRequired
	StatusPreconditionFailed        = rtsp.StatusPreconditionFailed // only for DESCRIBE, SETUP
	StatusRequestEntityTooLarge     = rtsp.StatusRequestEntityTooLarge
	StatusRequestURITooLong         = rtsp.StatusRequestURITooLong
	StatusUnsupportedMediaType      = rtsp.StatusUnsupportedMediaType
	StatusInvalidParameter          = rtsp.StatusInvalidParameter   // only for SETUP
	StatusConferenceNotFound        = rtsp.StatusConferenceNotFound // only for SETUP
	StatusNotEnoughBandwidth        = rtsp.StatusNotEnoughBandwidth // only for SETUP
	StatusSessionNotFound           = rtsp.StatusSessionNotFound
	StatusMethodNotValidInThisState = rtsp.StatusMethodNotValidInThisState
	StatusHeaderFieldNotValid       = rtsp.StatusHeaderFieldNotValid
	StatusInvalidRange              = rtsp.StatusInvalidRange        // only for PLAY
	StatusParameterIsReadOnly       = rtsp.StatusParameterIsReadOnly // only for SET_PARAMETER
	StatusAggregateOpNotAllowed     = rtsp.StatusAggregateOpNotAllowed
	StatusOnlyAggregateOpAllowed    = rtsp.StatusOnlyAggregateOpAllowed
	StatusUnsupportedTransport      = rtsp.StatusUnsupportedTransport
	StatusDestinationUnreachable    = rtsp.StatusDestinationUnreachable

	StatusInternalServerError     = rtsp.StatusInternalServerError
	StatusNotImplemented          = rtsp.StatusNotImplemented
	StatusBadGateway              = rtsp.StatusBadGateway
	StatusServiceUnavailable      = rtsp.StatusServiceUnavailable
	StatusGatewayTimeout          = rtsp.StatusGatewayTimeout
	StatusRTSPVersionNotSupported = rtsp.StatusRTSPVersionNotSupported
	StatusOptionNotSupported      = rtsp.StatusOptionNotSupported // 和 http 不同
)

// StatusText .
var StatusText = rtsp.StatusText
var formatDigestAuthResponse = rtsp.FormatDigestAuthResponse
