#!/usr/bin/env bash
# package-wsl.sh - Build the installer dist/AFVI-Teach-Setup-<version>.exe from WSL2/Linux.
#
# The installer is one self-extracting Windows program (cmd/setup, pure Go) with the payload
# appended by cmd/mkinstaller: teach-imgui.exe, camera-gui.exe, camera.exe, the documents and the
# example configuration, optionally the reference data.
#
#   ./scripts/package-wsl.sh [--offline] [--no-build] [--version V] [--out FILE] [--reference DIR]
#
#   --offline          GOPROXY=off (a warm module cache is needed)
#   --no-build         use the executables in bin/ as they are (default: run scripts/build-wsl.sh
#                      --no-installer first; build-wsl.sh itself ends by calling this script with --no-build)
#   --version V        version shown by the installer (default: git describe, else "dev")
#   --out FILE         setup executable to write (default: dist/AFVI-Teach-Setup-<version>.exe)
#   --reference DIR    also ship this reference data tree (it is machine specific and may be
#                      confidential, so it is never included unless you ask for it)
#
# Requires what build-wsl.sh requires (mingw-w64, for the GUIs and windres).

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

OFFLINE=false
BUILD=true
VERSION=""
OUT=""
REFERENCE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --offline) OFFLINE=true; shift ;;
        --no-build) BUILD=false; shift ;;
        --version) VERSION="$2"; shift 2 ;;
        --out) OUT="$2"; shift 2 ;;
        --reference) REFERENCE="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [ -z "$VERSION" ]; then
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
fi
# The version ends up in a file name and in the registry: keep it plain.
VERSION=$(printf '%s' "$VERSION" | tr -c 'A-Za-z0-9._+-' '-')
if [ -z "$OUT" ]; then
    OUT="dist/AFVI-Teach-Setup-${VERSION}.exe"
fi
if [ -n "$REFERENCE" ] && [ ! -d "$REFERENCE" ]; then
    echo "ERROR: --reference $REFERENCE is not a folder"
    exit 1
fi

if [ "$OFFLINE" = true ]; then
    export GOPROXY=off
    export GOFLAGS="-mod=readonly"
fi

# --- 1. the programs ---------------------------------------------------------
if [ "$BUILD" = true ]; then
    if [ "$OFFLINE" = true ]; then
        ./scripts/build-wsl.sh --offline --no-installer
    else
        ./scripts/build-wsl.sh --no-installer
    fi
fi
for exe in teach-imgui camera-gui camera; do
    if [ ! -f "bin/$exe.exe" ]; then
        echo "ERROR: bin/$exe.exe is missing (run without --no-build)"
        exit 1
    fi
done

# --- 2. the setup stub (pure Go, no cgo) -------------------------------------
mkdir -p dist
if command -v x86_64-w64-mingw32-windres &> /dev/null; then
    echo "==> windres (setup icon and manifest)"
    x86_64-w64-mingw32-windres -O coff -o cmd/setup/rsrc_windows_amd64.syso cmd/setup/setup.rc
else
    echo "ERROR: x86_64-w64-mingw32-windres not found (apt-get install mingw-w64)."
    echo "Without the manifest in setup.exe Windows asks for administrator rights."
    exit 1
fi
echo "==> go build setup stub"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H windowsgui" -o dist/setup-stub.exe ./cmd/setup

# --- 3. append the payload (runs on the build machine) ------------------------
echo "==> mkinstaller"
args=(-root . -bin bin -stub dist/setup-stub.exe -out "$OUT" -version "$VERSION")
if [ -n "$REFERENCE" ]; then
    args+=(-reference "$REFERENCE")
fi
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go run ./cmd/mkinstaller "${args[@]}"
rm -f dist/setup-stub.exe
chmod +x "$OUT"
# ship-wsl.sh desktop picks the installer up from here.
printf '%s\n' "$OUT" > dist/.last-installer

echo "OK: $OUT"
