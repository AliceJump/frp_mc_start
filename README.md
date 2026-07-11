# frp_mc_start

frpc 端口自动重试启动器 — 自动尝试 25565-25575 端口，遇到 `proxy already exists` 等错误时自动换端口重试。

## 功能

- 从 `config.json` 或命令行参数读取 frp 服务器配置
- 启动 frpc 并实时监控输出
- 检测到 `proxy already exists` 时自动杀掉 frpc 并尝试下一个端口
- 检测到连接超时等错误时自动重试下一个端口
- 启动成功后保持 frpc 运行

## 使用

1. 编辑 `config.json` 填写服务器信息
2. 确保 `frpc.exe` 在同一目录
3. 运行 `frp_mc_start.exe`

## 配置文件

```json
{
  "serverAddr": "你的服务器地址",
  "serverPort": 7000,
  "token": "你的token",
  "localPort": 25565,
  "portStart": 25565,
  "portEnd": 25575
}
```

命令行参数可覆盖配置文件（优先级更高）：

```
frp_mc_start.exe --server-addr 1.2.3.4 --server-port 7000 --token xxx
```

## 项目结构

```
├── main.go             # 主程序（端口重试、frpc 启动/监控）
├── frpc/               # frpc 配置生成 & 进程管理
├── port/               # 端口池
├── server/             # frp 服务端 API 查询
├── logger/             # 日志工具
├── start.bat           # 备用启动脚本
└── release/            # 打包发布目录
```
