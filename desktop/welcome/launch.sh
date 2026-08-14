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

cache_root="${XDG_CACHE_HOME:-${HOME:?HOME is required}/.cache}"
choice_file="$cache_root/kingai-welcome.ini"
mkdir -p "$cache_root"
rm -f "$choice_file"

"$qml_bin" /usr/share/kingai/desktop/welcome/Main.qml
[[ -f "$choice_file" ]] || exit 0

selected=$(sed -n 's/^[[:space:]]*experience[[:space:]]*=[[:space:]]*//p' "$choice_file" | tail -n1 | tr -d '\r"' | tr -d "'")
case "$selected" in
  kingai-intelligence|kingai-flow|kingai-classic) ;;
  "") exit 0 ;;
  *) echo "KINGAI Welcome returned an invalid desktop experience" >&2; exit 2 ;;
esac

# `desktop set` applies trusted assets first and only persists the selection
# after a successful theme/layout transition. Failed application remains
# recoverable: the next login will show KINGAI Welcome again.
/usr/bin/kingai desktop set "$selected"
rm -f "$choice_file"
