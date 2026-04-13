# Komari Cloudflared

`komari-cloudflared` 基于上游项目 [`komari-monitor/komari`](https://github.com/komari-monitor/komari) 开发。

这个分支保留了 Komari 原有的轻量自托管监控能力，并新增了类似 Uptime Kuma 的内置 Cloudflare Tunnel 管理体验：

- 后台设置页可直接管理 Cloudflare Tunnel
- 支持保存 Token、启动 `cloudflared`、停止 `cloudflared`、移除 Token
- 显示安装状态、运行状态、近期日志和错误信息
- 支持通过 `KOMARI_CLOUDFLARED_TOKEN` 启动时自动恢复
- Docker 镜像内置 `cloudflared`

如果你需要原版 Komari，请使用上游仓库。

## 快速说明

### Docker 运行

```bash
docker build -t komari-cloudflared:latest .

docker run -d \
  -p 25774:25774 \
  -v $(pwd)/data:/app/data \
  -e ADMIN_USERNAME=admin \
  -e ADMIN_PASSWORD=your-password \
  -e KOMARI_CLOUDFLARED_TOKEN=your-cloudflare-tunnel-token \
  --name komari \
  komari-cloudflared:latest
```

### 关键特性

- Token 持久化保存
- Token 加密存储
- `cloudflared` 进程由 Komari 后端自行管理
- Docker 场景内置 `cloudflared`
- 非 Docker 场景支持宿主机手动安装 `cloudflared`

## 说明文档

- [CloudflareTunnel支持说明](./CloudflareTunnel支持说明.md)

## 手动构建说明

本仓库已经包含 `public/defaultTheme` 的前端构建产物，因此可以直接构建 Go 后端和 Docker 镜像。

如果你要继续修改前端设置页逻辑，前端源码已经并入当前仓库：

```text
frontend/komari-web
```

推荐在这个目录里修改并重新构建，然后把新的 `dist` 覆盖到：

```text
public/defaultTheme/dist
```

同时同步：

- `komari-theme.json`
- `preview.png`

## 鸣谢

- 上游项目：[`komari-monitor/komari`](https://github.com/komari-monitor/komari)
- 设计参考：Uptime Kuma 的 Cloudflare Tunnel / Reverse Proxy 设置体验
