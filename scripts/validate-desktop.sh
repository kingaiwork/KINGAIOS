#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-.}"
cd "$ROOT"

[[ -f profiles/desktop.yaml ]] || { echo "missing profiles/desktop.yaml" >&2; exit 1; }
[[ ! -e profiles/pc.yaml ]] || { echo "profiles/pc.yaml must not exist; PC is KINGAI OS Desktop" >&2; exit 1; }
[[ ! -e profiles/PC.yaml ]] || { echo "profiles/PC.yaml must not exist; PC is KINGAI OS Desktop" >&2; exit 1; }
[[ ! -d desktop/profiles ]] || { echo "desktop/profiles is legacy terminology; use desktop/experiences" >&2; exit 1; }
[[ -d desktop/experiences ]] || { echo "missing desktop/experiences" >&2; exit 1; }

for path in \
  desktop/intelligence/Main.qml \
  desktop/intelligence/TaskCenter.qml \
  desktop/intelligence/launch.sh \
  internal/desktopbridge/snapshot.go \
  internal/desktopbridge/snapshot_test.go \
  internal/memory/summary.go \
  internal/memory/summary_test.go \
  cmd/kingaid/desktop_private.go \
  cmd/kingaid/desktop_private_test.go \
  cmd/kingai-desktop-bridge/main.go \
  cmd/kingai-desktop-bridge/main_test.go \
  distro/overlay/usr/lib/systemd/user/kingai-desktop-bridge.service \
  distro/overlay/etc/xdg/autostart/kingai-desktop-bridge.desktop \
  distro/overlay/usr/share/applications/kingai-intelligence.desktop \
  distro/overlay/usr/share/applications/kingai-desktop-experience.desktop \
  distro/overlay/etc/xdg/mimeapps.list; do
  [[ -f "$path" ]] || { echo "missing Desktop asset: $path" >&2; exit 1; }
done

bash -n desktop/intelligence/launch.sh
bash -n desktop/welcome/launch.sh

grep -q '^MimeType=x-scheme-handler/kingai;' distro/overlay/usr/share/applications/kingai-intelligence.desktop
grep -q '^x-scheme-handler/kingai=kingai-intelligence.desktop$' distro/overlay/etc/xdg/mimeapps.list
grep -q '^Exec=/usr/lib/kingai/kingai-welcome --settings$' distro/overlay/usr/share/applications/kingai-desktop-experience.desktop
grep -q '^ExecStart=/usr/lib/kingai/kingai-desktop-bridge$' distro/overlay/usr/lib/systemd/user/kingai-desktop-bridge.service
grep -q '^UMask=0077$' distro/overlay/usr/lib/systemd/user/kingai-desktop-bridge.service
grep -q '^RestrictAddressFamilies=AF_UNIX$' distro/overlay/usr/lib/systemd/user/kingai-desktop-bridge.service
grep -q '^Exec=systemctl --user start kingai-desktop-bridge.service$' distro/overlay/etc/xdg/autostart/kingai-desktop-bridge.desktop

python3 - <<'PY'
import json
from pathlib import Path

expected = {
    "intelligence.json": {
        "id": "kingai-intelligence",
        "name": "KINGAI Intelligence",
        "theme": "org.kingai.intelligence",
        "layout": "kingai-intelligence.js",
        "mode": "ai-first",
        "default": True,
    },
    "flow.json": {
        "id": "kingai-flow",
        "name": "KINGAI Flow",
        "theme": "org.kingai.flow",
        "layout": "kingai-flow.js",
        "mode": "dock-spatial",
        "default": False,
    },
    "classic.json": {
        "id": "kingai-classic",
        "name": "KINGAI Classic",
        "theme": "org.kingai.classic",
        "layout": "kingai-classic.js",
        "mode": "taskbar-menu",
        "default": False,
    },
}

root = Path("desktop/experiences")
actual = {p.name for p in root.glob("*.json")}
if actual != set(expected):
    raise SystemExit(f"desktop experiences mismatch: expected {sorted(expected)}, got {sorted(actual)}")

defaults = []
for filename, contract in expected.items():
    path = root / filename
    data = json.loads(path.read_text())
    if data.get("schema") != 1:
        raise SystemExit(f"{path}: schema must be 1")
    for key, value in contract.items():
        if data.get(key) != value:
            raise SystemExit(f"{path}: {key} must be {value!r}, got {data.get(key)!r}")
    surfaces = data.get("primary_surfaces")
    if not isinstance(surfaces, list) or not surfaces or not all(isinstance(v, str) and v for v in surfaces):
        raise SystemExit(f"{path}: primary_surfaces must be a non-empty string list")
    if data.get("default"):
        defaults.append(data["id"])

    theme_manifest = Path("desktop/look-and-feel") / data["theme"] / "manifest.json"
    if not theme_manifest.is_file():
        raise SystemExit(f"{path}: missing theme manifest {theme_manifest}")
    layout = Path("desktop/layouts") / data["layout"]
    if not layout.is_file():
        raise SystemExit(f"{path}: missing layout {layout}")

if defaults != ["kingai-intelligence"]:
    raise SystemExit(f"KINGAI Intelligence must be the single default desktop experience, got {defaults}")

profile = Path("profiles/desktop.yaml").read_text()
for required in ("profile: desktop", "edition_role: personal-computer", "kingai-intelligence", "kingai-flow", "kingai-classic"):
    if required not in profile:
        raise SystemExit(f"profiles/desktop.yaml missing required contract token: {required}")

