<div align="center">  
<img src="https://s2.loli.net/2025/10/30/whQl7sJryj1GbHU.png" style="width:100px;" width="100"/>  
<h2>DontCrack4OpenHarmonyLinuxKernelSide</h2>  
<h3>开源鸿蒙Linux Kernel侧专用进程管理器</h3>  
</div>  


### 一、功能简介

- 用于提高开源鸿蒙Linux Kernel的`init.cfg`中用户自定义进程健壮性、可用性、时序稳定性
- 有效避免因`init.cfg`因细微格式错误导致设备无法开机、系统进程无法顺利启动、卡开机Logo
- 支持管理这些类型的进程：`二进制可执行程序`、`sh脚本`
- 实现将进程对应到端口号，可通Restful API实现获取日志、开关进程等操作
- 启动的进程可以配置独立的：程序路径、环境变量、启动参数、预处理脚本、是否自动重启、崩溃自动重启次数、是否立即启动、端口号、日志最大缓存行数、单行日志最大字节数、日志本地存储路径、日志本地存储周期等
- 支持跨架构，免CGO，支持任何可被GO编译器编译程序的架构使用

### 二、基础用法

```  
./DontCrack \
  -path "/Volumes/老固态/Projects/FasterEdge/DontCrack4OpenHarmonyLinuxKernelSide/example/childproc/childproc" \
  -args "-mode normal -interval 500ms -lifetime 5s" \
  -env "EXTRA_INFO=from_manager RESTART_ENV_COUNT=0" \
  -file-log -log-path ./example/logs/ -log-life-day 7 \
  -auto-restart -max-retries 2 \
  -start-now \
  -password 123456
```  

| 配置项                | 类型     | 默认值                      | 说明                                                            |
| ------------------ | ------ | ------------------------ | ------------------------------------------------------------- |
| path               | string | ""                       | 要管理的程序路径（支持可执行文件、shell脚本等）                                    |
| args               | string | ""                       | 传递给程序的参数（可选）                                                  |
| pre                | string | ""                       | 启动前要执行的命令（在sh中执行，可用&&/;/\|\|连接多条命令，默认为空）                      |
| env                | string | ""                       | 为子进程追加环境变量，如: "PATH=/usr/local/bin:/usr/bin FOO=bar"；用空格或分号分隔 |
| auto-restart       | bool   | false                    | 是否自动重启                                                        |
| max-retries        | int    | 3                        | 最大重试次数（-1表示无限次，默认3次）                                          |
| start-now          | bool   | false                    | 是否立即启动                                                        |
| port               | int    | 11883                    | HTTP服务端口                                                      |
| password           | string | ""                       | 管理进程的密码（可选，默认为空且不开启密码保护）                                      |
| log-capacity       | int    | 200                      | 日志缓存的最大行数（默认200）                                              |
| log-max-line-bytes | int    | 1048576                  | 单行日志的最大字节数（用于bufio.Scanner，默认1MiB）                            |
| file-log           | bool   | false                    | 是否启用文件日志（默认false）                                             |
| log-path           | string | /data/logs/proc_manager/ | 本地日志文件目录（默认/data/logs/proc_manager/，按进程名创建子目录）                |
| log-life-day       | int    | 7                        | 本地日志文件保存天数（默认7天，新日志写入时会清理过期文件）                                |

### 三、接口文档

> /startup

- 接口说明：启动进程，同时会重置重启次数
- 请求方式：get、post
- 请求参数
  ```
  key: 密钥（可选params参数）
  ```
- 返回类型：文本
- 返回示例：
    ```
    ok
    ```

> /heartbeat

- 接口说明：获得心跳信息，会输出启动情况和缓存中的日志（同时会清除缓存）
- 请求方式：get、post
- 请求参数
  ```
  key: 密钥（可选params参数）
  ```
- 返回类型：JSON
- 返回示例：
     ```
	{
	"version": "1.1.20260112",
	"state": "stopped",
	"info": "进程管理器正常运行",
	"timestamp": "2026-02-24 15:28:04",
	"logs": [
	"[STDERR] 2026/02/24 15:27:55.647714 env restart count -> 1",
	"[STDERR] 2026/02/24 15:27:55.648316 childproc start | pid=32054 | mode=normal | interval=1s | lifetime=5s | msg=",
	"[STDERR] 2026/02/24 15:27:55.648345 args: /Volumes/老固态/Projects/FasterEdge/DontCrack4OpenHarmonyLinuxKernelSide/example/childproc/childproc -mode normal -interval 1s -lifetime 5s",
	"[STDERR] 2026/02/24 15:27:55.648352 env EXTRA_INFO=from_manager",
	"[STDERR] 2026/02/24 15:27:55.648353 env RESTART_ENV_COUNT=0",
	"[STDERR] 2026/02/24 15:27:56.668982 tick at 2026-02-24T15:27:56.64879725+08:00",
	"[STDERR] stderr burst at 2026-02-24T15:27:56.64879725+08:00",
	"[STDERR] 2026/02/24 15:27:57.686757 tick at 2026-02-24T15:27:57.648801166+08:00",
	"[STDERR] stderr burst at 2026-02-24T15:27:57.648801166+08:00",
	"[STDERR] 2026/02/24 15:27:58.652718 tick at 2026-02-24T15:27:58.648811791+08:00",
	"[STDERR] stderr burst at 2026-02-24T15:27:58.648811791+08:00",
	"[STDERR] 2026/02/24 15:27:59.668692 tick at 2026-02-24T15:27:59.648816958+08:00",
	"[STDERR] stderr burst at 2026-02-24T15:27:59.648816958+08:00",
	"[STDERR] 2026/02/24 15:28:00.686286 tick at 2026-02-24T15:28:00.648823625+08:00",
	"[STDERR] stderr burst at 2026-02-24T15:28:00.648823625+08:00",
	"[STDERR] 2026/02/24 15:28:00.686300 lifetime reached, exiting normally"
	],
	"process_pid": 0,
	"process_path": "/Volumes/老固态/Projects/FasterEdge/DontCrack4OpenHarmonyLinuxKernelSide/example/childproc/childproc",
	"restart_count": 0,
	"file_type": "binary_executable",
	"last_exit_time": "2026-02-24 15:28:00",
	"program_args": "-mode normal -interval 1s -lifetime 5s",
	"extra_env_raw": "PATH=/opt/tools/bin EXTRA_INFO=from_manager RESTART_ENV_COUNT=0"
	}
	```

> /shutdown

- 接口说明：终止进程
- 请求方式：get、post
- 请求参数
  ```
  key: 密钥（可选params参数）
  ```
- 返回类型：文本
- 返回示例：
  ```
  ok
  ```

### 四、细节说明

- 目标管理的进程的Path尽量使用全路径
- 开源鸿蒙系统一般只有一个sh命令工具位于/bin/sh，没有bash，但也可以执行脚本的
- 运行的文件使用.sh结尾、首行包含#! 都将被识别为脚本文件，由sh执行

### 五、使用技巧

- 单独使用时如果在会话中使用&作为命令结尾时一般会话结束的时候这个操作也会被杀死
- Go语言程序的log.Printf默认将数据写到os.Stderr，所以子进程中日志类型会显示为[STDERR]，可以换成fmt.Println得到非[STDERR]的消息
- 因为可以通过HTTP带加密的形式操作进程，你可以根据此文档将进程操作结合开源鸿蒙系统设备的角色，再结合合AI+MCP完成各种操作
- 除了可以使用直接功能保证进程健壮性，还可以利用反复重启机制实现进程轮询，预先脚本也能实现延迟启动，等待依赖进程等操作
