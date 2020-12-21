// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package rtsp

import (
	"bufio"
	"strings"

	"github.com/cnotch/xlog"
)

// 接收处理器接口
type receiveHandler interface {
	onRequest(req *Request) error
	onResponse(resp *Response) error
	onPack(pack *RTPPack) error
}

// 统一消息接受函数
func receive(logger *xlog.Logger, r *bufio.Reader,
	channels []int, handler receiveHandler) error {

	// 预取4个字节
	sl, err := r.Peek(4)
	if err != nil {
		return err
	}

	// 如果是 RTP 流
	if sl[0] == rtpPackPrefix {
		pack, err := ReadPacket(r, channels)
		if err != nil {
			if pack != nil { // 通道不匹配
				logger.Warn(err.Error())
				return nil
			}
			logger.Errorf("decode rtp pack failed; %v.", err)
			return err
		}
		return handler.onPack(pack)
	}

	i := 0
	for ; i < 4; i++ {
		if sl[i] != rtspProto[i] {
			break
		}
	}

	if i == 4 { // 比较完成并且相等，是Response
		resp, err := ReadResponse(r)
		if err != nil {
			logger.Errorf("decode response failed; %v.", err)
			return err
		}

		if logger.LevelEnabled(xlog.DebugLevel) {
			logger.Debugf("<<<===\r\n%s", strings.TrimSpace(resp.String()))
		}
		return handler.onResponse(resp)
	}

	// 是请求
	req, err := ReadRequest(r)
	if err != nil {
		logger.Errorf("decode request failed; %v.", err)
		return err
	}

	if logger.LevelEnabled(xlog.DebugLevel) {
		logger.Debugf("<<<===\r\n%s", strings.TrimSpace(req.String()))
	}
	return handler.onRequest(req)
}
