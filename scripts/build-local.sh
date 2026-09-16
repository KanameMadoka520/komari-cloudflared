#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export CGO_ENABLED=1
VERSION="${VERSION:-1.5.0-fix1-cloudflared.1}"
BUILD_HASH="${BUILD_HASH:-$(git rev-parse --short=12 HEAD)}"
if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then BUILD_HASH="${BUILD_HASH}-dirty"; fi
npm ci --no-audit --no-fund --prefix frontend/komari-web
npm run build --prefix frontend/komari-web
./scripts/pack-frontend.sh
mkdir -p build
go build -trimpath -ldflags "-s -w -X github.com/komari-monitor/komari/utils.CurrentVersion=${VERSION} -X github.com/komari-monitor/komari/utils.VersionHash=${BUILD_HASH}" -o build/komari .
