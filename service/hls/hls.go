// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package hls

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/xlog"
)

// GetM3u8 .
func GetM3u8(logger *xlog.Logger, path string, addr string, w http.ResponseWriter) {
	// 需要手动启动,如果需要转换或拉流，很耗时
	var c media.Hlsable
	s := media.GetOrCreate(path)
	if s != nil {
		c = s.Hlsable()
	}

	if c == nil {
		logger.Errorf("http-hls: not found stream '%s'", path)
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}

	var err error
	var cont []byte

	// 最多等待完成 30 秒
	for i := 0; i < 30; i++ {
		cont, err = c.M3u8()
		if err == nil {
			break
		}

		<-time.After(time.Second)
	}

	if err != nil {
		logger.Errorf("http-hls: request playlist error, %v.", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "application/x-mpegURL")
	w.Header().Set("Content-Length", strconv.Itoa(len(cont)))
	w.Write(cont)
}

// GetTS .
func GetTS(logger *xlog.Logger, path string, addr string, w http.ResponseWriter) {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		logger.Errorf("http-hls: path illegal `%s`", path)
		http.Error(w, "Path illegal", http.StatusBadRequest)
		return
	}

	streamPath := path[:i]
	seqStr := path[i+1:]
	seq, err := strconv.Atoi(seqStr)
	if err != nil {
		logger.Errorf("http-hls: path illegal `%s`", path)
		http.Error(w, "Path illegal", http.StatusBadRequest)
		return
	}

	// 查找的消费者但不创建
	var c media.Hlsable
	s := media.GetOrCreate(streamPath)
	if s != nil {
		c = s.Hlsable()
	}

	if c == nil {
		logger.Errorf("http-hls: not found `%s`", path)
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}

	reader, size, err := c.Segment(seq)
	if err != nil {
		logger.Errorf("http-hls: not found `%s`", path)
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			closer.Close()
		}
	}()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "video/mp2ts")
	w.Header().Set("Content-Length", strconv.Itoa(size))
	io.Copy(w, reader)
}
