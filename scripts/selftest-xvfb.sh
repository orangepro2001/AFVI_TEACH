#!/usr/bin/env bash
# selftest-xvfb.sh - run the ImGui programs' -selftest natively on Linux inside Xvfb.
#
# Builds teach-imgui and camera-gui for Linux (cgo, needs libgl/x11 dev headers), starts a
# private Xvfb with Mesa software GL and runs both self-tests; screenshots go through
# xdotool + ImageMagick `import` (capture_other.go). GLFW prefers Wayland when a socket
# exists, so XDG_RUNTIME_DIR points at an empty directory to force X11 (otherwise the windows
# appear on the real desktop).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
REF="${AFVI_REF:-$PWD/reference}"
OUT="${1:-selftest-out}"
mkdir -p bin "$OUT"
CGO_ENABLED=1 go build -o bin/teach-linux ./cmd/teach-imgui
CGO_ENABLED=1 go build -o bin/studio-linux ./cmd/camera-gui
CGO_ENABLED=0 go build -o bin/camera ./cmd/camera
XDG=$(mktemp -d); chmod 700 "$XDG"
Xvfb :77 -screen 0 1700x1100x24 >/dev/null 2>&1 & XPID=$!
trap 'kill $XPID 2>/dev/null; rm -rf "$XDG"' EXIT
sleep 2
export DISPLAY=:77 XDG_RUNTIME_DIR="$XDG" LIBGL_ALWAYS_SOFTWARE=1 AFVI_CAMERA_EXE="$PWD/bin/camera" AFVI_CAPTURES="$OUT/captures"
./bin/teach-linux -selftest -ref "$REF" -out "$OUT/teach"
./bin/studio-linux -selftest -ref "$REF" -out "$OUT/studio"
