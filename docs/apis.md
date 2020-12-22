## 1 系统API
系统API无需登录，可以匿名访问
### 1.1 服务器信息查询
GET /api/v1/server
#### 1.1.1 参数和响应

+ Body参数
无
+ 查询参数
无
+ 响应（200）

项目 | 类型 |  说明  
-|-|-
vendor|string| 软件提供商 |
name | string | 服务名称 |
version | string | 服务版本 |
os | string | 服务运行的平台 |
arch | string | 服务运行的平台架构 |
start_on | string(timestamp) | 服务启动时间(RFC3339Nano 格式) |
duration | string | 持续时间 |

#### 1.1.2 示例
curl 示例：
``` shell
curl http://locahost:1554/api/v1/server
```

响应：
``` json
{
	"vendor": "CAOHONGJU",
	"name": "ipchub",
	"version": "V0.8.0",
	"os": "Darwin",
	"arch": "AMD64",
	"start_on": "2019-07-15T14:02:16.638804+08:00",
	"duration": "23.319603373s"
}
```

### 1.2 运行信息查询
GET /api/v1/runtime?extra={0|1}
#### 1.2.1 参数和响应
+ Body参数
无
+ 查询参数

项目 | 类型 |  说明及示例  
-|-|-
extra | number | 0 或 1 ；如果=1响应会包含额外信息|
+ 响应（200）

获取运行是信息,extra=1返回额外信息
项目 | 类型 |  说明及示例  
-|-|-
on | timestamp | 采集时间(RFC3339Nano 格式)|
proc | object | 进程相关的统计信息 |
proc.cpu | number |cpu使用情况|
proc.priv | number |物理内存使用情况（kb）|
proc.cpu | number |虚拟内存使用情况（kb）|
proc.uptime|number | 进程运行时间（s）|
streams | object | 流信息 |
streams.sc| number | 流媒体源数量 |
streams.cc|number|流媒体消费者数量 |
rtsp| object| RTSP连接信息 |
rtsp.total|number|总链接数 |
rtsp.active | number | 活跃连接数 |
wsp| object| WSP连接信息 |
wsp.total|number|总链接数 |
wsp.active | number | 活跃连接数 |
flv| object| flv连接信息 |
flv.total|number|总链接数 |
flv.active | number | 活跃连接数 |
extra | object | 额外信息 |

#### 1.2.1 示例
curl 示例一：
```
curl http://localhost:1554/api/v1/runtime
```
响应：
``` json
{
	"on": "2019-07-15T14:05:20.524916+08:00",
	"proc": {
		"cpu": 0,
		"priv": 6876,
		"virt": 2545968,
		"uptime": 183
	},
	"streams": {
		"sources": 0,
		"consumers": 0
	},
	"rtsp": {
		"total": 0,
		"active": 0
	},
	"wsp": {
		"total": 0,
		"active": 0
	},
	"flv": {
		"total": 0,
		"active": 0
	}
}
```

示例二：
```
curl http://localhost:1554/api/v1/runtime?extra=1
```
响应：
``` json
{
	"on": "2019-07-15T14:06:41.012543+08:00",
	"proc": {
		"cpu": 0,
		"priv": 6912,
		"virt": 2545968,
		"uptime": 264
	},
	"streams": {
		"sources": 0,
		"consumers": 0
	},
	"rtsp": {
		"total": 0,
		"active": 0
	},
	"wsp": {
		"total": 0,
		"active": 0
	},
	"rtmp": {
		"total": 0,
		"active": 0
	},
	"extra": {
		"heap": {
			"inuse": 1768,
			"sys": 64992,
			"alloc": 641,
			"idle": 63224,
			"released": 0,
			"objects": 3988
		},
		"mcache": {
			"inuse": 13,
			"sys": 16
		},
		"mspan": {
			"inuse": 28,
			"sys": 32
		},
		"stack": {
			"inuse": 544,
			"sys": 544
		},
		"gc": {
			"cpu": 0,
			"sys": 2182
		},
		"go": {
			"count": 11,
			"procs": 8,
			"sys": 70462,
			"alloc": 641
		}
	}
}
```

### 1.3 登录
POST api/v1/login

#### 1.3.1 参数和响应
+ Body 参数

项目 | 类型 |  说明及示例  
-|-|-
username | string | 用户名称|
password | string | 密码 |
+ 查询参数
无
+ 响应（200）

项目 | 类型 |  说明及示例  
-|-|-
access_token | string | 访问令牌|
refresh_token | string | 刷新令牌 |

#### 1.3.2 示例
curl 示例：
```
curl -H "Content-Type: application/json" -X POST --data '{"username":"admin","password":"admin"}' http://localhost:1554/api/v1/login
```

响应：
``` json
{
	"access_token": "e8962d3214957043680e111d14e73721",
	"refresh_token": "b447808fe9ada297bae6e2e898711bb4"
}
```

