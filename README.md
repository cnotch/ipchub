# ipchub
一个小而美的流媒体服务器，即拷即用。

偶尔和前同事聊天，说到一些小的监控项目需要把IP摄像头集中管理，并提供html播放能力。闲来无事就试着开发一个打发时间，也作为学习 go 语言的一个实践。

在此之前没有流媒体经验，没有go语言项目开发经验。看了一些文档，参考了一些开源项目，主要包括：
+ [emitter](https://github.com/emitter-io/emitter) 学习多协议共享端口等网络编程技能
+ [EasyDarwin](https://github.com/EasyDarwin/EasyDarwin) 为加深对rtsp协议的理解
+ [seal](https://github.com/calabashdad/seal.git) rtmp/flv 相关协议学习及 hls 相关处理

## 做什么
摄像头集中、多级路由及h5播放

功能：
+ 流媒体源支持
    + RTSP拉流
    + RTSP推流
+ 流媒体消费支持
    + RTSP流
    + RTSP WEBSOCKET 代理
    + HTTP-FLV
    + WEBSOCKET-FLV
    + HTTP-HLS
+ 流媒体多级路由
+ 用户流媒体推拉权限管理
+ 业务系统集成API

## 不做什么
+ 不存储
+ 不转码

## 文档
+ [Quick Start](/docs/quickstart.md)
+ [Restful Api](/docs/apis.md)
+ [Server Config](/docs/config.md)
