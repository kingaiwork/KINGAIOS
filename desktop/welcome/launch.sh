#!/usr/bin/env bash
set -euo pipefail

current=$(/usr/bin/kingai desktop show 2>/dev/null || true)
if [[ -n "$current" && "$current" != "unselected" ]]; then
  exit 0
fi

qml_bin=""
for candidate in /usr/lib/qt6/bin/qml qml6 qml; do
  if [[ "$candidate" == /* && -x "$candidate" ]]; then qml_bin="$candidate"; break; fi
  if command -v "$candidate" >/dev/null 2>&1; then qml_bin=$(command -v "$candidate"); break; fi
done
[[ -n "$qml_bin" ]] || { echo "Qt 6 QML viewer is not available" >&2; exit 1; }

"$qml_bin" /usr/share/kingai/desktop/welcome/Main.qml
selected=$(/usr/bin/kingai desktop show)
[[ "$selected" != "unselected" ]] || exit 0
/usr/bin/kingai desktop apply
