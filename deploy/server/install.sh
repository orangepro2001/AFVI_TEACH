#!/usr/bin/env bash
# install.sh - install or update the build server driver from this checkout.
#
#   deploy/server/install.sh [--host AI] [--home DIR] [--no-enable]
#
#   --host HOST    ssh host name (default AI, from ~/.ssh/config)
#   --home DIR     absolute CI folder on the server (default ~/ci/afvi-teach)
#   --no-enable    install everything but do not start the timer
#
# Safe to run again: it replaces bin/ci, README.txt and the two systemd units and never touches
# etc/ci.conf, the token, builds or caches.

set -euo pipefail

# Always run from this folder
cd "$(dirname "${BASH_SOURCE[0]}")"

HOST=AI
CI_HOME=""
ENABLE=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --host)
            HOST="$2"
            shift 2
            ;;
        --home)
            CI_HOME="$2"
            shift 2
            ;;
        --no-enable)
            ENABLE=false
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

for tool in ssh rsync; do
    if ! command -v "$tool" &> /dev/null; then
        echo "ERROR: $tool not found."
        exit 1
    fi
done

if [ -z "$CI_HOME" ]; then
    CI_HOME="$(ssh "$HOST" 'printf %s "$HOME"')/ci/afvi-teach"
fi
case $CI_HOME in
    /*) ;;
    *)
        echo "ERROR: --home must be an absolute path on the server."
        exit 1
        ;;
esac

# --- 1. what the server needs ------------------------------------------------
echo "==> preflight on $HOST"
ssh "$HOST" bash -s << 'EOF'
missing=""
for c in git go jq curl flock rsync sha256sum timeout column x86_64-w64-mingw32-gcc x86_64-w64-mingw32-windres; do
    command -v "$c" > /dev/null || missing="$missing $c"
done
if [ -n "$missing" ]; then
    echo "ERROR: missing on the server:$missing"
    exit 1
fi
systemctl --user show-environment > /dev/null || { echo "ERROR: no systemd user session"; exit 1; }
loginctl show-user "$USER" -p Linger | grep -q yes || echo "WARNING: lingering is off, the timer stops when nobody is logged in (sudo loginctl enable-linger $USER)"
go version
x86_64-w64-mingw32-gcc --version | head -n 1
EOF

# --- 2. folders and files ----------------------------------------------------
echo "==> $HOST:$CI_HOME"
ssh "$HOST" "mkdir -p '$CI_HOME'/bin '$CI_HOME'/etc '$CI_HOME'/out/main '$CI_HOME'/out/tags '$CI_HOME'/work '$CI_HOME'/state '$CI_HOME'/cache/gomod '$CI_HOME'/cache/gobuild \"\$HOME/.config/systemd/user\""
rsync -a ci "$HOST:$CI_HOME/bin/ci"
rsync -a README.txt "$HOST:$CI_HOME/README.txt"
rsync -a --ignore-existing ci.conf.example "$HOST:$CI_HOME/etc/ci.conf"

# --- 3. systemd user units ---------------------------------------------------
echo "==> systemd user units"
for unit in afvi-ci.service afvi-ci.timer; do
    sed "s#@CI_HOME@#$CI_HOME#g" "$unit" | ssh "$HOST" "cat > \"\$HOME/.config/systemd/user/$unit\""
done
ssh "$HOST" "systemctl --user daemon-reload && AFVI_CI_HOME='$CI_HOME' '$CI_HOME/bin/ci' baseline"

if [ "$ENABLE" = true ]; then
    ssh "$HOST" "systemctl --user enable --now afvi-ci.timer"
fi

echo "OK"
echo "Next:"
echo "  ssh $HOST $CI_HOME/bin/ci status"
echo "  ssh -t $HOST $CI_HOME/bin/ci set-token     # lets the server publish Releases"
