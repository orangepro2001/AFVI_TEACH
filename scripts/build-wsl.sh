#!/usr/bin/env bash
# build-wsl.sh - Cross-compile the native AFVI teaching app from WSL2 to Windows.
# Requires: mingw-w64 (sudo apt-get install mingw-w64)

set -euo pipefail

# Always run from the project root
cd "$(dirname "${BASH_SOURCE[0]}")/.."

SELFTEST=false
OUT="selftest-out"
OFFLINE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --selftest)
            SELFTEST=true
            shift
            ;;
        --out)
            OUT="$2"
            shift 2
            ;;
        --offline)
            OFFLINE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

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

if [ "$OFFLINE" = true ]; then
    export GOPROXY=off
    export GOFLAGS="-mod=readonly"
fi

# --- 1. gofmt ----------------------------------------------------------------
echo "==> gofmt -l cmd internal"
unformatted=$(gofmt -l cmd internal || true)
if [ -n "$unformatted" ]; then
    echo "The following files are not gofmt-clean:"
    echo "$unformatted"
    exit 1
fi

# --- 2. go vet ---------------------------------------------------------------
echo "==> go vet ./..."
go vet ./...

# --- 3. config, core and inspect tests (pure Go) -----------------------------
echo "==> go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/..."
# Run tests natively on Linux (config, core, inspect, model, snap and camera are pure Go)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/...

# --- 4. build ----------------------------------------------------------------
echo "==> go build (cross-compile to Windows)"
mkdir -p bin
go build -ldflags "-s -w -H windowsgui" -o bin/teach-imgui.exe ./cmd/teach-imgui

exe="bin/teach-imgui.exe"
if [ -f "$exe" ]; then
    size_mb=$(du -h "$exe" | cut -f1)
    echo "Built $exe ($size_mb)"
else
    echo "ERROR: build failed, $exe not found"
    exit 1
fi

# --- 4b. camera (pure Go, no cgo) --------------------------------------------
echo "==> go build camera (pure Go, no cgo)"
CGO_ENABLED=0 go build -ldflags "-s -w" -o bin/camera.exe ./cmd/camera

camera_exe="bin/camera.exe"
if [ -f "$camera_exe" ]; then
    size_mb=$(du -h "$camera_exe" | cut -f1)
    echo "Built $camera_exe ($size_mb)"
else
    echo "ERROR: build failed, $camera_exe not found"
    exit 1
fi

# --- 5. optional self-test ---------------------------------------------------
if [ "$SELFTEST" = true ]; then
    echo "WARNING: Self-test requires Windows environment"
    echo "Transfer $exe to Windows and run:"
    echo "  .\\bin\\teach-imgui.exe -selftest -out $OUT"
fi

echo "OK"
