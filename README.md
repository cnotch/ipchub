# ipchub
一个即拷即用、支持摄像头集中管理、多级路由及h5播放的流媒体服务器。

## 项目背景
偶尔和前同事聊天，说到一些小的监控项目需要把IP摄像头集中管理，并提供html播放能力。闲来无事就试着开发一个打发时间，也作为学习 go 语言的一个实践。

在此之前没有流媒体经验，没有go语言项目开发经验。看了一些文档，参考了一些开源项目，主要包括：
+ [emitter](https://github.com/emitter-io/emitter) 学习多协议共享端口等网络编程技能
+ [EasyDarwin](https://github.com/EasyDarwin/EasyDarwin) 为加深对rtsp协议的理解
+ [seal](https://github.com/calabashdad/seal.git) rtmp/flv hls 服务的理解


## 主要特性

+ 基于纯 Golang 开发
+ 支持 Windows、Linux、macOS 平台
+ 支持 RTSP 推流（主动推送）
+ 支持 RTSP 拉流（拉取摄像头或其他流媒体服务器资源）
+ 支持 RTSP TCP、UDP、Multicast 播放
+ 支持 H264+AAC H5播放，包括：
    + HTTP-FLV
    + Websocket-FLV
    + HTTP-HLS
    + Websocket-RTSP（实验）: 实时性更好
+ 支持 H265+AAC H5播放（实验，需自行寻找播放软件），包括：
    + HTTP-FLV
    + Websocket-FLV
+ 支持流媒体用户推拉权限管理
+ 业务系统集成 RestfulAPI

## 文档
+ [Quick Start](/docs/quickstart.md)
+ [Restful Api](/docs/apis.md)
+ [Server Config](/docs/config.md)