#### 1.3.3 使用access_token
所有需要授权的访问都需要 access_token。使用查询参数 token={access_token}
包括：
+ 访问Api
http://.../api/v1/streams/rtsp/room/door?token=your_access_token
+ http-flv
http://.../streams/room/door.flv?token=your_access_token
+ websocket-flv
ws://.../ws/room/door.flv?token=your_access_token
+ wps
ws://.../ws/room/door?token=your_access_token
+ rtmp
rtmp://.../room/door?token=your_access_token

### 1.4 刷新access token
GET  api/v1/refreshtoken?token={refresh_tokebn}
#### 1.3.1 参数和响应
+ Body 参数
无
+ 查询参数

项目 | 类型 |  说明及示例  
-|-|-
token | string | 登录或上次Refreshtoken返回的refresh_token|
+ 响应（200）

项目 | 类型 |  说明及示例  
-|-|-
access_token | string | 访问令牌|
refresh_token | string | 刷新令牌 |

## 2 用户管理
需要管理员权限
### 2.1 获取用户信息
GET api/v1/users/{username}

#### 1.3.1 参数和响应
+ Body 参数
无
+ 查询参数
无
+ 路径参数

项目 | 类型 |  说明及示例  
-|-|-
username | string | 用户名称|
+ 响应（200）

项目 | 类型 |  说明及示例  
-|-|-
name | string | 用户名 |
admin | string | 是否是管理员 |
push | string |推送权限 |
pull | string | 拉取权限 |

### 2.2 删除用户
DELETE api/v1/users/{username}

删除用户信息，但不会断开已有连接

### 2.3 创建或更新用户信息
POST api/v1/users?update_password={0|1}

update_password 如果用户已存在，1 更新密码，其他值不会更新密码

### 2.4 获取用户列表
GET api/v1/users

#### 1.3.1 参数和响应
+ Body 参数
无
+ 查询参数

项目 | 类型 |  说明及示例  
-|-|-
page_size | number | 分页大小 |
page_token | string | 上次查询时返回的页token |
+ 路径参数
无
+ 响应（200）

项目 | 类型 |  说明及示例 |
-|-|-|
total | number | 用户总数 |
next_page_token | string | 下次查询的token |
users | array | 用户列表 |
 name | string | 用户名 |
 admin | string | 是否是管理员 |
 push | string |推送权限 |
 pull | string | 拉取权限 |


## 3 路由管理
### 3.1 基本对象
#### 3.1.1 路由
属性 | 类型 |  说明及示例   
-|-|-
pattern | string | 本地路径模式字串|
url | string | 路由的目标地址，用户名和密码可以直接写在url中 |
keepalive | bool|是否保持连接；如果没有消费者是否继续保持连接，如果为false在5分钟后自动断开 |

#### 3.1.2 路由表
属性 | 类型 |  说明及示例   
-|-|-
total |number |路由表中总个数
next_page_token | string |下一页查询需带上的 page_token
routes | array|路由信息数组

### 3.2 获取路由信息
GET api/v1/routes/{pattern=**}

### 3.2 删除路由
DELETE api/v1/routes/{pattern=**}
但不会断开已有连接

### 3.3 创建路由
POST api/v1/routes
创建或更新路由信息

### 3.4 获取路由表
GET api/v1/routes

+ 查询参数

项目 | 类型 |  说明及示例  
-|-|-
page_size | number | 分页大小 |
page_token | string | 上次查询时返回的页token |

### 4 流管理
### 4.1 基本对象
#### 4.1.1 流
属性 | 类型 |  说明及示例 
-|-|-
start_on | string(timestamp) | 流启动时间(RFC3339Nano 格式) |
path | string | 流路径|
addr | string | 流提供者的地址，push或pull|
size | number | 流的大小|
video| object | 视频元数据|
audio| object | 音频元数据|
cc | number | 正在消费流的消费者数量|
cs | array | 正在消费流的消费者数组|
[].id | number | 消费者ID|
[].start_on | string(timestamp) | 消费启动时间(RFC3339Nano 格式) |
[].packet_type | string | 消费的包类型|
[].extra | string | 消费者额外描述|
[].flow | object | 消费者接收和发送的流量统计|
[].flow.inbytes | number | 消费者接收和发送的流量统计(kb)|
[].flow.outbytes | number | 消费者接收和发送的流量统计(kb)|

#### 4.1.2 流列表
属性 | 类型 |  说明及示例  
-|-|-
total | number|流总个数
next_page_token | string |下一页查询需带上的 page_token
streams | array|流数组

### 4.2 获取流列表
GET api/v1/streams?c={0|1}
获取流列表
+ 查询参数

项目 | 类型 |  说明及示例  
-|-|-
page_size | number | 分页大小 |
page_token | string | 上次查询时返回的页token |
c |number|是否返回消费者信息 1 返回，其他值不返回|

### 4.3 获取流信息
GET api/v1/streams/{path=**}?c={0|1}

### 4.4 删除流
DELETE api/v1/streams/{path=**}

### 4.5 停止指定消费者
DELETE api/v1/streams/{path=**}:consumer?cid={cid}
+ 查询参数

项目 | 类型 |  说明及示例  
-|-|-
cid | number | 消费者id |
