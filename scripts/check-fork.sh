#!/usr/bin/env bash
# These are intentional fork contracts. Upstream removals require manual review.
set -euo pipefail
cd "$(dirname "$0")/.."
for source in database/clients/traffic.go utils/messageSender/legacy_channel.go utils/cloudflared/cloudflared.go utils/notifytmpl/template.go frontend/komari-web/src/contexts/CommandClipboardContext.tsx frontend/komari-web/src/hooks/useXtermjsSettings.ts; do
  test -s "$source" || { echo "Missing fork feature: $source" >&2; exit 1; }
done
for route in /admin/notification/load /admin/notification/traffic-report /admin/notification/templates /admin/settings/xtermjs /admin/settings/reverse-proxy; do
  grep -Fq "\"$route\"" frontend/komari-web/src/config/menuConfig.json || { echo "Missing custom navigation: $route" >&2; exit 1; }
done
if git ls-files --cached --others --exclude-standard | grep -E '(^|/)\.github/workflows/' > /dev/null; then
  echo 'GitHub workflows are not allowed in this fork.' >&2; exit 1
fi
