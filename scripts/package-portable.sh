#!/usr/bin/env bash
# package-portable.sh - build the green (no-install) distribution: dist/AFVI-Teach-Portable-<version>.zip
#
#   ./scripts/package-portable.sh [--no-build] [--version V] [--out FILE] [--reference DIR]
#
# The zip is unpacked and run in place: no registry entry, no shortcuts, no uninstaller, nothing
# outside its own folder. It carries the same three programs and documents as the installer, the
# example configuration and a portable.txt marker - the marker is what tells the programs that a
# new version replaces these files directly instead of running the installer.
#
# Update identity: the portable build asks the update service on platform win-x64-portable and
# downloads a new zip (scripts/publish-relhub.sh publishes both platforms), so the two
# distributions never offer each other's package.
#
#   --no-build       pack the programs that are already in bin/ (build-wsl.sh calls this way)
#   --version V      version shown in the file name (default: git describe, else "dev")
#   --out FILE       zip to write (default dist/AFVI-Teach-Portable-<version>.zip)
#   --reference DIR  also ship this reference data tree (never included otherwise)
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
# shellcheck source=scripts/version.sh
. scripts/version.sh

NO_BUILD=false
VERSION=""
OUT=""
REFERENCE=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-build) NO_BUILD=true; shift ;;
        --version) VERSION="$2"; shift 2 ;;
        --out) OUT="$2"; shift 2 ;;
        --reference) REFERENCE="$2"; shift 2 ;;
        -h|--help) sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) echo "ERROR: unknown option $1" >&2; exit 2 ;;
    esac
done

if [ -z "$VERSION" ]; then
    VERSION=$(git describe --tags --always --dirty 2> /dev/null || echo dev)
fi
VERSION=$(printf '%s' "$VERSION" | tr -c 'A-Za-z0-9._+-' '-')
if [ -z "$OUT" ]; then
    OUT="dist/AFVI-Teach-Portable-${VERSION}.zip"
fi

if [ "$NO_BUILD" != true ]; then
    ./scripts/build-wsl.sh --no-installer
fi
for exe in teach-imgui.exe camera-gui.exe camera.exe; do
    [ -f "bin/$exe" ] || {
        echo "ERROR: bin/$exe is missing (build first, or drop --no-build)"
        exit 1
    }
done
command -v zip > /dev/null || {
    echo "ERROR: zip is not installed (apt-get install zip)"
    exit 1
}

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$STAGE/docs"
for exe in teach-imgui.exe camera-gui.exe camera.exe; do
    cp -f "bin/$exe" "$STAGE/$exe"
done
cp -f configs/afvi.example.toml "$STAGE/afvi.example.toml"
# the same documents the installer carries (cmd/mkinstaller/main.go holds that list)
cp -f README.md "$STAGE/docs/README.md"
for doc in MANUAL USAGE ROLES GOLDEN LINK INSPECT_RULES SCENARIO REFERENCE_SETUP UPDATE PORTABLE; do
    [ -f "docs/$doc.md" ] && cp -f "docs/$doc.md" "$STAGE/docs/$doc.md"
done
cat > "$STAGE/portable.txt" <<'TXT'
AFVI Teach - green (no-install) build.

Unpack this folder anywhere writable and run teach-imgui.exe. Nothing is registered, no shortcut
is created and nothing is written outside this folder: the configuration (afvi.toml), the camera
output (captures) and downloaded updates (updates) all live next to the program.

This file is also the marker that tells the program it is the portable build: an update downloads
the new zip, waits for the program to close, replaces the files in this folder and starts the new
version - the installer is never involved. Delete this file and the program will treat itself as
an installed copy again.
TXT
if [ -n "$REFERENCE" ]; then
    [ -d "$REFERENCE" ] || {
        echo "ERROR: --reference $REFERENCE is not a directory"
        exit 1
    }
    mkdir -p "$STAGE/reference"
    cp -a "$REFERENCE/." "$STAGE/reference/"
fi

mkdir -p "$(dirname "$OUT")"
rm -f "$OUT"
(cd "$STAGE" && zip -q -r -X "$OLDPWD/$OUT" .)
printf '%s\n' "$OUT" > dist/.last-portable
echo "OK: $OUT"
unzip -l "$OUT" | tail -3
