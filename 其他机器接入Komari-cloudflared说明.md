# 其他机器接入 Komari-cloudflared 说明

本文档用于说明如何让其他 Linux、Windows、macOS 机器接入当前这套 `komari-cloudflared` 监控面板。

适用场景：

- 你已经部署好了 `komari-cloudflared` 面板
- 你希望把其他服务器、VPS、家用主机或云主机接入到这个面板里统一监控

## 一、先理解接入时真正需要的东西

其他机器接入这个监控，本质上只需要以下三项：

- 一个可以从远端机器访问到的面板地址
- 该节点专属的 `Token`
- 远端机器具备访问该地址的出站网络能力

其中最重要的是第一项：

- 如果远端机器不在这台面板主机本机上，绝对不能使用 `http://localhost:25774`
- `localhost` 只代表“当前这台机器自己”
- 远端机器必须使用公网域名、局域网 IP、反代域名，或者 Cloudflare Tunnel 提供的公网地址

如果你是通过 Cloudflare Tunnel 或 Caddy 对外提供面板访问，那么远端机器应当使用你最终实际访问后台的那个地址，例如：

```text
https://monitor.example.com
```

而不是：

```text
http://localhost:25774
```

## 二、面板端需要先做什么

在远端机器接入之前，面板端需要完成以下准备。

### 1. 确认面板地址可被远端机器访问

你需要先确认以下其中一种地址从远端机器可以访问：

- 公网域名
- 公网 IP
- 局域网 IP
- Cloudflare Tunnel 公网域名

建议优先使用域名，尤其是你已经通过 Cloudflare Tunnel 对外发布时。

### 2. 建议设置正确的脚本域名

Komari 后台会根据当前访问地址或 `script_domain` 来生成一键安装命令。

为了避免后台误生成 `localhost`、内网地址或临时地址，建议在后台设置中把脚本域名固定为你的最终访问地址。

建议填写为：

```text
https://你的监控域名
```

例如：

```text
https://monitor.example.com
```

这样之后后台生成的 Agent 安装命令就会稳定使用这个地址。

### 3. 在后台新增节点

进入后台节点管理页面后：

1. 新增一个节点
2. 给节点命名
3. 系统会为该节点生成唯一 `Token`
4. 后台可直接生成该节点的一键安装命令

注意：

- 每个节点都应使用自己的独立 `Token`
- 不建议多个不同机器共用同一个节点 Token
- Token 属于敏感信息，不要随意发到公开群、工单截图或公共仓库中

## 三、远端机器需要做什么

远端机器通常只需要满足以下条件：

- 能访问你的 `komari-cloudflared` 面板地址
- 能访问 GitHub 原始脚本地址，或者你配置的脚本代理地址
- 有管理员权限或 sudo 权限用于安装 Agent

一般不需要：

- 在远端机器上额外开放入站端口
- 手动安装数据库
- 手动配置 Web 服务

大多数情况下，远端机器只需要“主动向面板上报数据”，不需要被动暴露监听端口给公网。

## 四、推荐接入方式：后台生成一键安装命令

这是最简单、最推荐的方式。

操作流程：

1. 在后台新增节点
2. 打开该节点的安装命令对话框
3. 选择目标系统：Linux / Windows / macOS
4. 复制后台生成的一键安装命令
5. 去目标机器执行

后台生成的命令会自动带上两个关键参数：

- `-e`
  面板地址
- `-t`
  节点 Token

你只需要保证这个地址和 Token 是正确的即可。

## 五、不同系统的典型接入方式

以下内容用于帮助你理解后台生成命令的大致结构。实际使用时，优先以后台复制出来的命令为准。

### Linux

典型形式如下：

```bash
wget -qO- https://raw.githubusercontent.com/komari-monitor/komari-agent/refs/heads/main/install.sh | sudo bash -s -- -e https://你的监控域名 -t 你的节点Token
```

例如：

```bash
wget -qO- https://raw.githubusercontent.com/komari-monitor/komari-agent/refs/heads/main/install.sh | sudo bash -s -- -e https://monitor.example.com -t abcdefghijklmnopqrstuvwxyz
```

适用前提：

- 机器能访问 `raw.githubusercontent.com`
- 有 `sudo`
- 系统支持常见服务管理方式

### Windows

