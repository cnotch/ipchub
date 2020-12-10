// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"flag"
	"os"

	"github.com/cnotch/xlog"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig 日志配置
type LogConfig struct {
	// Level 是否启动记录调试日志
	Level xlog.Level `json:"level"`

	// ToFile 是否将日志记录到文件
	ToFile bool `json:"tofile"`

	// Filename 日志文件名称
	Filename string `json:"filename"`

	// MaxSize 日志文件的最大尺寸，以兆为单位
	MaxSize int `json:"maxsize"`

	// MaxDays 旧日志最多保存多少天
	MaxDays int `json:"maxdays"`

	// MaxBackups 旧日志最多保持数量。
	// 注意：旧日志保存的条件包括 <=MaxAge && <=MaxBackups
	MaxBackups int `json:"maxbackups"`

	// Compress 是否用 gzip 压缩
	Compress bool `json:"compress"`
}

func (c *LogConfig) initFlags() {
	// 日志配置的 Flag
	flag.Var(&c.Level, "log-level",
		"Set the log level to output")
	flag.BoolVar(&c.ToFile, "log-tofile", false,
		"Determines if logs should be saved to file")
	flag.StringVar(&c.Filename, "log-filename",
		"./logs/"+Name+".log", "Set the file to write logs to")
	flag.IntVar(&c.MaxSize, "log-maxsize", 20,
		"Set the maximum size in megabytes of the log file before it gets rotated")
	flag.IntVar(&c.MaxDays, "log-maxdays", 7,
		"Set the maximum days of old log files to retain")
	flag.IntVar(&c.MaxBackups, "log-maxbackups", 14,
		"Set the maximum number of old log files to retain")
	flag.BoolVar(&c.Compress, "log-compress", false,
		"Determines if the log files should be compressed")
}

// 初始化跟日志
func (c *LogConfig) initLogger() {
	if c.ToFile {
		// 文件输出
		fileWriter := &lumberjack.Logger{
			Filename:   c.Filename,   // 日志文件路径
			MaxSize:    c.MaxSize,    // 每个日志文件保存的最大尺寸 单位：M
			MaxBackups: c.MaxBackups, // 日志文件最多保存多少个备份
			MaxAge:     c.MaxDays,    // 文件最多保存多少天
			LocalTime:  true,         // 使用本地时间
			Compress:   c.Compress,   // 日志压缩
		}

		xlog.ReplaceGlobal(
			xlog.New(xlog.NewTee(xlog.NewCore(xlog.NewConsoleEncoder(xlog.LstdFlags|xlog.Lmicroseconds|xlog.Llongfile), xlog.Lock(os.Stderr), c.Level),
				xlog.NewCore(xlog.NewJSONEncoder(xlog.Llongfile), fileWriter, c.Level)),
				xlog.AddCaller()))
	} else {
		xlog.ReplaceGlobal(
			xlog.New(xlog.NewCore(xlog.NewConsoleEncoder(xlog.LstdFlags|xlog.Lmicroseconds|xlog.Llongfile), xlog.Lock(os.Stderr), c.Level),
				xlog.AddCaller()))
	}
}
