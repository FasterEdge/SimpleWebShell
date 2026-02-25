<div align="center">
  <img src="https://imagesandfont.s3.bitiful.net/icon/Logo.png?no-wait=on" alt="logo" width="96" />
  <h3>SimpleWebShell · 远程命令/文件/会话管理</h3>
</div>

### 功能简介
- 受密码保护的WebShell，支持GET/POST执行命令，可选携带session保持工作目录与上下文
- 会话持久化当前工作目录、环境快照、首选shell、命令历史、用户/组、资源探测（CPU/内存/磁盘）与主机元数据（OS/arch/hostname/uname等）
- 前端一页式：命令输入、会话选择/新建、历史会话列表（详情/删除）、文件上传/下载（进度条、可取消）
- 文件上传（multipart，支持指定目标路径）与文件下载（支持中途取消），无大小限制
- API 纯文本/JSON 返回，便于脚本调用

### 快速开始
```bash
# 启动服务（示例）
./SimpleWebShell -key 123456 -port 8878 -shell /bin/bash

# 浏览器访问
# http://<host>:8878/?key=123456
```
> 必须提供 -key。默认 shell 为 /bin/bash（Windows 推荐 cmd），默认端口 8878。

### 启动参数
| 参数      | 默认值      | 说明                           |
|-----------|-------------|--------------------------------|
| -key      | （必填）     | 访问密码                       |
| -shell    | /bin/bash   | 执行命令所用 shell 路径         |
| -port     | 8878        | 监听端口                       |

### API 一览
| 接口 | 方法 | 必要参数               | 说明 |
|------|------|--------------------|------|
| `/` | GET | key                | 返回前端页面（key 正确）或版本字符串（错误） |
| `/get` | GET | key, cmd,(session) | 执行命令；可选 session 保持目录/上下文 |
| `/post` | POST | key, cmd,(session) | 同上，支持 JSON/Form，session 可选 |
| `/get_current_path` | GET | key,(session)| 返回当前目录；session 可选，未提供则返回服务进程目录 |
| `/file_send` | POST | key                | multipart 上传，字段：file、path(可选) |
| `/file_receive` | GET | key, path          | 下载指定文件，支持取消 |
| `/session_create` | GET | key                | 新建 session，返回 session key |
| `/session_list` | GET | key                | 列出 session（id 与当前目录） |
| `/session_delete` | GET | key, session       | 删除指定 session |
| `/session_get` | GET | key, session       | 返回 session 详细 JSON |

### Session 机制
- 创建时自动探测：Owner/Groups、环境变量快照、首选 Shell、Git 分支、CPU/内存/磁盘容量、hostname/OS/arch/uname 等元数据，探测失败则跳过，不影响使用。
- 会话保存当前工作目录，`cd` 操作写入会话；命令执行成功后写入 History（限长 200 条）。
- 命令/路径类接口：传入 session 参数即在该会话目录下运行；不传则使用服务进程目录。

### 前端操作
- 顶部会话栏：默认不使用session，可勾选“使用session”，点击“新建session”生成并自动启用；列表可查看详情/删除，详情以弹窗展示完整JSON。
- 命令区：输入命令，选择GET/POST（POST可选JSON/Form），回车或点“执行”。
- 上传：选择文件，可填目标全路径（默认当前目录），显示进度，可取消。
- 下载：输入文件全路径，显示进度，可取消。

### 安全提示
- 仅供合法授权场景（远程运维/边缘计算热更新等），请妥善保管密码并限制网络可达性。
- 默认未启用HTTPS，如需公网暴露请自行加前置反向代理与访问控制。
