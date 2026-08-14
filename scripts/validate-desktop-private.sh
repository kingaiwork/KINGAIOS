#!/usr/bin/env bash
set -euo pipefail

for path in \
  desktop/intelligence/Main.qml \
  desktop/intelligence/TaskCenter.qml \
  desktop/intelligence/MemoryCenter.qml \
  internal/desktopbridge/snapshot.go \
  internal/memory/summary.go \
  cmd/kingaid/desktop_private.go \
  cmd/kingai-desktop-bridge/main.go; do
  test -f "$path" || { echo "missing private Desktop asset: $path" >&2; exit 1; }
done

python3 - <<'PY'
from pathlib import Path

main = Path('desktop/intelligence/Main.qml').read_text()
for token in ('TaskCenter {', 'MemoryCenter {', 'file:///run/kingai/public-status.json'):
    assert token in main, f'Main.qml missing {token}'

for filename in ('TaskCenter.qml', 'MemoryCenter.qml'):
    text = (Path('desktop/intelligence') / filename).read_text()
    for token in ('StandardPaths.RuntimeLocation', '/kingai/desktop-private.json', 'snapshot.schema !== 1', 'ageMs > 15000'):
        assert token in text, f'{filename} missing {token}'
    for forbidden in ('/v1/tasks/list', '/v1/memory/list', '/v1/approval/list', '/var/lib/kingai'):
        assert forbidden not in text, f'{filename} bypasses bridge with {forbidden}'

memory = Path('desktop/intelligence/MemoryCenter.qml').read_text()
for layer in ('M0','M1','M2','M3','M4','M5','M6'):
    assert f'id: "{layer}"' in memory, f'MemoryCenter missing {layer}'
for token in ('snapshot.memory.total', 'snapshot.memory.by_layer', 'snapshot.memory.by_sensitivity'):
    assert token in memory, f'MemoryCenter missing {token}'

bridge = Path('cmd/kingai-desktop-bridge/main.go').read_text()
assert '"/v1/desktop/private"' in bridge
for forbidden in ('"/v1/tasks/list"', '"/v1/memory/list"', 'taskgraph.Task', 'memory.Record'):
    assert forbidden not in bridge, f'bridge regressed to raw API: {forbidden}'

handler = Path('cmd/kingaid/desktop_private.go').read_text()
for token in ('taskStore.ListForPeer(uid, 100)', 'memoryStore.Summarize(ownerForUID(uid))', 'desktopbridge.Build'):
    assert token in handler, f'kingaid private endpoint missing {token}'

snapshot = Path('internal/desktopbridge/snapshot.go').read_text()
for forbidden in ('Target string', 'Capability string', 'ApprovalID string', 'Result json.RawMessage', 'Error string'):
    assert forbidden not in snapshot, f'private snapshot exposes {forbidden}'

summary = Path('internal/memory/summary.go').read_text()
assert '`json:"data"`' not in summary
assert 'json.RawMessage' not in summary
PY

echo "KINGAI OS Desktop private data contract: OK"
