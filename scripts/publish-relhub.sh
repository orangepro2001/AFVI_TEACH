#!/usr/bin/env bash
# publish-relhub.sh - publish an AFVI installer to the release-publish (relhub) update service.
#
#   ./scripts/publish-relhub.sh [INSTALLER.exe] [--channel C] [--dry-run] [--notes FILE]
#                               [--version V] [--code N] [--app A] [--platform P] [--url U] [--token T]
#                               [--git-dir DIR] [--git-rev REF]
#
# Default installer: dist/.last-installer (written by package-wsl.sh), else the newest
# dist/AFVI-Teach-Setup-*.exe.
#
# Version rule (--version / --code override it): see scripts/version.sh, the single place the
# rule lives - build-wsl.sh stamps the same name and code into the binaries. In short: the newest
# v* tag plus its commits, patch = tagPatch*100 + N, code = MAJOR*10000 + MINOR*100 + patch, so
# v0.1.0-rc2 + 11 commits is 0.1.11 / code 111 and the next tag v0.1.1 is 0.1.100 / code 200.
# relhub refuses a code that is not higher than the newest published one; this script checks that
# beforehand and stops with the tag advice instead of letting the upload fail.
#
# Settings: -url / RELHUB_URL (default http://127.0.0.1:18790/release), -token / RELHUB_TOKEN,
# else the token file $AFVI_CI_HOME/etc/relhub-token (default ~/ci/afvi-teach/etc/relhub-token,
# mode 600, written once when the app token was created; it is a secret, never commit it).
# -git-dir reads the tags and commits from another repository (the build server's bare src.git),
# for a build whose checkout is already deleted; -git-rev names the commit (default HEAD).
#
# Idempotent: when the newest release on the channel already has this package's sha256, nothing is
# uploaded and the script exits 0. Nothing else in the release history is touched.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
# The version rule lives in scripts/version.sh, shared with build-wsl.sh: the built binaries
# stamp the same name and code into themselves, so a client and this script can never disagree.
# shellcheck source=scripts/version.sh
. "$(dirname "${BASH_SOURCE[0]}")/version.sh"

APP="afvi-teach"
PLATFORM="win-x64"
CHANNEL="stable"
URL="${RELHUB_URL:-http://127.0.0.1:18790/release}"
TOKEN="${RELHUB_TOKEN:-}"
CI_HOME="${AFVI_CI_HOME:-$HOME/ci/afvi-teach}"
NOTES_FILE=""
VERSION=""
CODE=""
DRY_RUN=false
SRC=""
GIT_ROOT=""
GIT_REV=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --channel) CHANNEL="$2"; shift 2 ;;
        --app) APP="$2"; shift 2 ;;
        --platform) PLATFORM="$2"; shift 2 ;;
        --url) URL="$2"; shift 2 ;;
        --token) TOKEN="$2"; shift 2 ;;
        --notes) NOTES_FILE="$2"; shift 2 ;;
        --version) VERSION="$2"; shift 2 ;;
        --code) CODE="$2"; shift 2 ;;
        --git-dir) GIT_ROOT="$2"; shift 2 ;;
        --git-rev) GIT_REV="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        -h|--help) sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        -*) echo "ERROR: unknown option $1" >&2; exit 2 ;;
        *) SRC="$1"; shift ;;
    esac
done

say() { printf '%s\n' "$*"; }
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

URL="${URL%/}"

# gitr runs git against --git-dir (a bare mirror works), so the version rule holds for a build whose
# checkout is already gone: the build server keeps only src.git.
GIT_ROOT=${GIT_ROOT:-$PWD}
GIT_REV=${GIT_REV:-HEAD}
gitr() { git -C "$GIT_ROOT" "$@"; }

# --- the installer ---------------------------------------------------------------------------

if [[ -z $SRC ]]; then
    if [[ -s dist/.last-installer ]]; then
        SRC=$(tr -d ' \r\n' < dist/.last-installer)
    else
        SRC=$(ls -t dist/AFVI-Teach-Setup-*.exe 2>/dev/null | head -1 || true)
    fi
fi
[[ -n $SRC && -f $SRC ]] || die "no installer found (build one first with ./scripts/build-wsl.sh, or pass the file)"
SHA=$(sha256sum "$SRC" | cut -d' ' -f1)
SIZE=$(stat -c %s "$SRC")

# --- version and versionCode -----------------------------------------------------------------

export AFVI_GIT_DIR=$GIT_ROOT AFVI_GIT_REV=$GIT_REV
[[ -z $VERSION ]] && VERSION=$(afvi_version_name)
[[ -z $CODE ]] && CODE=$(afvi_version_code)
[[ $CODE =~ ^[0-9]+$ ]] || die "--code must be a number, got '$CODE'"
[[ $CODE -ge 1 ]] || die "--code must be at least 1 (the code the Windows PC compares against)"

