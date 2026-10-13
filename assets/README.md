# NetRelay Compatible Server Scripts

## 进入查看详情

服务端脚本与既有 `suo5.jsp`、`suo5.php` 等部署文件名及传输帧格式保持兼容；改名后的客户端仍可连接已部署的旧服务端。

- [Java](java)
- [.NET](dotnet)
- [PHP](php)
- [Node.js](nodejs)

## 服务端实现优化

本次优化不改变握手、帧编码或请求/响应格式，旧版 suo5 客户端和 NetRelay 客户端仍使用相同协议：

- Java 和 ASP.NET 实现预分配序列化缓冲区；ASP.NET 解码时直接读取键和长度字段，减少临时数组及拷贝。
- PHP 实现对二进制字符串原位进行 XOR，移除逐字节数组和 `implode` 带来的额外分配。
- Node.js 实现将请求缓冲改为分块队列，避免每次收到网络数据都复制全部待处理内容；读取跨块帧时才合并所需字节。

建议先在测试环境用现有客户端验证实际中间件和代理链路，再替换已部署的服务端脚本。PHP-FPM 的并发能力仍受 `pm.max_children` 等 Worker 配置限制，优化脚本无法消除平台本身的并发上限。
