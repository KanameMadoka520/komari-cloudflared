# Caddy 与动态背景说明

这份说明只针对当前 `komari-cloudflared` 分支。

## 结论

当前分支的默认 Docker 部署方式，**默认带有 Caddy**。

这不是因为 Cloudflare Tunnel 功能本身必须依赖 Caddy，而是因为本分支当前采用的主题资源方案，需要一个额外的同源静态文件层来提供 `/media/*` 资源。

## Caddy 的作用

当前分支里，Caddy 主要负责两件事：

1. 对外提供 `/media/*` 静态资源
2. 将其余网页请求反向代理到 Komari 主服务

在默认部署中：

- `komari` 容器负责业务服务本体
- `caddy` 容器负责对外监听端口
- `reference-media` 目录用于存放动态背景、头像等媒体资源

对应关系如下：

- 浏览器访问 `/media/desktop-snare.mp4`
- Caddy 从 `./reference-media/desktop-snare.mp4` 返回文件

## 为什么这里要用 Caddy

原因不是 `cloudflared`，而是当前主题视觉实现方式。

当前页面的动态背景脚本和挂件头像脚本，直接引用了这些路径：

- `/media/desktop-snare.mp4`
- `/media/desktop-snare-poster.webp`
- `/media/mobile-plum.mp4`
- `/media/mobile-plum-poster.webp`
- `/media/tcymc-avatar.png`

Komari 本体默认并不会把宿主机目录直接映射成 `/media/*`。

因此，为了实现下面这些效果，就需要一个轻量反向代理/静态文件服务层：

- 横屏动态视频背景
- 竖屏/窄屏自动切换的动态视频背景
- 视频封面图
- 自定义头像或挂件图
- 其他主题媒体资源

## 默认部署结构

当前分支推荐直接使用仓库根目录自带的：

- `docker-compose.yml`
- `Caddyfile`

启动方式：

```bash
docker compose up -d --build
```

默认行为：

- 构建 `komari-cloudflared:beta`
- 启动 `komari`
- 启动 `caddy`
- 对外默认暴露 `25774`

## Cloudflare Tunnel 与 Caddy 的关系

需要明确区分两件事：

### 1. Cloudflare Tunnel 功能

这部分是 `komari-cloudflared` 分支新增的后台能力：

- 保存 Token
- 启动 `cloudflared`
- 停止 `cloudflared`
- 查看状态和日志

这一部分不要求一定使用 Caddy。

### 2. 当前主题资源部署方式

这一部分才和 Caddy 有关系。

如果你的页面资源依赖 `/media/*`，那么当 Cloudflare Tunnel 对外发布网站时，Tunnel 的 Public Hostname 源站应指向：

```text
http://caddy:80
```

而不是：

```text
http://localhost:25774
```

否则会出现：

- 本机访问正常
- 公网访问时页面结构还在
- 但动态背景、头像、视频封面等资源丢失

## 这是否属于上游 Komari PR 的内容

不是。

对于上游 `komari-monitor/komari` 和 `komari-monitor/komari-web` 的 PR，我们只提交：

- Cloudflare Tunnel 的后端管理能力
- Reverse Proxy / Cloudflare Tunnel 设置页
- Docker 镜像内置 `cloudflared`

不会把当前分支自己的这些部署细节一起带去上游：

- Caddy
- `reference-media`
- 动态背景媒体资源
- 本地主题复制方案
- 本地 `/media/*` 托管方式

## 建议

对当前 `komari-cloudflared` 分支：

- 保留 Caddy 作为默认部署组成部分
- 因为它正好服务于当前横竖屏动态壁纸背景方案

对上游官方仓库 PR：

- 保持干净
- 不提及 Caddy
- 不绑定当前分支的主题资源部署结构
