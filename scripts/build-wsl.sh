#!/usr/bin/env bash
# build-wsl.sh - One-click build from WSL2: checks, the three programs, and the installer.
#
#   ./scripts/build-wsl.sh [--offline] [--no-installer] [--version V] [--reference DIR] [--selftest] [--out DIR]
#
# gofmt, go vet and the pure-Go tests, then bin/teach-imgui.exe, bin/camera-gui.exe and
# bin/camera.exe, then the installer dist/AFVI-Teach-Setup-<version>.exe (scripts/package-wsl.sh
# --no-build packs the programs that were just built).
#
#   --offline        GOPROXY=off (a warm module cache is needed)
#   --no-installer   stop after the three programs (scripts/package-wsl.sh calls this way)
#   --version V      version shown by the installer (default: git describe, else "dev")
#   --reference DIR  also ship this reference data tree in the installer (never included otherwise)
#
# Requires: mingw-w64 (sudo apt-get install mingw-w64)

set -euo pipefail

# Always run from the project root
cd "$(dirname "${BASH_SOURCE[0]}")/.."

SELFTEST=false
OUT="selftest-out"
OFFLINE=false
INSTALLER=true
VERSION=""
REFERENCE=""

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
        --no-installer)
            INSTALLER=false
            shift
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --reference)
            REFERENCE="$2"
            shift 2
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
echo "==> go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/... ./internal/link/... ./internal/appicon/... ./internal/installer/... ./internal/access/... ./internal/golden/..."
# Run tests natively on Linux (config, core, inspect, model, snap, camera, link, appicon and installer are pure Go)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test ./internal/config/... ./internal/core/... ./internal/inspect/... ./internal/model/... ./internal/snap/... ./internal/camera/... ./internal/link/... ./internal/appicon/... ./internal/installer/... ./internal/access/... ./internal/golden/...

# --- 3b. application icons (Windows resources) -------------------------------
# The icon resource scripts need windres; without it the executables simply get the default icon.
if command -v x86_64-w64-mingw32-windres &> /dev/null; then
    echo "==> windres (application icons)"
    x86_64-w64-mingw32-windres -O coff -o cmd/teach-imgui/rsrc_windows_amd64.syso cmd/teach-imgui/teach.rc
    x86_64-w64-mingw32-windres -O coff -o cmd/camera-gui/rsrc_windows_amd64.syso cmd/camera-gui/studio.rc
else
    echo "WARNING: x86_64-w64-mingw32-windres not found, the executables get no icon"
fi

# --- 4. build ----------------------------------------------------------------
# The GUIs link the MinGW runtime statically (-extldflags=-static): otherwise they import
# libgcc_s_seh-1.dll and libstdc++-6.dll, which a Windows without MinGW does not have, and the
# installer (scripts/package-wsl.sh) refuses such executables.
echo "==> go build (cross-compile to Windows)"
mkdir -p bin
go build -ldflags "-s -w -H windowsgui -extldflags=-static" -o bin/teach-imgui.exe ./cmd/teach-imgui

exe="bin/teach-imgui.exe"
if [ -f "$exe" ]; then
    size_mb=$(du -h "$exe" | cut -f1)
    echo "Built $exe ($size_mb)"
else
    echo "ERROR: build failed, $exe not found"
    exit 1
fi

# --- 4b. camera-gui (studio, cgo like the teach program) ---------------------
echo "==> go build camera-gui (cross-compile to Windows)"
go build -ldflags "-s -w -H windowsgui -extldflags=-static" -o bin/camera-gui.exe ./cmd/camera-gui

studio_exe="bin/camera-gui.exe"
if [ -f "$studio_exe" ]; then
    size_mb=$(du -h "$studio_exe" | cut -f1)
    echo "Built $studio_exe ($size_mb)"
else
    echo "ERROR: build failed, $studio_exe not found"
    exit 1
fi

# --- 4c. camera (pure Go, no cgo) --------------------------------------------
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
    echo "Transfer the executables to Windows and run:"
    echo "  .\\bin\\teach-imgui.exe -selftest -out $OUT"
    echo "  .\\bin\\camera-gui.exe -selftest -out $OUT"
fi

# --- 6. installer -------------------------------------------------------------
# One self-extracting setup.exe with the three programs, the documents and the example
# configuration. package-wsl.sh needs the programs built above, so it must not build them again.
if [ "$INSTALLER" = true ]; then
    echo "==> installer (scripts/package-wsl.sh --no-build)"
    pkg_args=(--no-build)
    if [ "$OFFLINE" = true ]; then
        pkg_args+=(--offline)
    fi
    if [ -n "$VERSION" ]; then
        pkg_args+=(--version "$VERSION")
    fi
    if [ -n "$REFERENCE" ]; then
        pkg_args+=(--reference "$REFERENCE")
    fi
    ./scripts/package-wsl.sh "${pkg_args[@]}"
fi

echo "OK"
