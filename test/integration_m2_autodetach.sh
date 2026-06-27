#!/usr/bin/env bash
# Verifies auto-detach-on-user-activity against REAL tmux: when a human attaches
# to the session, the supervisor steps aside (detaches) and leaves it running.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/agentkeeper-linux"
export AGENTKEEPER_STATE_DIR="$(mktemp -d)"
N="ak-auto-$$"
fail=0

cleanup() { tmux kill-session -t "$N" 2>/dev/null; rm -rf "$AGENTKEEPER_STATE_DIR"; }
trap cleanup EXIT
check() { if eval "$2"; then echo "  ok: $1"; else echo "  FAIL: $1"; fail=1; fi; }

"$BIN" run --name "$N" -- "sleep 600" >/tmp/ak_auto.log 2>&1 &
SUP=$!
sleep 3

echo "== simulate a human attaching (tmux attach in a pty) =="
script -qfc "tmux attach -t $N" /dev/null >/dev/null 2>&1 &
ATT=$!
sleep 5

check "supervisor auto-detached" '! kill -0 "$SUP" 2>/dev/null'
check "session still alive (handed to user)" 'tmux has-session -t "$N" 2>/dev/null'
check "status shows DETACHED" '"$BIN" status --name "$N" | grep -q DETACHED'
check "log mentions auto-detach" 'grep -qi "auto-detach" /tmp/ak_auto.log'

kill "$ATT" 2>/dev/null
echo "---- supervisor log ----"; cat /tmp/ak_auto.log
if [ "$fail" -eq 0 ]; then echo "RESULT: PASS"; else echo "RESULT: FAIL"; fi
exit "$fail"
