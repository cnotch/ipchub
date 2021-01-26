## 1. 安装
即拷即用，根据自己的操作系统版本拷贝相应的可执行文件。

## 2. 配置
服务器需要配置自己的摄像头拉流。
默认配置拉流的路由信息在：routetable.json中；详细参考配置文档说明。

以下是一个典型的例子：
``` json
[
	{
        "pattern": "/group/door",
        "url": "rtsp://admin:888888@192.168.110.250:8554/H264MainStream",
        "keepalive":true
    },
    {
        "pattern": "/hr/",
        "url": "rtsp://admin:admin@192.168.110.145:1554",
		"keepalive": false
	}
]
```

我们配置了两个路由：
+ /group/door : 集团大门直接连接到摄像头
+ /hr/ : 人力资源部门的摄像头路由到下级的服务器
    假设下级服务器有 /door/video1 和 /door/video2 两个摄像头，那么你可以通过 .../hr/door/video1 和 .../hr/door/video2 访问它们。

## 3. 访问流媒体
服务器提供了多种访问终端摄像头的方式，包括：
+ rtsp
+ websocket-rtsp
+ http-flv
+ websocket-flv
+ http-hls

下面我们分别使用不同的方式访问上面两个路由的摄像头。

### 3.1 使用rtsp访问
```
ffplay -rtsp_transport tcp  rtsp://localhost:1554/group/door -fflags nobuffer
ffplay -rtsp_transport udp  rtsp://localhost:1554/group/door -fflags nobuffer
ffplay -rtsp_transport udp_multicast  rtsp://localhost:1554/group/door -fflags nobuffer
```
上面分别使用了 TCP、UDP、multicast 等三种 rtsp 播放模式。

要访问hr的/door/video1，只要将/group/door换成/hr/door/video1即可。

```
ffplay -rtsp_transport tcp  rtsp://localhost:1554/hr/door/video1 -fflags nobuffer
```

rtsp://localhost:1554/hr/door/video1 请求在服务器内自动变成去拉取rtsp://admin:admin@192.168.110.145:1554/door/video1。

### 3.2 使用wsp
打开demo地址：http://localhost:1554/demos/wsp

输入：rtsp://localhost:1554/group/door 即可访问。

### 3.3 使用websocket-rtsp
打开demo地址：http://localhost:1554/demos/rtsp

输入：ws://localhost:1554/ws/group/door 即可访问。

### 3.4 使用http-flv访问
打开demo地址：http://localhost:1554/demos/flv

输入：http://locaolhost:1554/streams/group/door.flv 即可访问。

**注意：**由于 Chrome 对长连接的流限制为6个，因此如果使用 Chrome 打开更多建议使用websocket-flv。

### 3.5 使用 websocket-flv访问
打开demo地址：http://localhost:1554/demos/flv

输入：ws://locaolhost:1554/ws/group/door.flv 即可访问。

### 3.6 使用 http-hls访问
由于 iOS的Safari不支持上述任何http访问模式，请使用 http-hls

在浏览器输入: http://localhost:1554/streams/group/door.m3m8 即可访问。

**注意:** 由于http-hls的段文件默认被放在内存中，占用大量的内存；如系统内存不足，请配置存储路径。

### 3.7 访问 h265 flv
打开demo地址：http://localhost:1554/demos/flv265

输入：http://locaolhost:1554/streams/group/door.flv 即可访问。

## 4. 需要授权的情况
除rtsp外，其他使用token进行访问。
如果 http-flv,
输入：http://locaolhost:1554/streams/group/door.flv?token=7f97509e321a18ccf281607f4c0bd4fb

其中 token 通过登录api获得的相关信息请参考[配置文档](config.md) 和 [Api 文档](apis.md)。

## 5. 浏览器支持情况
http-flv、websocket-flv、websocket-rtsp等浏览器访问，支持：
+ Firefox v.42+
+ Chrome v.23+
+ OSX Safari v.8+
+ MS Edge v.13+
+ Opera v.15+
+ Android browser v.5.0+
+ IE Mobile v.11+

不支持 iOS Safari 和 IE。