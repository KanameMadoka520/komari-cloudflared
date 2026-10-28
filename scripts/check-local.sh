#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
bash scripts/check-fork.sh
export CGO_ENABLED=1
package_list="$(go list ./...)"
mapfile -t packages < <(printf '%s\n' "$package_list" | grep -v /node_modules/)
go test "${packages[@]}"
go vet "${packages[@]}"
