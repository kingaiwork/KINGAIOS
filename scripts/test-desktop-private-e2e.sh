#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

tmp=$(mktemp -d)
daemon_pid=""
cleanup() {
  if [[ -n "$daemon_pid" ]]; then
    kill "$daemon_pid" 2>/dev/null || true
    wait "$daemon_pid" 2>/dev/null || true
  fi
  rm -rf "$tmp"
}
trap cleanup EXIT

KINGAI_BIN="${KINGAI_TEST_CLI:-$tmp/kingai}"
KINGAID_BIN="${KINGAI_TEST_DAEMON:-$tmp/kingaid}"
BRIDGE_BIN="${KINGAI_TEST_DESKTOP_BRIDGE:-$tmp/kingai-desktop-bridge}"

if [[ ! -x "$KINGAI_BIN" ]]; then
  go build -trimpath -o "$KINGAI_BIN" ./cmd/kingai
fi
if [[ ! -x "$KINGAID_BIN" ]]; then
  go build -trimpath -o "$KINGAID_BIN" ./cmd/kingaid
fi
if [[ ! -x "$BRIDGE_BIN" ]]; then
  go build -trimpath -o "$BRIDGE_BIN" ./cmd/kingai-desktop-bridge
fi

export KINGAI_SOCKET="$tmp/kingaid.sock"
export KINGAI_POLICY="$ROOT/configs/policy.json"
export KINGAI_AGENTS="$ROOT/configs/agents.json"
export KINGAI_MODELS="$ROOT/configs/models.json"
export KINGAI_AUDIT="$tmp/audit.jsonl"
export KINGAI_PUBLIC_STATUS="$tmp/public-status.json"
export KINGAI_APPROVAL_ROOT="$tmp/approvals"
export KINGAI_MEMORY_ROOT="$tmp/memory"
export KINGAI_TASK_ROOT="$tmp/tasks"
export KINGAI_REQUIRE_EXECD=false
export KINGAI_TASK_RUN_BUDGET=64
export XDG_RUNTIME_DIR="$tmp/runtime"
mkdir -m 0700 "$XDG_RUNTIME_DIR"

"$KINGAID_BIN" >"$tmp/kingaid.log" 2>&1 &
daemon_pid=$!
for _ in $(seq 1 80); do
  [[ -S "$KINGAI_SOCKET" ]] && break
  sleep 0.05
done
[[ -S "$KINGAI_SOCKET" ]] || { cat "$tmp/kingaid.log" >&2; echo "kingaid socket was not created" >&2; exit 1; }

"$KINGAI_BIN" task create main "desktop-private-e2e-goal" > "$tmp/task.json"
"$KINGAI_BIN" memory put semantic '{"private_marker":"desktop-private-e2e-memory-body"}' > "$tmp/memory.json"

"$BRIDGE_BIN" --once
snapshot="$XDG_RUNTIME_DIR/kingai/desktop-private.json"
[[ -f "$snapshot" ]] || { echo "Desktop private snapshot missing" >&2; exit 1; }
[[ "$(stat -c '%a' "$XDG_RUNTIME_DIR/kingai")" == "700" ]]
[[ "$(stat -c '%a' "$snapshot")" == "600" ]]

python3 - "$snapshot" "$(id -u)" <<'PY'
import json, pathlib, sys
path = pathlib.Path(sys.argv[1])
uid = int(sys.argv[2])
data = json.loads(path.read_text())
assert data['schema'] == 1, data
assert data['product'] == 'KINGAI OS Desktop', data
assert data['user_uid'] == uid, (data.get('user_uid'), uid)
assert isinstance(data['agents'], list) and data['agents'], data
main = next((a for a in data['agents'] if a.get('id') == 'main'), None)
assert main is not None and main['authorized_for_peer'] is True, main
assert isinstance(main.get('capability_count'), int), main
assert isinstance(data['tasks'], list), data
assert any(t.get('goal') == 'desktop-private-e2e-goal' for t in data['tasks']), data['tasks']
assert data['memory']['total'] >= 1, data['memory']
assert data['memory']['by_layer'].get('M4', 0) >= 1, data['memory']
text = path.read_text().lower()
for forbidden in (
    'desktop-private-e2e-memory-body',
    'private_marker',
    'filesystem.read',
    'filesystem.write',
    'service.restart',
    'package.install',
    'approval_id',
    'target_hash',
):
    assert forbidden not in text, (forbidden, text)
PY

echo "KINGAI OS Desktop private E2E: OK"
