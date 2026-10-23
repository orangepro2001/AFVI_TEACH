#!/usr/bin/env bash
# build-wsl.sh - One-click build from WSL2: checks, the three programs, and the installer.
#
#   ./scripts/build-wsl.sh [--offline] [--no-installer] [--version V] [--reference DIR] [--selftest]
#                          [--commit MESSAGE] [--push] [--desktop] [--desktop-dir DIR] [--fetch]
#
# frontend build, gofmt, go vet and all tests, then bin/teach-imgui.exe, bin/camera-gui.exe and
# bin/camera.exe, then the installer dist/AFVI-Teach-Setup-<version>.exe (scripts/package-wsl.sh
# --no-build packs the programs that were just built).
#
#   --offline        GOPROXY=off (a warm module cache is needed)
#   --no-installer   stop after the three programs (scripts/package-wsl.sh calls this way)
#   --version V      version shown by the installer (default: git describe, else "dev")
#   --reference DIR  also ship this reference data tree in the installer (never included otherwise)
#
# Hand-over steps, all off by default (scripts/ship-wsl.sh does the work; run it alone for details):
#   --commit MESSAGE   once the checks passed and the programs are built, commit the working tree
#                      (ONE commit; nothing is committed when a check fails; an already staged index is refused)
#   --push             then push the branch to origin (never implied by --commit)
#   --desktop          copy the installer to the Windows desktop (sha256 checked)
#   --desktop-dir DIR  copy it to DIR instead (implies --desktop)
#   --fetch            wait for the build server's installer of the pushed commit (GitHub Release nightly,
#                      up to 15 minutes) and download it to the desktop / DIR instead of copying the local one
#
# Requires: Go; node (frontend build, optional when frontend/dist is committed); mingw-w64 windres (icons, optional)

set -euo pipefail

# Always run from the project root
cd "$(dirname "${BASH_SOURCE[0]}")/.."

SELFTEST=false
OUT="selftest-out"
OFFLINE=false
INSTALLER=true
VERSION=""
REFERENCE=""
COMMIT_MSG=""
PUSH=false
DESKTOP=false
DESKTOP_DIR=""
FETCH=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --selftest|--e2e)
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
        --commit)
            COMMIT_MSG="${2:?--commit needs a message}"
            shift 2
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --desktop)
            DESKTOP=true
            shift
            ;;
        --desktop-dir)
            DESKTOP=true
            DESKTOP_DIR="${2:?--desktop-dir needs a folder}"
            shift 2
            ;;
        --fetch)
            FETCH=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# --- Toolchain ---------------------------------------------------------------
# The programs are cgo-free now (Wails + WebView2 front end, no GLFW/OpenGL): plain Go
# cross-compiles them. MinGW is only needed for windres (application icons, optional).
export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=0

if [ "$OFFLINE" = true ]; then
    export GOPROXY=off
    export GOFLAGS="-mod=readonly"
fi

# --- 0. front end (Vue 3 + win32-vue) ----------------------------------------
# frontend/dist is embedded into the executables. Without node the committed build is used
# (CI servers without node); with node it is type-checked, unit-tested and rebuilt.
NODE_BIN=""
if command -v node &> /dev/null; then
    NODE_BIN=$(dirname "$(command -v node)")
elif [ -x "$HOME/.local/node-v24.22.1-linux-x64/bin/node" ]; then
    NODE_BIN="$HOME/.local/node-v24.22.1-linux-x64/bin"
fi
if [ -n "$NODE_BIN" ] && [ -d frontend/node_modules ]; then
    echo "==> frontend (vue-tsc, vitest, vite build)"
    (cd frontend && PATH="$NODE_BIN:$PATH" npx vue-tsc --noEmit && PATH="$NODE_BIN:$PATH" npx vitest run && PATH="$NODE_BIN:$PATH" npx vite build)
elif [ -f frontend/dist/index.html ]; then
    echo "WARNING: node or frontend/node_modules missing, using the committed frontend/dist"
else
    echo "ERROR: frontend/dist is missing and node is not available to build it"
    exit 1
fi

# --- 1. gofmt ----------------------------------------------------------------
echo "==> gofmt -l cmd internal frontend"
unformatted=$(command gofmt -l cmd internal frontend || true)
if [ -n "$unformatted" ]; then
    echo "The following files are not gofmt-clean:"
    echo "$unformatted"
    exit 1
fi

# --- 2. go vet ---------------------------------------------------------------
echo "==> go vet ./... (windows and linux)"
go vet ./...
GOOS=linux go vet ./...

# --- 3. tests (everything is pure Go now, so it all runs natively on Linux) ---
echo "==> go test ./..."
GOOS=linux GOARCH=amd64 go test ./...

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
# No MinGW runtime to link statically any more: the executables import only system DLLs
# (the installer, scripts/package-wsl.sh, still refuses lib*.dll imports). WebView2 (the
# Edge runtime) must be present on the target Windows; Windows 11 has it.
mkdir -p bin
for prog in teach-imgui camera-gui; do
    echo "==> go build $prog (cross-compile to Windows)"
    go build -tags production -ldflags "-s -w -H windowsgui" -o "bin/$prog.exe" "./cmd/$prog"
    if [ ! -f "bin/$prog.exe" ]; then
        echo "ERROR: build failed, bin/$prog.exe not found"
        exit 1
    fi
    echo "Built bin/$prog.exe ($(du -h "bin/$prog.exe" | cut -f1))"
done

echo "==> go build camera (command line)"
go build -ldflags "-s -w" -o bin/camera.exe ./cmd/camera
echo "Built bin/camera.exe ($(du -h bin/camera.exe | cut -f1))"

# --- 5. UI end-to-end tests (Linux, headless Chrome against the programs' -web mode) ----
if [ "$SELFTEST" = true ]; then
    ./scripts/e2e-wsl.sh
fi

# --- 5b. hand-over: commit and push -------------------------------------------
# Only after gofmt, vet, the tests and the builds above passed (set -e stops the script otherwise),
# and before the installer, so its version names the new commit instead of "-dirty".
if [ -n "$COMMIT_MSG" ]; then
    echo "==> commit (scripts/ship-wsl.sh commit)"
    ./scripts/ship-wsl.sh commit "$COMMIT_MSG"
fi
if [ "$PUSH" = true ]; then
    echo "==> push (scripts/ship-wsl.sh push)"
    ./scripts/ship-wsl.sh push
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

# --- 7. hand-over: installer to the desktop -------------------------------------
ship_args=()
if [ -n "$DESKTOP_DIR" ]; then
    ship_args+=(--dir "$DESKTOP_DIR")
fi
if [ "$FETCH" = true ]; then
    echo "==> fetch the server build (scripts/ship-wsl.sh fetch)"
    ./scripts/ship-wsl.sh fetch --wait 900 "${ship_args[@]}"
elif [ "$DESKTOP" = true ]; then
    if [ "$INSTALLER" != true ]; then
        echo "ERROR: --desktop needs the installer (drop --no-installer) or use --fetch"
        exit 1
    fi
    echo "==> installer to the desktop (scripts/ship-wsl.sh desktop)"
    ./scripts/ship-wsl.sh desktop "${ship_args[@]}"
fi

echo "OK"
