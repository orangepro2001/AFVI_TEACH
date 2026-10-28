#!/usr/bin/env bash
# version.sh - the ONE place the release version rule lives. Sourced, not run.
#
#   . scripts/version.sh
#   afvi_version_name [rev]     -> 0.1.11
#   afvi_version_code [rev]     -> 111
#
# Rule (both the built binaries and the update service must agree on it, or every client
# offers itself an update for ever):
#   base  = the newest v* tag reachable from rev, its MAJOR.MINOR.PATCH as-is; a prerelease
#           suffix names the tag only (v0.1.0-rc2 -> 0.1.0)
#   N     = commits since that tag
#   patch = tagPatch*100 + N            (v0.1.0-rc2 + 11 commits -> 0.1.11)
#   code  = MAJOR*10000 + MINOR*100 + patch
# so v0.1.1 (code 200 at N=0) overtakes the 11th commit after v0.1.0 (code 111). More than
# 100 commits between tags collide, and the publish step refuses with "not higher".
#
# Repository: $AFVI_GIT_DIR (default: the current directory); ref: $AFVI_GIT_REV (default:
# HEAD, argument wins).

_afvi_git() { git -C "${AFVI_GIT_DIR:-$PWD}" "$@"; }

# afvi_version_base [rev]: "MAJOR.MINOR.PATCH commits-since-tag", or "0.0.0 0" without tags.
afvi_version_base() {
    local rev=${1:-${AFVI_GIT_REV:-HEAD}} desc base n
    desc=$(_afvi_git describe --tags --match 'v*' "$rev" 2> /dev/null) || {
        printf '0.0.0 0\n'
        return
    }
    if [[ $desc =~ ^(.*)-([0-9]+)-g[0-9a-f]+$ ]]; then
        base=${BASH_REMATCH[1]}
        n=${BASH_REMATCH[2]}
    else
        base=$desc
        n=0
    fi
    base=${base#v}
    base=${base%%-*}
    printf '%s %s\n' "$base" "$n"
}

afvi_version_name() {
    local base n major minor patch
    read -r base n <<< "$(afvi_version_base "$@")"
    [[ $base =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]] || {
        printf '0.0.0\n'
        return
    }
    major=${BASH_REMATCH[1]}
    minor=${BASH_REMATCH[2]}
    patch=${BASH_REMATCH[3]}
    printf '%s.%s.%s\n' "$major" "$minor" "$((patch * 100 + n))"
}

afvi_version_code() {
    local name major minor patch
    name=$(afvi_version_name "$@")
    IFS=. read -r major minor patch <<< "$name"
    printf '%s\n' "$((major * 10000 + minor * 100 + patch))"
}
