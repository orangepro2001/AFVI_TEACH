#!/usr/bin/env bash
# ship-wsl.sh - Hand work over: commit the working tree, push it, put an installer on the Windows desktop.
#
#   ./scripts/ship-wsl.sh commit MESSAGE
#   ./scripts/ship-wsl.sh push
#   ./scripts/ship-wsl.sh desktop [--file FILE] [--dir DIR]
#   ./scripts/ship-wsl.sh fetch   [--ref nightly|vX.Y.Z] [--wait SECONDS] [--any] [--dir DIR]
#
#   commit   Stage the working tree (tracked changes, deletions and new files that .gitignore does not
#            hide; never an already staged index, a file over 5 MB or a detached HEAD) and create ONE
#            commit with MESSAGE. No amend, no --no-verify. Nothing to commit is not an error.
#   push     git push origin <current branch>. No force.
#   desktop  Copy the installer built last (dist/.last-installer, written by package-wsl.sh) or --file
#            to the Windows desktop (asked from Windows, so a OneDrive-redirected desktop works) or --dir,
#            and check the copy by its sha256.
#   fetch    Download the installer the build server published (GitHub Release --ref, default the rolling
#            prerelease "nightly") and verify it against the release's SHA256SUMS.txt. For nightly the
#            installer must belong to the commit at HEAD (--any skips that check); --wait polls up to
#            SECONDS while the server is still building. Needs only curl and sha256sum.
#
# build-wsl.sh calls these through --commit, --push, --desktop and --fetch. The repository comes from
# origin (override with AFVI_REPO=owner/name). Commit and push only run when you ask for them.

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

die() {
    echo "ERROR: $*" >&2
    exit 1
}

