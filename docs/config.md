## 1. 配置文件(config.json)
属性 | 说明 |  示例  
-|-|-
listen | 侦听地址，":1554" | 默认：":1554" |
auth | 访问流媒体时，是否启用身份和权限验证 |默认：false |
cache_gop | 是否缓存GOP，缓存GOP会提高打开速度|默认：false |
hlsfragment | hls 分段大小（单位秒）| 默认：10 |
hlspath | hls临时文件存储目录，不设置则在内存存储|默认：空字串，使用内存文件 |
profile | 是否启动在线诊断|默认：false |
tls | 安全连接配置 |如果需要http范围，设置该配置向 |
routetable | 路由表提供者 | 默认：json provider|
users | 用户提供者 |默认：json provider|
log | 日志配置 | |

### 1.1 tls 配置
属性 | 说明 |  示例  
-|-|-
listen | 安全连接侦听地址 |默认":443" |
cert | 证书内容或文件 | |
key | 私钥内容或文件 | |

### 1.2 routetable 配置
属性 | 说明 |  示例  
-|-|-
provider | 路由表提供者名称 |默认"json" |
config | 提供者配置 | |
config.xxx | 提供者所需的配置名称和值 | |

目前支持 json 和 memory 提供者，以json为例：
``` json
	"routetable":{
		"provider":"json",
		"config":{
 			"file":"./cfg/routetable.json"
		}
	}
```
需要其他路由表提供者，需自行开发。

### 1.3 users 配置
属性 | 说明 |  示例  
-|-|-
provider | 用户安全提供者名称 |默认"json" |
config | 提供者配置 | |
config.xxx | 提供者所需的配置名称和值 | |

目前支持 json 和 memory 提供者，以json为例：
``` json
	"users":{
		"provider":"json",
		"config":{
 			"file":"./cfg/users.json"
		}
    }
```
需要其他用户安全提供者，需自行开发。

### 1.4 完整配置文件示例
``` json
{
	"listen": ":1554",
	"auth": false,
	"cache_gop": true,
	"hlspath":"./",
	"hlsfragment":10,
	"profile": false,
	"routetable":{
		"provider":"json",
		"config":{
 			"file":"./cfg/routetable.json"
		}
	},
	"users":{
		"provider":"json",
		"config":{
 			"file":"./cfg/users.json"
		}
	},
	"log": {
		"level": "debug",
		"tofile": false,
		"filename": "./logs/ipchub.log",
		"maxsize": 20,
		"maxdays": 7,
		"maxbackups": 14,
		"compress": false
	}
}
```
## 2. 路由表配置文件
默认位置在可执行文件同目录，默认名称：routetable.json。

属性 | 说明 |  示例  
-|-|-
pattern | 本地路径模式字串 | 当以'/'结尾，表示一个以pattern开头的请求都路由到下面的url |
url | 路由的目标地址，用户名和密码可以直接写在url中 | rtsp://admin:admin@localhost/live2 |
keepalive | 是否保持连接；如果没有消费者是否继续保持连接，如果为false在5分钟后自动断开 | false/true |

### 2.1 pattern
模式字串有两种形式：
+ 精确形式
+ 目录形式

目录形式以'/'字符结束，表示以此pattern开始的流路径都将路由到它对应的url。它适合于多层组织结构的路由导航。
### 2.2 完整实例：
``` json
[
	{
        "pattern": "/entrance/A1",
        "url": "rtsp://admin:admin@localhost:5540/live2",
		"keepalive": true
    },
    {
        "pattern": "/hr/",
        "url": "rtsp://admin:admin@localhost:8540/video",
		"keepalive": false
	}
]
```


访问流媒体描述
+ rtsp://localhost:1554/entrance/A1

将路由到 rtsp://admin:admin@localhost:5540/live2
+ rtsp://localhost:1554/hr/door

将路由到 rtsp://admin:admin@localhost:8540/video/door

## 3. 用户配置文件
默认位置在可执行文件同目录，默认名称：users.json。

属性 | 说明 |  示例  
-|-|-
name | 用户名 | admin |
password | 密码 |  |
admin | 是否是管理员 | false/true |
push | 推送权限 | /rooms/+/entrace |
pull | 拉取权限 | * |

### 4.1 完整示例：
``` json
[
    {
        "name":"admin",
        "password":"admin",
        "admin":true,
        "push":"*",
        "pull":"*"
    },
    {
        "name":"user1",
        "password":"user1",
        "push":"/rooms/+/entrance",
        "pull":"/test/*;/rooms/*"
    }
]
```

### 3.2 权限配置格式说明
+ `*` 0-n 段通配
+ `+` 表示可以一个路径端通配

可以通过分号设置多个
#### 3.2.1 例子1
当权限设置为 /a
+ 路径 /a 通过授权
+ 路径 /a/b 不通过授权

#### 3.2.2 例子2
当权限设置为 /a/*
+ 路径 /a 通过授权
+ 路径 /a/b, /a/c, /a/b/c 都通过授权

#### 3.2.3 例子3
当权限设置为 /a/+/c/*
+ 路径 a/b/c, a/d/c, a/b/c/d, a/b/c/d/e 都通过授权
+ 路径 a/c 不通过授权
