# Komari Cloudflared

`komari-cloudflared` is based on the upstream project [`komari-monitor/komari`](https://github.com/komari-monitor/komari).

This branch keeps Komari's original monitoring features and adds built-in Cloudflare Tunnel support with an admin experience inspired by Uptime Kuma.

## Added Features

- Reverse Proxy / Cloudflare Tunnel settings page in the admin panel
- Save Cloudflare Tunnel token
- Start / stop `cloudflared` from the admin panel
- Show installation state, running state, recent logs, and error messages
- Auto-start `cloudflared` with `KOMARI_CLOUDFLARED_TOKEN`
- Docker image with built-in `cloudflared`

## Repository Layout

- Backend source: repository root
- Frontend source: `frontend/komari-web`
- Embedded built theme: `public/defaultTheme`

## Docker Usage

Build the image:

```bash
docker build -t komari-cloudflared:beta .
```

Run it:

```bash
docker run -d \
  -p 25774:25774 \
  -v $(pwd)/data:/app/data \
  -e ADMIN_USERNAME=admin \
  -e ADMIN_PASSWORD=your-password \
  -e KOMARI_CLOUDFLARED_TOKEN=your-cloudflare-tunnel-token \
  --name komari \
  komari-cloudflared:beta
```

## Notes

- Token is persisted in Komari config storage
- Token is encrypted at rest
- `cloudflared` is managed by Komari itself
- The runtime image uses Debian slim instead of Alpine

## Documents

- [CloudflareTunnel支持说明](./CloudflareTunnel支持说明.md)

## Frontend Development

Frontend source is included in this repository:

```text
frontend/komari-web
```

If you modify frontend logic:

1. Update code under `frontend/komari-web`
2. Build the frontend
3. Copy the new `dist` into `public/defaultTheme/dist`
4. Sync `komari-theme.json` and `preview.png` if needed

## Acknowledgements

- Upstream project: [`komari-monitor/komari`](https://github.com/komari-monitor/komari)
- UX reference: Uptime Kuma Cloudflare Tunnel / Reverse Proxy settings
