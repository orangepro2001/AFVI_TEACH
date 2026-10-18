#!/usr/bin/env bash
# build-wsl.sh - Cross-compile teach-imgui-spike from WSL2 to Windows.
# Requires: mingw-w64 (sudo apt-get install mingw-w64)

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

# --- Toolchain: MinGW-w64 cross-compiler -------------------------------------
if ! command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    echo "ERROR: x86_64-w64-mingw32-gcc not found."
    echo "Install with: sudo apt-get install mingw-w64"
    exit 1
fi

export CGO_ENABLED=1
export GOOS=windows
export GOARCH=amd64
export CC=x86_64-w64-mingw32-gcc
export CXX=x86_64-w64-mingw32-g++

echo "==> go build (cross-compile to Windows)"
go build -ldflags "-s -w -H windowsgui" -o teach-imgui-spike.exe .

exe="teach-imgui-spike.exe"
if [ -f "$exe" ]; then
    size_mb=$(du -h "$exe" | cut -f1)
    echo "Built $exe ($size_mb)"
else
    echo "ERROR: build failed, $exe not found"
    exit 1
fi

echo "OK"
echo ""
echo "Transfer to Windows and run:"
echo "  .\\$exe"
echo "  .\\$exe -ref <型号目录>"
echo "  .\\$exe -selftest -out <目录>"
