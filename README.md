<h1 align="center">NetRelay</h1>

<p align="center">高性能 HTTP 隧道客户端</p>

<div align="center">

![Workflow Status](https://img.shields.io/github/actions/workflow/status/bfdfhdsfdd-crypto/netrelay/build.yml?label=build&style=flat-square)
![Latest release](https://img.shields.io/github/v/release/bfdfhdsfdd-crypto/netrelay?label=latest&style=flat-square)
![License](https://img.shields.io/github/license/bfdfhdsfdd-crypto/netrelay?style=flat-square)

</div>


NetRelay 是一款高性能 HTTP 隧道客户端，面向获授权的网络测试与管理场景，重点优化传输效率和连接稳定性。

> [!WARNING]
> 本工具仅供安全研究人员、网络管理员及相关技术人员进行授权的安全测试、漏洞评估和安全审计工作使用。使用本工具进行任何未经授权的网络攻击或渗透测试等行为均属违法，使用者需自行承担相应的法律责任。

## 主要特性

- **四种连接模式**：自动探测、全双工、半双工和短连接（`classic`），按目标服务器和反向代理能力选择。
- **兼容既有服务端**：帧编码与服务端请求格式保持 wire-format 兼容，可继续使用已部署的 `suo5.jsp`、`suo5.php`、`suo5.aspx` 等旧端点；新部署可使用 `netrelay.jsp`、`netrelay.php`、`netrelay.aspx`。
- **本地代理和端口转发**：提供 SOCKS5 代理，也可将本地 TCP 连接转发到指定目标。
- **可配置的 HTTP 通信**：支持上游 HTTP/SOCKS5 代理、自定义请求头、Cookie Jar、心跳和 Gzip 开关、请求重试及负载均衡重定向。
- **传输稳定性与资源控制**：处理短读/短写、缓冲区复用和并发背压；分块编码并限制帧解码输入，对传输帧设置 32 MiB 大小上限。编码和序列化优化不改变与既有服务端通信的 wire format。
- **持续集成和发布**：自动运行 Go 测试、GUI 前端构建及多平台 CLI 构建；版本标签会触发 Linux amd64/arm64 容器镜像发布。

## 快速开始

### 1. 下载客户端

前往 [Releases](https://github.com/bfdfhdsfdd-crypto/netrelay/releases) 下载对应平台的二进制文件：

- `netrelay-gui-*` - 图形界面版本
- `netrelay-*` - 命令行版本

CLI 当前提供 Windows amd64、macOS amd64/arm64 和 Linux amd64/arm64 构建。GUI 安装包按发布页提供的操作系统下载。

### 容器镜像

每个 `v*` 版本会自动发布 Linux amd64/arm64 多架构镜像到 GitHub Packages：

```bash
echo "$GHCR_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
docker pull ghcr.io/bfdfhdsfdd-crypto/netrelay:v0.1.0
docker run --rm -p 1111:1111 ghcr.io/bfdfhdsfdd-crypto/netrelay:v0.1.0 \
  --target https://your-authorized-server.example/netrelay.jsp \
  --listen 0.0.0.0:1111
```

`GHCR_TOKEN` 需要具有 `read:packages` 权限。首次发布的镜像包默认是私有的；如需匿名拉取，可在 GitHub Packages 的包设置中将可见性改为 Public。

服务端主入口使用 `netrelay.jsp`、`netrelay.php`、`netrelay.aspx` 等清晰名称，旧的 `suo5.*` 文件名仍可作为兼容入口；客户端也可继续连接已有部署。

### 2. 部署服务端

将 [assets](./assets/) 目录下对应的服务端文件上传到目标服务器并确认可以正常执行。

Java 目标推荐使用 [MemshellParty](https://github.com/ReaJason/MemShellParty) 项目来生成内存马并进行注入。

### 3. 连接服务端

NetRelay 针对 `auto` 模式（默认）做了大量的优化，大多数时候你无需复杂参数配置，仅使用 `-t` 指定兼容服务端
URL，即可自动确认目标支持的通信模式并在本地启动代理服务：

```bash
$ ./netrelay -t http://target.com/netrelay.jsp
...
[INFO] 14:28 connecting to target http://target.com/netrelay.jsp
[INFO] 14:28 preferred connection mode: half
[INFO] 14:28 handshake success, using session id 05q21upecl90yccl
[INFO] 14:28 netrelay is going to work on half mode
[INFO] 14:28 starting tunnel at 127.0.0.1:1111
[INFO] 14:28 creating a test connection to the remote target
[INFO] 14:28 start connection to 127.0.0.1:0
[INFO] 14:28 successfully connected to 127.0.0.1:0
[INFO] 14:28 connection closed, 127.0.0.1:0
[INFO] 14:28 congratulations! everything works fine
```

连接成功后会在本地 `127.0.0.1:1111` 开启 SOCKS5 代理服务，使用该代理会让流量从远程服务器转发出去。

```
curl -x socks5h://127.0.0.1:1111 http://internal.target.com
```

实战建议搭配 [Proxifier](https://www.proxifier.com/)，将指定应用的流量通过该代理转发。

## 客户端

GUI 版基于 [Wails](https://github.com/wailsapp/wails) 构建，提供 SOCKS5 代理和 TCP 端口转发两种工作方式、连接模式选择、运行日志、实时连接数和传输速率，以及 JSON 配置导入/导出。高级选项可配置超时、重试、请求体大小、短连接 QPS、上游代理、请求头、Cookie Jar、心跳和 Gzip。

> GUI 和 CLI 共用后端默认配置：请求方法为 POST，监听地址为 `127.0.0.1:1111`。GUI 依赖系统 WebView 组件。

CLI 版本适合命令行和服务器场景。常用参数如下（CLI 默认值）：

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-t`, `--target` | 必填 | 兼容服务端 URL |
| `-l`, `--listen` | `127.0.0.1:1111` | 本地 SOCKS5 或 TCP 转发监听地址 |
| `-m`, `--method` | `POST` | HTTP 请求方法 |
| `--mode` | `auto` | `auto`、`full`、`half` 或 `classic` |
| `--auth` | 空（无认证） | 本地 SOCKS5 用户名和密码，格式为 `username:password` |
| `-f`, `--forward` | 空 | 设置后将监听到的 TCP 连接转发到此目标，而不是提供 SOCKS5 代理 |
| `-r`, `--redirect` | 空 | 负载均衡场景下的重定向 URL |
| `-H`, `--header` | 无 | 额外 HTTP 请求头，可重复指定 |
| `--ua` | 自动生成 | 设置 HTTP User-Agent；未指定时由客户端生成 |
| `--proxy` | 无 | 上游 HTTP(S)/SOCKS5 代理，可重复指定 |
| `--timeout` | `5` 秒 | 请求超时 |
| `-S`, `--max-body-size` | `524288` 字节 | 单次请求体大小上限 |
| `--retry` | `1` | 请求重试次数 |
| `--classic-poll-qps` (`--qps`) | `6` | `classic` 模式轮询 QPS |
| `--classic-poll-interval` (`--qi`) | `200` 毫秒 | `classic` 模式轮询间隔 |
| `--dirty-size` (`--jk`) | `0` | 填充传输数据以尝试绕过代理缓冲，单位字节 |
| `-E`, `--exclude-domain` | 无 | 不经代理转发的域名规则，可重复指定 |
| `--exclude-domain-file` (`--ef`) | 无 | 每行一个排除域名规则的文件 |
| `-c`, `--config` | 无 | 从 JSON 文件读取配置 |
| `-d`, `--debug` | `false` | 输出调试信息 |
| `--no-heartbeat` (`--nh`) | `false` | 禁用每 5 秒发送的心跳 |
| `--no-gzip` (`--ng`) | `false` | 禁用 Gzip，提升旧服务端兼容性 |
| `--jar` | `false` | 启用 Cookie Jar |
| `--no-browser-headers` (`--nb`) | `false` | 不自动添加浏览器请求头 |

查看完整帮助和版本：`./netrelay --help`、`./netrelay --version`。

## 传输模式对比

NetRelay 提供四种传输模式，以适应不同的网络环境和中间件限制：

| 模式 | 通信方式 | 适用场景 |
|:---:|---|---|
| **自动**  | 探测服务端支持的通信方式并选择模式 | 默认推荐，适合先尝试 |
| **全双工** | 通过双向 Chunked-Encoding 建立持续双向通信 | 目标及中间代理支持流式请求/响应 |
| **半双工** | 下行使用长连接，上行使用短请求 | 长连接上行受限的反向代理环境 |
| **短连接** | 通过请求轮询收发数据 | 长连接不可用或不稳定的环境 |

## 使用指南

以下示例使用 CLI。GUI 可通过界面填写连接信息，也支持导入/导出同格式 JSON 配置。

### 基本用法

```bash
# 自动检测最佳传输模式
./netrelay -t https://target.com/netrelay.jsp

# 指定 HTTP 请求方法为 GET, 默认 POST
./netrelay -m GET -t https://target.com/netrelay.jsp

# 修改 Socks5 监听地址和端口, 并设置认证信息。 默认监听 127.0.0.1:1111，无认证
./netrelay -t https://target.com/netrelay.jsp -l 0.0.0.0:7788 --auth user:pass123

# 设置上游代理
# 特别注意! 仅短链接模式支持在 Burp 中抓包，其他模式会导致连接异常
./netrelay -t https://target.com/netrelay.jsp --proxy socks5://127.0.0.1:7890
./netrelay -t https://target.com/netrelay.jsp --proxy http://127.0.0.1:8080
```

### 负载均衡场景

当目标存在负载均衡时，需在所有后端部署服务端，并指定重定向 URL：

```bash
./netrelay -t https://target.com/netrelay.jsp -r http://172.0.3.2/netrelay.jsp
```

原理：判断 `-r` 中 URL 的 IP 是否与服务器网卡 IP 匹配，不匹配则转发请求。

### 域名过滤

避免特定域名走代理：

```bash
# 直接指定域名
./netrelay -t https://target.com/netrelay.jsp -E example.com -E google.com

# 使用域名列表文件（一行一个）
./netrelay -t https://target.com/netrelay.jsp -ef ./excludes.txt
```

> **注意**：确保代理请求中包含域名而非 IP。使用 `socks5h://` 协议让 DNS 解析在代理端进行：
> ```bash
> curl -x socks5h://127.0.0.1:1111 https://example.com
> ```

### 端口转发模式

直接转发 TCP 端口（非 SOCKS5 模式）：

```bash
# 将本地 1111 端口转发到目标内网的 MySQL
./netrelay -t https://target.com/netrelay.jsp -f 172.16.0.10:3306
```

### 启用 Cookie Jar

保持会话 Cookie，某些服务端可能需要：

```bash
./netrelay -t https://target.com/netrelay.jsp --jar
```

### 使用配置文件

支持通过 JSON 配置文件管理参数。字段名与 Go 配置结构一致，GUI 导入/导出使用相同格式：

```json
{
  "method": "POST",
  "listen": "127.0.0.1:1111",
  "target": "https://target.com/netrelay.jsp",
  "username": "",
  "password": "",
  "mode": "auto",
  "timeout": 5,
  "debug": false,
  "upstream_proxy": [],
  "redirect_url": "",
  "raw_header": [],
  "disable_heartbeat": false,
  "disable_gzip": false,
  "enable_cookiejar": false,
  "exclude_domain": [],
  "forward_target": "",
  "dirty_size": 0,
  "max_body_size": 524288,
  "classic_poll_qps": 6,
  "classic_poll_interval": 200,
  "retry_count": 1,
  "impersonate_browser": true
}
```

使用配置文件：

```bash
# CLI 加载配置
./netrelay -c config.json

# GUI 可通过界面导入/导出配置
```

### Go 二次开发

Go 模块路径为 `github.com/bfdfhdsfdd-crypto/netrelay`，隧道客户端 API 位于 `github.com/bfdfhdsfdd-crypto/netrelay/tunnel`。常用入口包括 `tunnel.Config`、`tunnel.DefaultConfig()` 和 `tunnel.Connect(ctx, config)`；传输连接与协议实现也在 `tunnel` 包中。

## 常见问题

**Q: 密码/认证功能**

NetRelay 的 `--auth` 参数用于设置**本地 SOCKS5 代理**的认证，格式为 `username:password`。这是为了防止本地代理被未授权使用，与服务端无关。

```bash
# 设置 SOCKS5 认证
./netrelay -t https://target.com/netrelay.jsp --auth admin:P@ssw0rd

# 使用时需要带上认证信息
curl -x socks5h://admin:P@ssw0rd@127.0.0.1:1111 http://internal.com
```

服务端脚本本身不含密码保护功能。如需保护服务端，建议使用反向代理身份验证、网络访问控制或应用层准入规则。

**Q: 如何在 Burp 中抓包调试？**

仅**短连接模式**支持在 Burp 中抓包，其他模式的长连接会导致 Burp 无法正常处理：

```bash
./netrelay -t https://target.com/netrelay.jsp --mode classic --proxy http://127.0.0.1:8080
```

**Q: 如何选择传输模式？**

推荐使用默认的 `auto` 模式，客户端会自动探测最佳模式。如果自动探测不准确，可以手动指定：

- 直连或简单反代 → `--mode full`
- Nginx 反代 → `--mode half`
- 多层反代/CDN/严格环境 → `--mode classic`

## 参考项目

- [Chunk-Proxy](https://github.com/BeichenDream/Chunk-Proxy)
- [Neo-reGeorg](https://github.com/L-codes/Neo-reGeorg)
