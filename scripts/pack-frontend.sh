#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
test -s frontend/komari-web/dist/index.html
mkdir -p web/public/defaultTheme
archive_tmp="$(mktemp)"
trap 'rm -f "$archive_tmp"' EXIT
tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner -cf "$archive_tmp" -C frontend/komari-web/dist .
zstd -10 -T2 -q -f "$archive_tmp" -o web/public/defaultTheme/dist.tar.zst
cp frontend/komari-web/komari-theme.json web/public/defaultTheme/
