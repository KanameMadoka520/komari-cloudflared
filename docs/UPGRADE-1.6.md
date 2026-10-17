# 1.6.0 equivalent update (2026-10-17)

Upstream baselines:

- Backend: `komari-monitor/komari`, tag `1.6.0`, commit `1b76b7ad5bc3de6529bb7f0577069b4ac241fec3`.
- Frontend: `komari-monitor/komari-web`, tag `1.6.0`, commit `a96b686bff667d0b04dd0f654842a1caa7aa04c6`.
- Fork build: `1.6.0-cloudflared.1`. Inject the actual fork commit through `BUILD_HASH` when building the image.

## Changes

- Customizable admin dashboard: reorder, hide/add cards, choose widths/heights and ranking lengths, and save the layout. Additional cards show resource capacity, disk usage and shortcuts.
- Guided onboarding for server installation, workbench, notifications, markets and terminal tabs; supports small screens, dark mode, dismiss/skip and persisted progress.
- Dedicated admin login page with password, 2FA and OAuth support, preserving the original internal destination after login.
- Notification channel configuration uses the shared registry, supporting built-in and plugin channels. Existing legacy channels remain registered and retain their stored configurations.
- JavaScript notifications gain the upstream Node-compatible runtime capabilities, with files/modules confined to `data/notification/javascript`, a 30-second execution limit and the existing outbound HTTP restrictions.
- Release notes render Markdown. Traffic-limit input uses literal number/unit parsing instead of evaluating expressions; invalid or unsafe numeric values are rejected.

## Fork adaptations and fixes

- Keep cloudflared management, Cloudflare Access OAuth, Caddy media resources, real login IPs, legacy v1 agents, notification templates/timezones, load alerts, traffic reports, command clipboard and xterm settings.
- Preserve separate lifetime upload/download totals and billing-cycle resets/manual calibration. This update does not reset any node counters or rewrite credentials.
- Keep the `Javascript` provider ID and the existing per-send legacy adapter, including `sendMessage(message, title)`, optional structured `sendEvent(event)`, and text fallback. Do not force users to reconfigure notifications or remove the existing Telegram/email/Bark/ServerChan providers.
- Dashboard, theme and onboarding writes merge fresh configuration and serialize writes (including across tabs where Web Locks is available). Failed reads and active-theme changes prevent stale writes.
- Notification forms ignore obsolete fetch responses and block saving until the selected channel's configuration is loaded.
- Login redirects reject external targets and redirects back to the login page itself.
- Fix a reproduced child-process output race: drain and deliver stdout/stderr before waiting on the process and publishing its close event. The old implementation intermittently dropped output from fast processes.
- Failed notification initialization unregisters newly registered channels while preserving the pre-existing registry.
- Update stale source/build version defaults. No GitHub Actions or workflow files are included.

The backend's 15-minute metrics gap handling, WebSocket compression and previous-startup-version reporting were already equivalently incorporated before this update. They are retained, not counted as newly implemented here.

## Local validation

- `bash scripts/check-local.sh` in the Go 1.27.2 Alpine toolchain: all backend package tests and `go vet`.
- `go test ./pkg/jsruntime -run TestChildProcessPermissionAndExecution -count=500`: repeated reproduction check after fixing the output race.
- New JavaScript notification tests cover stored provider ID, Node-compatible filesystem access/confinement and structured events.
- `npm run test:upgrade --prefix frontend/komari-web`: numeric parsing, login redirects, layout recovery and merged settings writes/failures.
- `npm run build --prefix frontend/komari-web` and `bash scripts/pack-frontend.sh`: type checks and embedded default-theme build.
- `script/onboarding.browser.mjs` with local Playwright and mocked APIs: desktop/mobile/dark onboarding, terminal menu, theme preservation, EULA precedence, failed saves, dashboard persistence and login destination.

Browser test API calls are mocked and never modify production settings. Operational database backups and deployment checks stay outside the Git repository.
