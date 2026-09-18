# 节点流量周期

管理员在节点管理的“编辑信息 → 流量周期”中，可以立即重置上传/下载为零，或填写当前已用上传/下载量。节点的统计方式（sum/up/down/max/min）及流量阈值继续沿用原设置。

- 操作仅针对所选节点，不改 Agent Token、不重新注册节点、不删除历史。
- 流量单位按 Komari 页面原有口径以 1024 进制换算（GB、TB、GiB、TiB 均按对应的二进制倍数处理）；打开表单使用精确字节，避免将四舍五入的展示值误存为基线。
- 未执行过校准的节点维持 Agent 原始累计值。启用周期后，服务端持久化累计每次上报的增量；Agent 计数器重置和服务端重启不会清空本周期已累计用量。
- `common:getNodesLatestStatus` 的 `net_total_up/down` 保持原始含义，另提供 `traffic_used_up/down`。默认前端首页卡片、列表、概览和阈值提醒使用周期数值。第三方主题需使用新字段才能显示周期值。
- 历史指标、24 小时图表、日报/周报/月报继续记录实际流量；手动校准不制造历史流量尖峰。
- 无任何可用上报计数的节点拒绝校准，等待 Agent 首次上报后再操作。离线期间重置基于最近可用计数，重连后的累计增量可能包含离线期间的流量。
- 本功能提供手动重置和校准。没有设置自动月度重置计划；安装 Agent 时已有的 `--month-rotate` 选项属于 Agent 独立配置。

管理员 REST 接口：

- GET `/api/admin/client/:uuid/traffic`
- POST `/api/admin/client/:uuid/traffic/reset`
- POST `/api/admin/client/:uuid/traffic/set`，JSON `{ "upload": 0, "download": 0 }`，单位字节，两个字段必填。

对应 RPC：`admin:getClientTraffic`、`admin:resetClientTraffic`、`admin:setClientTrafficUsage`。写操作记录审计日志。普通节点编辑入口不能覆盖周期内部字段。

计量来自 Agent 上报，无法还原节点离线期间多次计数重置丢失的采样；如供应商账单口径不同，可使用手动校准对齐。
