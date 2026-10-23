#!/usr/bin/env bash
# publish-latest.sh - put an installer where the Windows side can pick it up.
#
#   ./scripts/publish-latest.sh [INSTALLER.exe]      (default: the newest dist/AFVI-Teach-Setup-*.exe)
#
# Copies the installer to $AFVI_LATEST_DIR (default ~/afvi-latest), keeps the three newest, and writes
#   LATEST.txt       name of the newest installer (one line)
#   SHA256SUMS.txt   "<sha256>  <name>" for every installer there
# scripts/fetch-latest.ps1 on the Windows PC reads these over ssh/scp. build-wsl.sh --publish-latest
# calls this after the installer is built.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
dir="${AFVI_LATEST_DIR:-$HOME/afvi-latest}"
src="${1:-}"
if [ -z "$src" ]; then
    src=$(ls -t dist/AFVI-Teach-Setup-*.exe 2>/dev/null | head -1 || true)
fi
[ -n "$src" ] && [ -f "$src" ] || { echo "ERROR: no installer found (build one first, or pass the file)"; exit 1; }
mkdir -p "$dir"
name=$(basename "$src")
cp -f "$src" "$dir/$name.part"
mv -f "$dir/$name.part" "$dir/$name"
# keep the three newest installers
ls -t "$dir"/AFVI-Teach-Setup-*.exe 2>/dev/null | tail -n +4 | xargs -r rm -f --
(cd "$dir" && sha256sum AFVI-Teach-Setup-*.exe > SHA256SUMS.txt)
printf '%s\n' "$name" > "$dir/LATEST.txt.tmp"
mv -f "$dir/LATEST.txt.tmp" "$dir/LATEST.txt"
echo "published $name to $dir"
grep -F "$name" "$dir/SHA256SUMS.txt"