典型形式如下：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "iwr 'https://raw.githubusercontent.com/komari-monitor/komari-agent/refs/heads/main/install.ps1' -UseBasicParsing -OutFile 'install.ps1'; & '.\install.ps1' '-e' 'https://你的监控域名' '-t' '你的节点Token'"
```

建议：

- 使用管理员权限打开 PowerShell
- 首次安装时尽量关闭一些会拦截脚本执行的安全软件白名单限制

### macOS

典型形式如下：

```bash
zsh <(curl -sL https://raw.githubusercontent.com/komari-monitor/komari-agent/refs/heads/main/install.sh) -e https://你的监控域名 -t 你的节点Token
```

## 六、接入完成后会发生什么

远端机器安装并启动 Agent 后，会主动向面板上报以下信息：

- 操作系统
- CPU / 架构
- 内存
- 磁盘
- 网络流量
- 负载
- 温度
- 进程数
- 节点基础信息和定时性能数据

正常情况下，你在后台或首页节点列表里会看到：

- 节点出现在列表中
- 节点状态变为在线
- 开始刷新 CPU、内存、流量等监控数据

## 七、如果你使用的是 Cloudflare Tunnel

如果面板本身是通过 Cloudflare Tunnel 对外访问，那么远端机器应该使用 Tunnel 对应的公网域名接入。

正确示例：

```text
https://monitor.example.com
```

错误示例：

```text
http://localhost:25774
http://127.0.0.1:25774
http://192.168.x.x:25774
```

说明：

- `localhost` 和 `127.0.0.1` 只代表目标机器自己
- `192.168.x.x` 这类地址只有在远端机器与面板处于同一内网时才有意义
- 跨公网接入时，应使用公开可访问的域名

## 八、网络与防火墙要求

### 面板端

面板端需要保证：

- 远端机器可以访问到你的面板地址
- 反向代理、Caddy、Cloudflare Tunnel 或端口映射工作正常

### 远端机器

远端机器通常只需要允许出站访问：

- 你的监控面板地址
- GitHub 原始脚本地址，或者你设置的脚本代理地址

通常不需要在远端机器额外开放入站端口。

## 九、常见问题

### 1. 为什么不能用 `localhost`

因为 `localhost` 永远表示“当前这台机器自己”。

如果你在远端机器上执行：

```text
http://localhost:25774
```

那它访问的是远端机器自己的 25774 端口，不是你的监控面板。

### 2. 远端机器连不上面板怎么办

优先检查这些内容：

- 面板域名能否从远端机器打开
- Cloudflare Tunnel 是否正常运行
- Caddy / 反代是否正常
- 防火墙是否阻断访问
- 后台生成的命令里 `-e` 地址是否写错

### 3. 节点创建了但一直离线怎么办

重点检查：

- Token 是否对应这个节点
- 面板地址是否写错
- 远端机器能否访问该地址
- Agent 服务是否真正启动成功

### 4. 远端机器无法访问 GitHub 原始脚本怎么办

你可以：

- 在后台设置里配置 `script_domain`
- 在安装命令里使用你自己的脚本代理地址
- 手动下载脚本后再执行

## 十、自动发现模式

如果你有批量部署需求，可以使用自动发现功能。

自动发现的大致流程是：

1. 在后台设置中配置 `auto_discovery_key`
2. 远端机器使用这个 Key 调用注册接口
3. 面板自动创建节点并返回 `uuid` 和 `token`
4. 远端机器再使用返回的 `token` 正式接入

这更适合批量注册、自动化运维或云初始化脚本。

如果你只是接入少量机器，仍然建议优先使用“手动创建节点 + 后台复制一键安装命令”的方式，最直观，也最不容易出错。

## 十一、安全建议

- 节点 Token 不要公开传播
- 不要把 Token 发到公开聊天群或截图里
- 一台机器一个 Token，尽量不要复用
- 如果怀疑 Token 泄露，建议在后台为该节点重新生成或替换 Token
- 尽量使用 HTTPS 域名或 Cloudflare Tunnel 域名接入，不建议长期直接裸露明文 HTTP 公网地址

## 十二、推荐的实际做法

对你当前这套 `komari-cloudflared` 部署，最稳妥的接入方式是：

1. 先用一个固定公网域名访问面板
2. 在后台设置里把脚本域名也设置成这个固定域名
3. 每新增一台机器，就在后台新建一个独立节点
4. 直接复制后台生成的一键安装命令去目标机器执行

这样后续维护、替换机器、排查故障都会更清晰。
