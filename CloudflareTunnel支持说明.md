# Cloudflare Tunnel 支持说明

本仓库 `komari-cloudflared` 基于上游项目 `komari-monitor/komari` 开发。

在保留 Komari 原有功能的基础上，额外增加了类似 Uptime Kuma 的内置 Cloudflare Tunnel 支持，目标是让管理员可以直接在后台设置页面中完成：

- 保存 Cloudflare Tunnel Token
- 启动 `cloudflared`
- 停止 `cloudflared`
- 查看安装状态、运行状态、错误信息和近期日志
- 在 Komari 重启后按已有 Token 自动恢复启动

## 设计说明

### 1. 基于 Komari 开发

- 后端仍然是 Komari 的 Go 服务
- 前端仍然基于 Komari 默认主题前端
- 默认主题构建产物已复制到 `public/defaultTheme`

### 2. 新增的 Cloudflare Tunnel 行为

- 管理员设置页新增了 Reverse Proxy / Cloudflare Tunnel 页面
- Token 会保存在 Komari 配置表中
- Token 保存前会经过加密处理
- 启动 `cloudflared` 时不通过命令行参数直接暴露 Token，而是通过环境变量传递给进程
- Docker 镜像内置 `cloudflared` 二进制，容器内即可直接启动

### 3. 环境变量

- `KOMARI_CLOUDFLARED_TOKEN`
  启动时自动注入 Token 并尝试启动 `cloudflared`

- `KOMARI_CLOUDFLARED_BIN`
  手动指定 `cloudflared` 可执行文件路径

- `KOMARI_SECRET_KEY`
  可选。用于覆盖本地自动生成的加密主密钥

## 源码结构

### 后端

- `api/admin/cloudflared.go`
  管理员接口：状态、保存 Token、启动、停止、移除 Token

- `utils/cloudflared/cloudflared.go`
  Cloudflare Tunnel 进程管理器

- `utils/secureconfig/secureconfig.go`
  Token 加密与解密

### 前端

- `public/defaultTheme/dist`
  已构建好的默认主题前端资源，包含 Reverse Proxy / Cloudflare Tunnel 页面

## Docker 镜像说明

当前 Dockerfile 已改为：

- 在 builder 阶段编译 Komari
- 在运行阶段内置 `cloudflared`
- 运行镜像使用 Debian slim，而不是 Alpine

这样做的目的，是尽量贴近 Uptime Kuma 在 Docker 场景下“镜像内直接具备 cloudflared 能力”的体验。