usage() {
    sed -n '2,/^$/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

current_branch() {
    git symbolic-ref --short -q HEAD || die "detached HEAD, switch to a branch first"
}

# --- commit ------------------------------------------------------------------
cmd_commit() {
    local msg=${1:-}
    [ -n "$msg" ] || die "commit needs a message"
    git rev-parse --is-inside-work-tree > /dev/null 2>&1 || die "not a git repository"
    current_branch > /dev/null
    git diff --cached --quiet || die "the index already holds staged changes; commit or unstage them first (this commits the working tree and nothing else)"

    local added=() f
    mapfile -d '' added < <(git ls-files -z -o --exclude-standard)
    if ((${#added[@]})); then
        for f in "${added[@]}"; do
            [ "$(stat -c %s -- "$f")" -le 5242880 ] || die "new file $f is larger than 5 MB; git add it yourself if it belongs in the repository"
        done
        git add -- "${added[@]}"
    fi
    git add -u -- .

    if git diff --cached --quiet; then
        echo "Nothing to commit."
        return 0
    fi
    git diff --cached --stat
    git commit -q -m "$msg"
    git log -1 --oneline
}

# --- push --------------------------------------------------------------------
cmd_push() {
    local branch
    branch=$(current_branch)
    git push origin "$branch"
}

# --- desktop -----------------------------------------------------------------
desktop_dir() {
    local win
    win=$(cd /mnt/c 2> /dev/null && powershell.exe -NoProfile -NonInteractive -Command "[Environment]::GetFolderPath('Desktop')" 2> /dev/null | tr -d '\r') || true
    [ -n "$win" ] || die "cannot ask Windows for the desktop folder (powershell.exe missing?); pass --dir"
    local dir
    dir=$(wslpath -u "$win")
    [ -d "$dir" ] || die "desktop folder $win ($dir) does not exist"
    printf '%s\n' "$dir"
}

# place_file SRC DIR: copy through a temporary name, compare the sha256, then rename into place.
place_file() {
    local src=$1 dir=$2 name tmp want got
    [ -f "$src" ] || die "$src does not exist"
    mkdir -p -- "$dir"
    name=$(basename -- "$src")
    tmp="$dir/.$name.part"
    cp -- "$src" "$tmp"
    want=$(sha256sum -- "$src" | cut -d' ' -f1)
    got=$(sha256sum -- "$tmp" | cut -d' ' -f1)
    if [ "$want" != "$got" ]; then
        rm -f -- "$tmp"
        die "the copy of $name is damaged (sha256 differs)"
    fi
    mv -f -- "$tmp" "$dir/$name"
    echo "OK: $(wslpath -w "$dir/$name" 2> /dev/null || echo "$dir/$name")  sha256 ${got:0:16}...  $(du -h -- "$dir/$name" | cut -f1)"
}

cmd_desktop() {
    local file="" dir=""
    while [[ $# -gt 0 ]]; do
        case $1 in
            --file) file=${2:?--file needs a path}; shift 2 ;;
            --dir) dir=${2:?--dir needs a folder}; shift 2 ;;
            *) die "desktop: unknown option $1" ;;
        esac
    done
    if [ -z "$file" ]; then
        [ -f dist/.last-installer ] || die "no installer recorded (dist/.last-installer); build one first or pass --file"
        file=$(head -n 1 dist/.last-installer)
    fi
    [ -n "$dir" ] || dir=$(desktop_dir)
    place_file "$file" "$dir"
}

# --- fetch -------------------------------------------------------------------
repo_slug() {
    local url slug
    url=${AFVI_REPO:-$(git remote get-url origin 2> /dev/null || true)}
    slug=$(printf '%s' "$url" | sed -E 's#^(git@github\.com:|https?://github\.com/|ssh://git@github\.com/)##; s#\.git$##')
    [[ $slug =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || die "cannot find the GitHub repository (origin is '$url'); set AFVI_REPO=owner/name"
    printf '%s\n' "$slug"
}

cmd_fetch() {
    local ref=nightly wait=0 any=false dir=""
    while [[ $# -gt 0 ]]; do
        case $1 in
            --ref) ref=${2:?--ref needs a release tag}; shift 2 ;;
            --wait) wait=${2:?--wait needs seconds}; shift 2 ;;
            --any) any=true; shift ;;
            --dir) dir=${2:?--dir needs a folder}; shift 2 ;;
            *) die "fetch: unknown option $1" ;;
        esac
    done
    [[ $wait =~ ^[0-9]+$ ]] || die "--wait needs a number of seconds"
    [[ $ref =~ ^[A-Za-z0-9._-]+$ ]] || die "bad release tag: $ref"

    local base head deadline why line name want sha got
    base="https://github.com/$(repo_slug)/releases/download/$ref"
    head=$(git rev-parse HEAD)
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT
    deadline=$(($(date +%s) + wait))

    while :; do
        why=""
        if curl -fsSL --retry 2 -m 30 -o "$TMP_DIR/SHA256SUMS.txt" "$base/SHA256SUMS.txt" 2> /dev/null; then
            line=$(grep -E '^[0-9a-f]{64} [ *]AFVI-Teach-Setup-[^ ]+\.exe$' "$TMP_DIR/SHA256SUMS.txt" | head -n 1 || true)
            want=$(printf '%s' "$line" | awk '{print $1}')
            name=$(printf '%s' "$line" | awk '{print $2}' | sed 's/^\*//')
            sha=$(printf '%s' "$name" | sed -nE 's/.*-g([0-9a-f]{7,40})\.exe$/\1/p')
            if [ -z "$name" ]; then
                why="SHA256SUMS.txt of release '$ref' lists no installer"
            elif [ "$any" = true ] || [ "$ref" != nightly ] || { [ -n "$sha" ] && [[ $head == "$sha"* ]]; }; then
                break
            else
                why="release '$ref' holds $name, not commit ${head:0:7} (not pushed yet, or the server is still building)"
            fi
        else
            why="release '$ref' is not published (yet) or is being replaced"
        fi
        if [ "$(date +%s)" -ge "$deadline" ]; then
            die "$why"
        fi
        echo "waiting: $why"
        sleep 15
    done

    echo "==> downloading $name"
    curl -fsSL --retry 3 -m 600 -o "$TMP_DIR/$name" "$base/$name"
    got=$(sha256sum -- "$TMP_DIR/$name" | cut -d' ' -f1)
    [ "$got" = "$want" ] || die "$name does not match SHA256SUMS.txt (got $got, want $want)"
    [ -n "$dir" ] || dir=$(desktop_dir)
    place_file "$TMP_DIR/$name" "$dir"
}

# --- main --------------------------------------------------------------------
case ${1:-} in
    commit) shift; cmd_commit "${1:-}" ;;
    push) shift; cmd_push ;;
    desktop) shift; cmd_desktop "$@" ;;
    fetch) shift; cmd_fetch "$@" ;;
    -h | --help | help) usage ;;
    *) usage >&2; exit 1 ;;
esac
