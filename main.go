// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"

	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/provider/auth"
	"github.com/cnotch/ipchub/provider/route"
	"github.com/cnotch/ipchub/service"
	"github.com/cnotch/scheduler"
	"github.com/cnotch/xlog"
)

func main() {
	// 初始化配置
	config.InitConfig()
	// 初始化全局计划任务
	scheduler.SetPanicHandler(func(job *scheduler.ManagedJob, r interface{}) {
		xlog.Errorf("scheduler task panic. tag: %v, recover: %v", job.Tag, r)
	})

	// 初始化各类提供者
	// 路由表提供者
	routetableProvider := config.LoadRoutetableProvider(route.JSON)
	route.Reset(routetableProvider.(route.Provider))

	// 用户提供者
	userProvider := config.LoadUsersProvider(auth.JSON)
	auth.Reset(userProvider.(auth.UserProvider))

	// Start new service
	svc, err := service.NewService(context.Background(), xlog.L())
	if err != nil {
		xlog.L().Panic(err.Error())
	}

	// Listen and serve
	svc.Listen()
}
