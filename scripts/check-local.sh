#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if find .github frontend/komari-web/.github -type f -path '*/workflows/*' -print -quit | grep -q .; then
  echo 'GitHub workflows are not allowed in this fork.' >&2
  exit 1
fi
export CGO_ENABLED=1
package_list="$(go list ./...)"
mapfile -t packages < <(printf '%s\n' "$package_list" | grep -v /node_modules/)
go test "${packages[@]}"
go vet "${packages[@]}"
