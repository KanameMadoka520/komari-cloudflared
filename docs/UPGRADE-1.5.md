# 1.5.0-fix1-cloudflared.1 升级说明

服务端合入官方 1.5.0-fix1 完整历史，前端源码更新至 komari-web 同名标签。旧前端与 d83b76f（2026-04-10）只有 8 个文件的定制差异，已按新版菜单、路由和类型适配。精确 SHA 见 UPSTREAM_VERSIONS.json。

保留内置 Tunnel 的加密令牌、保存/启停、停止确认、环境变量和自动恢复。Tunnel 在主配置初始化后启动，迁移向导期间也能连接；停机由统一生命周期回收。通用设置接口既不返回也不接受 Tunnel Token。

保留上游移除前已适配 metricstore 的流量定时报告，去除弃用提示。保留旧 v1 Agent 的 HTTP/WebSocket 上报、基本信息、Ping、执行结果、终端请求和命令下发；身份来自鉴权上下文，不信任上报体 UUID；数据进入与 v2 相同的新存储。文件管理要求 Agent 本身支持 v2 文件协议。

其余采用官方新版，包括 RPC Principal/权限检查、敏感操作 2FA、PWA/第三方主题修复、插件与主题市场、分块上传、终端重连、指标压缩与保留策略。Cloudflare Access OAuth 也保留，避免上游启动清理逻辑删除已有提供者配置。哪吒兼容按官方新版移除（现有实例未启用）。

## 本地构建与验证

运行 scripts/build-local.sh，然后 scripts/check-local.sh。需要 Go 1.27、C 编译器、Node.js 22、npm、GNU tar 和 zstd。Dockerfile 也提供完整前后端构建。修改前端后重新打包为 web/public/defaultTheme/dist.tar.zst，由服务端内嵌；不再使用旧 dist 目录。

本地构建注入 Git SHA，未提交源码带 dirty 标记。Docker 构建传入 BUILD_HASH。GitHub 只用于代码存储，不恢复 workflows 或 composite actions。

## 数据保护、切换与回滚

1. 保留旧镜像及部署配置，备份整个 data 目录，包括 secret.key、主题、主数据库及 SQLite sidecar。在线数据库用 SQLite backup API，或停服后整体复制；不能只复制运行中的 komari.db。
2. 用隔离副本演练，禁用副本的真实 Tunnel 和通知。比对迁移前后的节点 UUID、Token、用户、通知、主题和加密密钥；原节点不重新注册。
3. 新版检测到旧历史时进入管理员鉴权的迁移向导。将 records、长期 records、GPU 和 Ping 数据导入目标 metricstore，不选择丢弃历史。完成后进入正常服务。
4. 核验旧历史查询、节点上报、图表、登录、Token 保存、Tunnel 重启恢复和媒体资源。远程文件等新功能受旧 Agent 自身能力限制。
5. 正式切换前重新备份最新生产数据，再使用已验证镜像迁移；新旧实例不得同时写同一份数据库。重启会短暂断开 WebSocket，节点用原 Token 自动重连。

回滚必须同时恢复升级前 data 备份与旧镜像。迁移后的数据不能仅换旧镜像可靠回退。新存储通常为 data/metrics.db，也必须纳入备份。

官方迁移器将旧历史整理为每小时 P95 点，并沿用历史跨度设置保留期。新曲线不是逐条原始采样的无损副本；升级前原始数据库必须单独保留，以保全原始采样和支持回滚。
