#!/usr/bin/env bash
# Render the "responsive layout" demo at three terminal sizes.
#
# VHS only honours `Set Width`/`Set Height` before the first non-setting
# command, so one tape cannot resize mid-recording — this script stamps the
# same tape body out once per size instead of keeping three near-identical
# .tape files around.
#
# Requires: vhs, ttyd, ffmpeg on PATH.
# Usage:  ./vhs/render-responsive.sh
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> building kafui"
make build >/dev/null

mkdir -p vhs/gifs

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

# name  width  height  fontsize   (px; ~80 / ~120 / ~200 columns)
sizes=(
  "narrow 700 460 14"
  "medium 1060 620 14"
  "wide   1700 840 14"
)

tape=$(mktemp -d)/size.tape
trap 'rm -rf "$(dirname "$tape")"' EXIT

for s in "${sizes[@]}"; do
  read -r name width height font <<<"$s"
  echo "==> rendering responsive-$name (${width}x${height})"
  cat > "$tape" <<EOF
Output vhs/gifs/responsive-$name.gif
Require kafui
Set Shell "bash"
Set FontSize $font
Set Width $width
Set Height $height
Set Padding 6
Set Margin 8
Set MarginFill "#1A1A2E"
Set BorderRadius 6
Set TypingSpeed 55ms
Env HOME "$DEMO_HOME"

Hide
Type "./kafui --mock --resource topics" Enter
Sleep 4s
Show

# Topics: the Name column flexes to whatever the pane can spare.
Sleep 2s
Down@300ms 4
Sleep 1500ms

# Connectors: eight columns, the widest table kafui renders.
Type ":"
Sleep 800ms
Type "connectors" Enter
Sleep 3s
Down@300ms 2
Sleep 2s

# ACLs: long principal/resource strings truncate instead of wrapping.
Type ":"
Sleep 800ms
Type "acls" Enter
Sleep 3s
Sleep 1500ms
EOF
  vhs "$tape"
done

echo "==> done: vhs/gifs/responsive-{narrow,medium,wide}.gif"