shell = Path("desktop/intelligence/Main.qml").read_text()
for center in ("home", "agents", "tasks", "approvals", "memory", "models", "automations", "health"):
    if f'id: "{center}"' not in shell:
        raise SystemExit(f"KINGAI Intelligence shell missing center: {center}")
for token in (
    'file:///run/kingai/public-status.json',
    'Application.arguments',
    'TaskCenter {',
    'running_tasks',
    'waiting_approval_tasks',
    'blocked_tasks',
):
    if token not in shell:
        raise SystemExit(f"KINGAI Intelligence shell missing required token: {token}")
for forbidden in (
    "/v1/memory/list",
    "/v1/approval/list",
    "/v1/tasks/list",
    "api_key=",
    "password=",
    "secret=",
    "authorization: bearer",
):
    if forbidden.lower() in shell.lower():
        raise SystemExit(f"KINGAI Intelligence public shell contains forbidden direct/sensitive pattern: {forbidden}")

task_center = Path("desktop/intelligence/TaskCenter.qml").read_text()
for token in (
    'StandardPaths.RuntimeLocation',
    '/kingai/desktop-private.json',
    'snapshot.schema !== 1',
    'ageMs > 15000',
):
    if token not in task_center:
        raise SystemExit(f"Task Center missing private-bridge safety token: {token}")
for forbidden in (
    '/var/lib/kingai/tasks',
    '/v1/tasks/list',
    '/v1/approval/list',
    '/v1/memory/list',
):
    if forbidden in task_center:
        raise SystemExit(f"Task Center bypasses the private bridge: {forbidden}")

bridge = Path("cmd/kingai-desktop-bridge/main.go").read_text()
for token in (
    '"/v1/desktop/private"',
    'desktopbridge.Snapshot',
    'snapshot.UserUID != uint32(os.Geteuid())',
    '0o600',
    '0o700',
    'XDG_RUNTIME_DIR',
    'refuses to run as root',
):
    if token not in bridge:
        raise SystemExit(f"Desktop Bridge missing security contract token: {token}")
for forbidden in (
    '"/v1/tasks/list"',
    '"/v1/memory/list"',
    'taskgraph.Task',
    'memory.Record',
    'step.Target',
    'step.Capability',
    'step.ApprovalID',
    'step.Result',
    'step.Error',
):
    if forbidden in bridge:
        raise SystemExit(f"Desktop Bridge must consume only the server-redacted snapshot: {forbidden}")

daemon = Path('cmd/kingaid/main.go').read_text()
if 'registerDesktopPrivateHandler(mux, taskStore, memoryStore, version)' not in daemon:
    raise SystemExit('kingaid must register the Desktop private handler on its peer-credential Unix-socket mux')
handler = Path('cmd/kingaid/desktop_private.go').read_text()
for token in (
    '"/v1/desktop/private"',
    'peerUID(r.Context())',
    'taskStore.ListForPeer(uid, 100)',
    'memoryStore.Summarize(ownerForUID(uid))',
    'desktopbridge.Build',
):
    if token not in handler:
        raise SystemExit(f"kingaid Desktop private handler missing boundary token: {token}")
for forbidden in ('/v1/approval/list', 'memoryStore.List(', 'taskStore.List('):
    if forbidden in handler:
        raise SystemExit(f"kingaid Desktop private handler bypasses scoped summary API: {forbidden}")

snapshot = Path('internal/desktopbridge/snapshot.go').read_text()
for token in ('TaskSummary', 'memory.Summary', 'truncate(strings.TrimSpace(task.Goal), 512)'):
    if token not in snapshot:
        raise SystemExit(f"Desktop snapshot contract missing redaction token: {token}")
for forbidden in ('Target ', 'Capability ', 'ApprovalID ', 'Result ', 'Error '):
    if forbidden in snapshot:
        raise SystemExit(f"Desktop snapshot schema contains forbidden raw task field: {forbidden.strip()}")

memory_summary = Path('internal/memory/summary.go').read_text()
if 'Data ' in memory_summary or 'json.RawMessage' in memory_summary:
    raise SystemExit('Memory summary implementation must not model record Data')
for token in ('ByLayer', 'BySensitivity', 'Expiring', 'record.ExpiresAt'):
    if token not in memory_summary:
        raise SystemExit(f"Memory summary contract missing token: {token}")

layouts = {
    'kingai-intelligence.js': 'kingai-intelligence',
    'kingai-flow.js': 'kingai-flow',
    'kingai-classic.js': 'kingai-classic',
}
for filename, experience in layouts.items():
    text = (Path('desktop/layouts') / filename).read_text()
    for token in ('kingaiManaged', 'clearWidgets', 'widget.remove()', experience):
        if token not in text:
            raise SystemExit(f"{filename}: managed layout contract missing {token}")

launch = Path("desktop/intelligence/launch.sh").read_text()
if 'Main.qml -- --center' not in launch:
    raise SystemExit("KINGAI Intelligence launcher must separate qml runtime options from app arguments")

launcher = Path("distro/overlay/usr/share/applications/kingai-intelligence.desktop").read_text()
for action in ("Agents", "Tasks", "Approvals", "Memory", "Models", "Automations", "Health"):
    if f"[Desktop Action {action}]" not in launcher:
        raise SystemExit(f"KINGAI Intelligence launcher missing desktop action: {action}")
PY

echo "KINGAI OS Desktop contract: OK"
