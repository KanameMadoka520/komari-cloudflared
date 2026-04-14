# Komari Cloudflared

`komari-cloudflared` 基于上游项目 [`komari-monitor/komari`](https://github.com/komari-monitor/komari) 开发。

这个分支保留了 Komari 原有的监控能力，并额外集成了内置 `cloudflared` 管理功能，让管理员可以像在 Uptime Kuma 中那样，在后台的“反向代理 / Cloudflare Tunnel”页面里直接保存 Token、启动或停止 `cloudflared`，并查看运行状态、错误信息和近期日志。

当前对外版本标识为 `Komari-cloudflared分支beta`。

## 项目特性

- 基于 Komari 的原始监控功能继续维护
- 后台新增 Reverse Proxy / Cloudflare Tunnel 设置页面
- 支持保存 Cloudflare Tunnel Token，并在后端加密存储
- 支持在 Komari 进程内启动、停止和查看 `cloudflared`
- 支持通过环境变量 `KOMARI_CLOUDFLARED_TOKEN` 自动恢复启动
- Docker 镜像内置 `cloudflared` 二进制，容器部署开箱即用
- 前端源码已纳入仓库，便于后续继续维护自定义主题与设置页

## 仓库结构

- 后端源码：仓库根目录
- 前端源码：`frontend/komari-web`
- 主题静态资源：`public/defaultTheme`
- Cloudflare Tunnel 补充说明：[`CloudflareTunnel支持说明.md`](./CloudflareTunnel支持说明.md)
- 其他机器接入说明：[`其他机器接入Komari-cloudflared说明.md`](./其他机器接入Komari-cloudflared说明.md)

## 快速开始

### 构建镜像

```bash
docker build -t komari-cloudflared:beta .
```

### 运行容器

```bash
docker run -d \
  --name komari \
  -p 25774:25774 \
  -v $(pwd)/data:/app/data \
  -e ADMIN_USERNAME=admin \
  -e ADMIN_PASSWORD=your-password \
  -e KOMARI_CLOUDFLARED_TOKEN=your-cloudflare-tunnel-token \
  komari-cloudflared:beta
```

容器内默认监听端口为 `25774`，SQLite 数据默认保存在 `/app/data/komari.db`。

## 常用环境变量

- `ADMIN_USERNAME`
  初始化管理员用户名
- `ADMIN_PASSWORD`
  初始化管理员密码
- `KOMARI_CLOUDFLARED_TOKEN`
  可选。容器启动时自动注入 Cloudflare Tunnel Token，并尝试恢复启动 `cloudflared`
- `KOMARI_CLOUDFLARED_BIN`
  可选。手动指定 `cloudflared` 可执行文件路径，主要用于非 Docker 场景
- `KOMARI_SECRET_KEY`
  可选。用于覆盖默认生成的配置加密主密钥

## Cloudflare Tunnel 使用方式

1. 进入后台设置页：`/admin/settings/reverse-proxy`
2. 在 “Cloudflare Tunnel” 卡片中输入 Token
3. 点击 `Start cloudflared`
4. 返回 Cloudflare Zero Trust Dashboard，为该 tunnel 配置 Public Hostname
   对于当前仓库提供的 Docker Compose 部署，源站应填写 `http://caddy:80`

注意：

- 对于本项目当前的 Docker Compose 部署，Cloudflare Tunnel 的 Public Hostname 源站必须填写 `http://caddy:80`
- 不要在 Cloudflare Tunnel 配置里填写容器内的 `http://localhost:25774`
- 如果你的页面主题、动态背景、头像或其他资源依赖 `/media/*`，绕过 Caddy 会直接导致这些资源无法加载
- 典型现象是：本机访问正常，但通过公网域名访问时看不到动态背景、头像或其他媒体资源

安全说明：

- Token 会在后端加密保存
- 状态接口不会将明文 Token 回传给前端
- 前端只会收到 `tokenStored=true/false`，已保存状态通过掩码占位提示展示

## 前端开发

本仓库包含可编辑的前端源码：

```text
frontend/komari-web
```

如果修改了前端代码，需要重新构建并同步产物到内置主题目录。当前环境下可使用：

```bash
npm ci
node_modules/.bin/tsc -b
node_modules/.bin/vite build
```

构建完成后，将新的 `dist` 同步到 `public/defaultTheme/dist`。

## Docker 设计说明

本项目的 Dockerfile 使用 Debian slim 作为运行时基础镜像，并在镜像构建阶段直接下载 `cloudflared`，以尽量贴近 Uptime Kuma 在 Docker 场景下“镜像内置 cloudflared、后台一键启停”的使用体验。

这意味着：

- Docker 环境中不需要再单独安装 `cloudflared`
- Komari-cloudflared 会直接在容器内管理 `cloudflared` 进程
- 容器重启后，如已保存 Token 或提供了环境变量 Token，可自动恢复运行

## 致谢

- 上游项目：[`komari-monitor/komari`](https://github.com/komari-monitor/komari)
- Cloudflare Tunnel 交互参考：Uptime Kuma 的 Reverse Proxy / Cloudflare Tunnel 设计
- 站点主题参考：[`https://ss.akz.moe/`](https://ss.akz.moe/)
