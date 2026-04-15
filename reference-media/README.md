# reference-media

这个目录用于存放当前 `komari-cloudflared` 默认部署里通过 Caddy 对外提供的静态媒体资源。

默认用途包括：

- 横屏动态背景视频
- 竖屏/窄屏动态背景视频
- 视频封面图
- 头像挂件图
- 其他通过 `/media/*` 引用的主题资源

在默认部署结构中：

- 浏览器访问 `/media/*`
- 会先到 Caddy
- Caddy 再从本目录读取对应文件

例如：

- `/media/desktop-snare.mp4`
- `/media/mobile-plum.mp4`
- `/media/tcymc-avatar.png`

注意：

- 仓库默认不提交实际的大媒体文件，只保留目录和说明
- 你应当把自己的视频、图片手动放进这个目录
- 如需保持当前主题效果，请确保你在视觉配置中引用的文件名，和本目录实际存在的文件名一致
