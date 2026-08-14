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
  desktop/intelligence/launch.sh \
  distro/overlay/usr/share/applications/kingai-intelligence.desktop \
  distro/overlay/etc/xdg/mimeapps.list; do
  [[ -f "$path" ]] || { echo "missing Desktop Phase 2 asset: $path" >&2; exit 1; }
done

bash -n desktop/intelligence/launch.sh

grep -q '^MimeType=x-scheme-handler/kingai;' distro/overlay/usr/share/applications/kingai-intelligence.desktop
grep -q '^x-scheme-handler/kingai=kingai-intelligence.desktop$' distro/overlay/etc/xdg/mimeapps.list

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
if 'file:///run/kingai/public-status.json' not in shell:
    raise SystemExit("KINGAI Intelligence shell must use the sanitized public runtime status channel")
if 'Application.arguments' not in shell:
    raise SystemExit("KINGAI Intelligence shell must use the Qt Application.arguments interface")
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

launch = Path("desktop/intelligence/launch.sh").read_text()
if 'Main.qml -- --center' not in launch:
    raise SystemExit("KINGAI Intelligence launcher must separate qml runtime options from app arguments")

launcher = Path("distro/overlay/usr/share/applications/kingai-intelligence.desktop").read_text()
for action in ("Agents", "Tasks", "Approvals", "Memory", "Models", "Automations", "Health"):
    if f"[Desktop Action {action}]" not in launcher:
        raise SystemExit(f"KINGAI Intelligence launcher missing desktop action: {action}")
PY

echo "KINGAI OS Desktop contract: OK"
