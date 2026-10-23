#!/usr/bin/env bash
# qt-test.sh - build the Qt front ends and run the scripted UI tests offscreen (no display needed).
#
# Each qt-tests/*.json drives one program: commands go through the host, "click" presses the
# real Qt button, "text" asserts a label is on screen, "shot" saves the window as PNG.
# Needs qt6-base-dev and the miqt module (first compile of miqt takes ~4 minutes).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
OUT="$(realpath -m "${1:-qt-out}")"; mkdir -p "$OUT" bin
# reference/ is git-ignored and lives in the main checkout (also when run from a worktree)
REF="${AFVI_REF:-$(realpath "$(git rev-parse --git-common-dir)/..")/reference}"
CGO_ENABLED=1 go build -o bin/teach-qt ./cmd/teach-qt
CGO_ENABLED=1 go build -o bin/studio-qt ./cmd/studio-qt
CGO_ENABLED=0 go build -o bin/camera ./cmd/camera
export QT_QPA_PLATFORM=offscreen AFVI_CAMERA_EXE="$PWD/bin/camera" AFVI_QT_OUT="$OUT"
rm -rf /tmp/qt-test-captures
AFVI_CAPTURES=/tmp/qt-test-captures/s AFVI_QT_SCRIPT=qt-tests/studio.json ./bin/studio-qt -ref "$REF" -link off -login admin:1 2>&1 | grep -v propagateSizeHints || true
AFVI_CAPTURES=/tmp/qt-test-captures/t AFVI_QT_SCRIPT=qt-tests/teach.json ./bin/teach-qt -ref "$REF" -link off 2>&1 | grep -v propagateSizeHints || true
echo "screenshots in $OUT"
