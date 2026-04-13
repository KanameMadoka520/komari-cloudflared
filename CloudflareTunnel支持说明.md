# Cloudflare Tunnel 支持说明

本仓库 `komari-cloudflared` 基于上游项目 `komari-monitor/komari` 开发。

在保留 Komari 原有功能的基础上，本分支新增了类似 Uptime Kuma 的内置 Cloudflare Tunnel 管理能力，目标是让管理员可以直接在后台完成：

- 保存 Cloudflare Tunnel Token
- 启动 `cloudflared`
- 停止 `cloudflared`
- 查看安装状态、运行状态、错误信息、近期日志
- 在 Komari 重启后按已有 Token 自动恢复启动

## 实现说明

### 后端

- `api/admin/cloudflared.go`
  管理员接口：状态、保存 Token、启动、停止、移除 Token

- `utils/cloudflared/cloudflared.go`
  `cloudflared` 进程管理器

- `utils/secureconfig/secureconfig.go`
  Token 加密与解密

### 前端

- `frontend/komari-web/src/pages/admin/settings/reverse-proxy.tsx`
  Reverse Proxy / Cloudflare Tunnel 设置页源码

- `public/defaultTheme/dist`
  已构建好的默认主题前端资源

## 环境变量

- `KOMARI_CLOUDFLARED_TOKEN`
  启动时自动注入 Token 并尝试启动 `cloudflared`

- `KOMARI_CLOUDFLARED_BIN`
  手动指定 `cloudflared` 可执行文件路径

- `KOMARI_SECRET_KEY`
  可选。用于覆盖默认生成的加密主密钥

## Docker 镜像设计

当前 Dockerfile 采用两阶段构建：

1. 在 builder 阶段编译 Komari
2. 在运行阶段内置 `cloudflared`

运行镜像使用 Debian slim，而不是 Alpine。这样做是为了尽量贴近 Uptime Kuma 在 Docker 场景下“镜像内直接具备 cloudflared 能力”的体验。

## 当前行为

- 管理员可以在设置页直接保存 Token 并启动 `cloudflared`
- Token 会持久化保存
- Token 存储前会经过加密
- 容器重启后，如存在 Token，可自动恢复启动
