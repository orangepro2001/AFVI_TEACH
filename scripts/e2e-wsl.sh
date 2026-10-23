#!/usr/bin/env bash
# e2e-wsl.sh - headless UI tests: Playwright (Chrome) against the real programs in -web mode.
#
# Builds the Linux -web binaries (bin/camera-gui-web, bin/teach-web), then runs
# frontend/e2e (mock frames + the studio and teach programs on ./reference).
# Needs node (PATH or ~/.local/node-*), frontend/node_modules, google-chrome and ./reference.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

for d in "$HOME"/.local/node-v*-linux-x64/bin; do
    [ -x "$d/node" ] && PATH="$d:$PATH"
done
command -v node >/dev/null || { echo "ERROR: node not found"; exit 1; }
[ -d reference ] || { echo "ERROR: ./reference (machine data) is missing"; exit 1; }

mkdir -p bin
CGO_ENABLED=0 GOOS=linux go build -o bin/camera-gui-web ./cmd/camera-gui
CGO_ENABLED=0 GOOS=linux go build -o bin/teach-web ./cmd/teach-imgui
CGO_ENABLED=0 GOOS=linux go build -o bin/camera ./cmd/camera
pkill -f bin/camera-gui-web 2>/dev/null || true
pkill -f bin/teach-web 2>/dev/null || true
(cd frontend && npx playwright test "$@")
