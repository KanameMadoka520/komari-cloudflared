# komari-cloudflared

`komari-cloudflared` 基于上游项目 [`komari-monitor/komari`](https://github.com/komari-monitor/komari) 开发。

这个分支保留了 Komari 原有的监控能力，并额外集成了内置 `cloudflared` 管理功能，让管理员可以像在 Uptime Kuma 中那样，在后台的“反向代理 / Cloudflare Tunnel”页面中直接保存 Token、启动或停止 `cloudflared`，并查看运行状态、错误信息和近期日志。

当前对外版本标识为：

```text
Komari-cloudflared分支beta
```

## 分支定位

这个分支做了两类事情：

1. 给 Komari 增加内置 Cloudflare Tunnel 管理能力
2. 提供一套默认带 Caddy 的部署方式，用于支撑当前横竖屏动态视频背景、头像挂件和其他 `/media/*` 主题资源

请注意，这两件事不是一回事：

- `cloudflared` 功能是分支新增的核心能力
- Caddy 是当前分支默认部署结构中的配套组件，用来托管主题媒体资源

## 主要特性

- 后台新增 Reverse Proxy / Cloudflare Tunnel 设置页面
- 支持保存 Cloudflare Tunnel Token，并在后端加密存储
- 支持在 Komari 进程内启动、停止和查看 `cloudflared`
- 支持通过环境变量 `KOMARI_CLOUDFLARED_TOKEN` 自动恢复启动
- Docker 运行镜像内置 `cloudflared` 二进制
- 仓库默认提供 `docker-compose.yml + Caddyfile`
- 默认部署方式包含 Caddy，用于托管横竖屏动态背景和头像等 `/media/*` 资源

## 仓库结构

- 后端源码：仓库根目录
- 前端源码：`frontend/komari-web`
- 内置主题静态资源：`public/defaultTheme`
- 默认部署编排：`docker-compose.yml`
- 默认部署反向代理：`Caddyfile`
- 默认媒体目录：`reference-media`
- Cloudflare Tunnel 说明：[`CloudflareTunnel支持说明.md`](./CloudflareTunnel支持说明.md)
- Caddy 与动态背景说明：[`Caddy与动态背景说明.md`](./Caddy与动态背景说明.md)
- 其他机器接入说明：[`其他机器接入Komari-cloudflared说明.md`](./其他机器接入Komari-cloudflared说明.md)

## 默认部署方式

当前分支推荐直接使用仓库根目录提供的 Docker Compose 部署，而不是单独 `docker run`。

原因是：

- 默认部署会同时启动 `komari` 和 `caddy`
- Caddy 负责对外监听端口
- `/media/*` 资源由 Caddy 从 `reference-media` 目录读取
- 这正是当前横竖屏动态背景、海报和头像挂件所依赖的结构

### 1. 准备环境变量

复制一份环境变量模板：

```bash
cp .env.example .env
```

然后至少修改：

- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`

如果你要默认自动连 Tunnel，也可以填写：

- `KOMARI_CLOUDFLARED_TOKEN`

### 2. 启动

```bash
docker compose up -d --build
```

默认行为：

- 构建 `komari-cloudflared:beta`
- 启动 `komari`
- 启动 `caddy`
- 对外默认监听 `25774`

### 3. 访问

默认访问地址：

```text
http://localhost:25774
```

## 为什么默认带 Caddy

当前分支的主题视觉方案直接引用了 `/media/*` 路径，例如：

- `/media/desktop-snare.mp4`
- `/media/mobile-plum.mp4`
- `/media/tcymc-avatar.png`

Komari 本体默认并不会把宿主机目录直接映射成 `/media/*`。

因此当前分支使用 Caddy 来完成：

1. 托管横屏动态背景视频
2. 托管竖屏 / 窄屏动态背景视频
3. 托管视频封面图
4. 托管头像和挂件图
5. 将其他网页请求反代到 Komari 本体

更详细说明见：

- [`Caddy与动态背景说明.md`](./Caddy与动态背景说明.md)

## Cloudflare Tunnel 使用方式

1. 进入后台设置页：`/admin/settings/reverse-proxy`
2. 在 “Cloudflare Tunnel” 卡片中输入 Token
3. 点击 `Start cloudflared`
4. 返回 Cloudflare Zero Trust Dashboard 配置 Public Hostname

### 重要提醒

如果你使用的是当前分支默认部署结构，并且页面主题依赖 `/media/*`，那么 Public Hostname 源站应填写：

```text
http://caddy:80
```

而不是：

```text
http://localhost:25774
```

否则会出现：

- 本机访问正常
- 公网页面结构存在
- 但动态背景、头像、海报等资源缺失

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
- `KOMARI_PORT`
  可选。默认 Docker Compose 对外端口，默认值为 `25774`

## 媒体资源目录

默认部署使用：

```text
./reference-media
```

这个目录用于放置：

- 动态背景视频
- 视频封面图
- 头像挂件图
- 其他主题媒体文件

仓库默认不提交实际大媒体文件，只保留目录和说明，避免把本地资源垃圾带进仓库。

说明见：

- [`reference-media/README.md`](./reference-media/README.md)

## 前端开发

本仓库包含可编辑的前端源码：

```text
frontend/komari-web
```

如果修改了前端代码，需要重新构建并同步产物到内置主题目录。

示例：

```bash
cd frontend/komari-web
npm ci
npm run build
```

然后把新的构建产物同步到：

```text
public/defaultTheme/dist
```

## Docker 设计说明

本项目的 Komari 运行镜像使用 Debian slim 作为基础镜像，并在镜像构建阶段直接下载 `cloudflared`，以尽量贴近 Uptime Kuma 在 Docker 场景下“镜像内置 cloudflared、后台一键启停”的使用体验。

而默认部署层面，则通过 Docker Compose 额外加入 Caddy：

- Komari 镜像负责业务与 `cloudflared`
- Caddy 容器负责对外监听与 `/media/*` 托管

这意味着当前分支默认部署是：

```text
komari + caddy
```

而不是单容器直接裸露 Komari。

## 与上游 PR 的关系

当前分支默认部署包含 Caddy，但这不代表上游官方仓库必须采用 Caddy。

对于上游 `komari-monitor/komari` 和 `komari-monitor/komari-web` 的 PR，我们只提交：

- Cloudflare Tunnel 的后端管理能力
- Reverse Proxy / Cloudflare Tunnel 设置页
- Docker 镜像内置 `cloudflared`

不会把下面这些当前分支特有的内容带到上游：

- Caddy
- `reference-media`
- 本地动态背景资源
- `/media/*` 静态托管方案
- 当前主题复制方案

也就是说：

- `komari-cloudflared` 分支：默认带 Caddy
- 上游官方 PR：保持干净，不提 Caddy

## 致谢

- 上游项目：[`komari-monitor/komari`](https://github.com/komari-monitor/komari)
- Cloudflare Tunnel 交互参考：Uptime Kuma 的 Reverse Proxy / Cloudflare Tunnel 设计
- 站点主题参考：[`https://ss.akz.moe/`](https://ss.akz.moe/)
