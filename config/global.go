// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cnotch/ipchub/provider/auth"
	cfg "github.com/cnotch/loader"
	"github.com/cnotch/xlog"
)

// 服务名
const (
	Vendor  = "CAOHONGJU"
	Name    = "ipchub"
	Version = "V1.0.0"
)

var (
	globalC       *config
	consoleAppDir string
	demosAppDir   string
)

// InitConfig 初始化 Config
func InitConfig() {
	exe, err := os.Executable()
	if err != nil {
		xlog.Panic(err.Error())
	}

	configPath := filepath.Join(filepath.Dir(exe), Name+".conf")
	consoleAppDir = filepath.Join(filepath.Dir(exe), "console")
	demosAppDir = filepath.Join(filepath.Dir(exe), "demos")

	globalC = new(config)
	globalC.initFlags()

	// 创建或加载配置文件
	if err := cfg.Load(globalC,
		&cfg.JSONLoader{Path: configPath, CreatedIfNonExsit: true},
		&cfg.EnvLoader{Prefix: strings.ToUpper(Name)},
		&cfg.FlagLoader{}); err != nil {
		// 异常，直接退出
		xlog.Panic(err.Error())
	}

	if globalC.HlsPath != "" {
		if !filepath.IsAbs(globalC.HlsPath) {
			globalC.HlsPath = filepath.Join(filepath.Dir(exe), globalC.HlsPath)
		}

		_, err = os.Stat(globalC.HlsPath)
		if err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(globalC.HlsPath, os.ModePerm); err != nil {
					panic(err)
				}
			} else {
				panic(err)
			}
		}
	}

	// 初始化日志
	globalC.Log.initLogger()
}

// Addr Listen addr
func Addr() string {
	if globalC == nil {
		return ":554"
	}
	return globalC.ListenAddr
}

// Auth 是否启用验证
func Auth() bool {
	if globalC == nil {
		return false
	}
	return globalC.Auth
}

// CacheGop 是否Cache Gop
func CacheGop() bool {
	if globalC == nil {
		return false
	}
	return globalC.CacheGop
}

// Profile 是否启动 Http Profile
func Profile() bool {
	if globalC == nil {
		return false
	}
	return globalC.Profile
}

// GetTLSConfig 获取TLSConfig
func GetTLSConfig() *TLSConfig {
	if globalC == nil {
		return nil
	}
	return globalC.TLS
}

// ConsoleAppDir 管理员控制台应用的目录
func ConsoleAppDir() (string, bool) {
	if consoleAppDir == "" {
		return "", false
	}
	finfo, err := os.Stat(consoleAppDir)
	if err != nil || !finfo.IsDir() {
		return "", false
	}
	return consoleAppDir, true
}

// DemosAppDir 例子应用目录
func DemosAppDir() (string, bool) {
	if demosAppDir == "" {
		return "", false
	}
	finfo, err := os.Stat(demosAppDir)
	if err != nil || !finfo.IsDir() {
		return "", false
	}
	return demosAppDir, true
}

// NetTimeout 返回网络超时设置
func NetTimeout() time.Duration {
	return time.Second * 45
}

// NetHeartbeatInterval 返回网络心跳间隔
func NetHeartbeatInterval() time.Duration {
	return time.Second * 30
}

// NetBufferSize 网络通讯时的BufferSize
func NetBufferSize() int {
	return 128 * 1024
}

// NetFlushRate 网络刷新频率
func NetFlushRate() int {
	return 30
}

// RtspAuthMode rtsp 认证模式
func RtspAuthMode() auth.Mode {
	if globalC == nil || !globalC.Auth {
		return auth.NoneAuth
	}
	return auth.DigestAuth
}

// MulticastTTL 组播TTL值
func MulticastTTL() int {
	return 127
}

// HlsEnable 是否启动 Hls
func HlsEnable() bool {
	return true
}

// HlsFragment TS片段时长（s）
func HlsFragment() int {
	if globalC == nil || globalC.HlsFragment < 5 {
		return 5
	}
	return globalC.HlsFragment
}

// HlsPath hls 存储目录
func HlsPath() string {
	if globalC == nil {
		return ""
	}
	return globalC.HlsPath
}

// LoadRoutetableProvider 加载路由表提供者
func LoadRoutetableProvider(providers ...Provider) Provider {
	if globalC == nil {
		return LoadProvider(nil, providers...)
	}
	return LoadProvider(globalC.Routetable, providers...)
}

// LoadUsersProvider 加载用户提供者
func LoadUsersProvider(providers ...Provider) Provider {
	if globalC == nil {
		return LoadProvider(nil, providers...)
	}
	return LoadProvider(globalC.Users, providers...)
}

// DetectFfmpeg 判断ffmpeg命令行是否存在
func DetectFfmpeg(l *xlog.Logger) bool {
	out, err := exec.Command("ffmpeg", "-version").Output()
	if err != nil {
		return false
	}

	i := strings.Index(string(out), "Copyright")
	if i > 0 {
		l.Infof("detect %s", out[:i])
	}
	return true
}
