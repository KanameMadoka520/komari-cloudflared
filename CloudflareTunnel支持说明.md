# Cloudflare Tunnel 支持说明

本文档说明 `komari-cloudflared` 分支相较于上游 `komari-monitor/komari` 新增的 Cloudflare Tunnel 能力、实现结构、部署方式和安全边界。

## 项目定位

`komari-cloudflared` 基于 Komari 开发，保留原有监控功能，并新增了内置 `cloudflared` 管理能力。

目标是让管理员可以在后台设置页中，尽量按照 Uptime Kuma 的使用体验完成以下操作：

- 保存 Cloudflare Tunnel Token
- 启动 `cloudflared`
- 停止 `cloudflared`
- 查看安装状态、运行状态、近期日志和错误信息
- 在容器重启后自动恢复 `cloudflared`

## 后台页面位置

管理页面入口：

```text
/admin/settings/reverse-proxy
```

页面中会展示：

- `cloudflared` 是否已安装
- 当前状态：运行中 / 已停止
- Cloudflare Tunnel Token 输入框
- `Start cloudflared` / `Stop cloudflared` 按钮
- 最近状态消息
- 错误信息
- 近期日志

## 后端实现结构

主要文件如下：

- `api/admin/cloudflared.go`
  管理员接口，负责状态查询、保存 Token、启动、停止、移除 Token
- `utils/cloudflared/cloudflared.go`
  `cloudflared` 进程管理器，负责启动、停止、日志采集、运行状态维护和自动恢复
- `utils/secureconfig/secureconfig.go`
  Token 的加密与解密
- `config`
  持久化保存 Cloudflare Tunnel Token 等配置项

核心实现方式：

- 使用 Go 的 `os/exec` 启动长期运行的 `cloudflared` 进程
- 通过 `StdoutPipe` / `StderrPipe` 持续采集日志
- 在内存中维护运行状态、PID、最近日志和错误消息
- 在停止时优先尝试优雅结束，超时后再强制终止
- 启动 Komari-cloudflared 时尝试根据已保存 Token 或环境变量自动恢复 `cloudflared`

## 安全设计

### Token 存储

- Token 会保存到配置存储中
- 保存前会先加密，而不是直接明文落盘

### Token 返回策略

当前状态接口不会把明文 Token 回传给前端。

前端只能拿到：

- `tokenStored: true`
- `tokenStored: false`

这意味着：

- 浏览器不会再收到完整 Token
- 输入框不会回填真实 Token
- 页面只会通过掩码占位文本提示“已保存”

### 停止 cloudflared 的二次确认

为了避免误停：

- 如果启用了密码登录，停止时必须再次输入当前密码
- 如果已禁用密码登录，则必须在对话框中手动输入确认短语 `STOP CLOUDFLARED`

这比“禁用密码登录时直接跳过确认”更稳妥，也更接近“仍然需要重新确认管理员当前意图”的安全语义。

## 前端实现结构

主要文件如下：

- `frontend/komari-web/src/lib/cloudflared.ts`
  前端请求封装与状态类型定义
- `frontend/komari-web/src/pages/admin/settings/reverse-proxy.tsx`
  Reverse Proxy / Cloudflare Tunnel 设置页源码
- `public/defaultTheme/dist`
  构建后的前端产物

当前页面行为：

- 已保存 Token 时，输入框使用掩码占位，不回显真实值
- 输入框留空时，如后端已有已保存 Token，仍可直接点击 Start
- 停止时根据当前登录模式展示不同的二次确认方式
- 页面会定时刷新状态、日志和错误信息

## Docker 镜像设计

`komari-cloudflared` 的 Docker 镜像使用 Debian slim 作为运行时基础镜像，并在镜像构建时直接下载 `cloudflared` 可执行文件。

这样做的目的是：

- 让 Docker 部署开箱即用
- 不要求宿主机单独安装 `cloudflared`
- 让 Komari-cloudflared 能在容器内部直接管理 `cloudflared`

镜像特征：

- 运行镜像内置 `cloudflared`
- 默认监听端口为 `25774`
- 默认 SQLite 数据路径为 `/app/data/komari.db`

## Cloudflare Tunnel 源站填写注意事项

如果你使用当前仓库提供的 Docker Compose 结构，并且站点主题依赖 `/media/*` 这类由反向代理层单独托管的静态资源，那么在 Cloudflare Zero Trust Dashboard 中配置 Public Hostname 时：

- 应填写：`http://caddy:80`
- 不应填写：`http://localhost:25774`

原因是：

- `cloudflared` 运行在 `komari` 容器内部
- 容器内的 `localhost:25774` 指向的是 Komari 本体，而不是 Caddy
- 一旦绕过 Caddy，`/media/*` 资源就不会被正确提供

典型现象：

- 本机访问正常
- 公网域名访问时主题结构还在
- 但动态背景、头像、视频海报或其他 `/media/*` 资源缺失

如果你已经把 Public Hostname 源站改成了 `http://caddy:80`，但公网访问仍然表现异常，请优先检查 Cloudflare 缓存：

- 先清理首页和 `/media/*` 相关资源缓存
- 必要时直接执行 `Purge Everything`
- 否则 Cloudflare 边缘节点可能继续返回旧的错误响应，即使源站链路已经修正

## 相关环境变量

- `KOMARI_CLOUDFLARED_TOKEN`
  启动时注入 Cloudflare Tunnel Token，并尝试自动启动 `cloudflared`
- `KOMARI_CLOUDFLARED_BIN`
  指定 `cloudflared` 二进制路径，适用于非 Docker 或自定义安装场景
- `KOMARI_SECRET_KEY`
  指定配置加密主密钥

## 维护建议

如果你后续继续维护这个分支，建议遵循以下原则：

- 前端源码以 `frontend/komari-web` 为准，不要只改构建产物
- 修改前端后要重新构建，并同步 `public/defaultTheme/dist`
- 任何状态接口都不要返回明文 Token
- 对“停止 tunnel”这类高风险操作保持显式二次确认
- Docker 变更尽量保持“镜像内置 cloudflared”的体验一致性

## 与上游的关系

本项目不是对上游 Komari 的替代说明，而是一个基于上游继续开发、额外增加 Cloudflare Tunnel 支持的分支版本。

对外说明时建议明确写为：

```text
komari-cloudflared
基于 komari-monitor/komari 开发，增加内置 cloudflared 支持
```
