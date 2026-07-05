#!/usr/bin/env bash
# Render every kafui feature demo GIF from its .tape file.
#
# Requires: vhs, ttyd, ffmpeg on PATH (https://github.com/charmbracelet/vhs).
# Usage:  ./vhs/render-all.sh [feature ...]
#   With no args, renders all tapes. With args, renders only those
#   (e.g. ./vhs/render-all.sh acls-and-quotas kafka-connect).
set -euo pipefail

cd "$(dirname "$0")/.."

# Build the binary the tapes invoke as ./kafui (with version metadata).
echo "==> building kafui"
make build >/dev/null

mkdir -p vhs/gifs

# Seed the isolated demo HOME (see _config.tape's `Env HOME`) with known-good
# defaults so every GIF is deterministic and the real ~/.config/kafui is never
# read or written by the demos.
DEMO_HOME="/tmp/kafui-vhs-home"
rm -rf "$DEMO_HOME"
mkdir -p "$DEMO_HOME/.config/kafui"
cat > "$DEMO_HOME/.config/kafui/config.yaml" <<'YAML'
ui:
  theme: dark
  showSidebar: true
  timezone: local
releaseCheck:
  enabled: false
YAML

if [ "$#" -gt 0 ]; then
  tapes=("$@")
else
  tapes=(
    cluster-management brokers topics messages consumer-groups
    schema-registry kafka-connect ksql acls-and-quotas
    application-config metrics-and-monitoring auth-rbac-audit ui-shell
  )
fi

for t in "${tapes[@]}"; do
  echo "==> rendering $t"
  vhs "vhs/$t.tape"
done

echo "==> done: $(ls vhs/gifs/*.gif | wc -l | tr -d ' ') gifs in vhs/gifs/"