say "installer: $(basename "$SRC")"
say "sha256:    $SHA"
say "release:   $APP/$PLATFORM $VERSION (code $CODE) on channel $CHANNEL"

# --- token -----------------------------------------------------------------------------------

if [[ -z $TOKEN ]]; then
    token_file="$CI_HOME/etc/relhub-token"
    if [[ -r $token_file ]]; then
        TOKEN=$(tr -d ' \r\n' < "$token_file")
    fi
fi
[[ -n $TOKEN ]] || die "no token: pass --token, set RELHUB_TOKEN, or write $CI_HOME/etc/relhub-token (relhub token create -name ci -app $APP)"

# --- what is published already ---------------------------------------------------------------

# api METHOD PATH [curl args...]: sets HTTP_CODE and API_BODY (globals: no command substitution, or
# the values would be lost in the subshell). True when the status is 2xx. Without -f, so a 404 body
# (an unknown app or nothing published yet) is readable instead of vanishing as a curl error.
api() {
    local method=$1 path=$2 out
    shift 2
    API_BODY=""
    if ! out=$(curl -sS -m 60 -X "$method" "$URL$path" -w '\n%{http_code}' "$@" 2>/dev/null); then
        HTTP_CODE=000
        return 1
    fi
    HTTP_CODE=${out##*$'\n'}
    API_BODY=${out%$'\n'*}
    [[ $HTTP_CODE == 2* ]]
}

list_path="/v1/apps/$APP/$PLATFORM/releases?limit=1"
if api GET "$list_path"; then
    body=$API_BODY
    newest_code=$(jq -r '.releases[0].versionCode // 0' <<< "$body")
    newest_sha=$(jq -r '.releases[0].sha256 // ""' <<< "$body")
    newest_name=$(jq -r '.releases[0].versionName // ""' <<< "$body")
    say "published: newest is $newest_name (code $newest_code)"
    if [[ $newest_sha == "$SHA" ]]; then
        say "already published (same sha256), nothing to do"
        exit 0
    fi
    if [[ $CODE -le $newest_code ]]; then
        die "versionCode $CODE is not higher than the newest published $newest_code.
       Build after tagging (git tag v$VERSION && git push origin v$VERSION) or pass --code <higher>.
       See the version rule at the top of this script."
    fi
else
    case $HTTP_CODE in
        404) say "published: nothing yet for $APP/$PLATFORM" ;;
        000) die "cannot reach $URL (is relhub running? --url / RELHUB_URL)" ;;
        *) die "cannot list the releases of $APP/$PLATFORM (HTTP $HTTP_CODE): $API_BODY" ;;
    esac
fi

# --- release notes ---------------------------------------------------------------------------

if [[ -n $NOTES_FILE ]]; then
    NOTES=$(cat "$NOTES_FILE")
else
    sha_short=$(gitr rev-parse --short "$GIT_REV" 2>/dev/null || echo "?")
    subject=$(gitr log -1 --format=%s "$GIT_REV" 2>/dev/null || echo "")
    NOTES=$(printf '%s\n\n- 提交：%s（%s）\n- 安装包：%s（%s MiB）\n- 构建：自有构建服务器，未签名，安装包需手动运行' \
        "AFVI Teach $VERSION" "$sha_short" "$subject" "$(basename "$SRC")" \
        "$(awk -v s="$SIZE" 'BEGIN { printf "%.1f", s / 1048576 }')")
fi

if $DRY_RUN; then
    say "dry run: would publish"
    say "  POST $URL/v1/apps/$APP/$PLATFORM/releases"
    say "  fields: versionCode=$CODE versionName=$VERSION channel=$CHANNEL sha256=$SHA"
    say "  file:   $SRC"
    exit 0
fi

# --- upload ----------------------------------------------------------------------------------

# -F fields first, the file last: relhub rejects a too low versionCode before the upload starts.
resp=$(curl -sS -m 900 -X POST "$URL/v1/apps/$APP/$PLATFORM/releases" \
    -H "Authorization: Bearer $TOKEN" \
    -F "versionCode=$CODE" \
    -F "versionName=$VERSION" \
    -F "channel=$CHANNEL" \
    -F "sha256=$SHA" \
    -F "notes=$NOTES" \
    -F "file=@$SRC" \
    -w '\n%{http_code}') || die "upload failed (curl exit $?)"
code=${resp##*$'\n'}
body=${resp%$'\n'*}
case $code in
    201|200)
        say "published $VERSION (code $CODE) on $CHANNEL"
        say "  $(jq -r '.release.file // "?"' <<< "$body" 2>/dev/null || echo '?')"
        say "  $URL/v1/apps/$APP/$PLATFORM/latest?versionCode=0"
        ;;
    *)
        die "relhub answered HTTP $code: $(jq -r '.error.message // .' <<< "$body" 2>/dev/null || echo "$body")"
        ;;
esac
