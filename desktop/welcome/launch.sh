#!/usr/bin/env bash
set -euo pipefail

mode="first-run"
case "${1:-}" in
  --settings|--force) mode="settings" ;;
  "") ;;
  *) echo "usage: kingai-welcome [--settings]" >&2; exit 2 ;;
esac

current=$(/usr/bin/kingai desktop show 2>/dev/null || true)
if [[ "$mode" == "first-run" && -n "$current" && "$current" != "unselected" ]]; then
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

qml_args=(/usr/share/kingai/desktop/welcome/Main.qml --)
if [[ "$mode" == "settings" ]]; then
  qml_args+=(--settings)
  case "$current" in
    kingai-intelligence|kingai-flow|kingai-classic) qml_args+=(--current "$current") ;;
  esac
fi
"$qml_bin" "${qml_args[@]}"

[[ -f "$choice_file" ]] || exit 0
selected=$(sed -n 's/^[[:space:]]*experience[[:space:]]*=[[:space:]]*//p' "$choice_file" | tail -n1 | tr -d '\r"' | tr -d "'")
case "$selected" in
  kingai-intelligence|kingai-flow|kingai-classic) ;;
  "") exit 0 ;;
  *) echo "KINGAI Desktop selector returned an invalid experience" >&2; exit 2 ;;
esac

if [[ "$selected" == "$current" ]]; then
  rm -f "$choice_file"
  exit 0
fi

# `desktop set` applies trusted assets first and only persists the selection
# after a successful theme/layout transition. Failed application remains
# recoverable and the previous persisted selection remains authoritative.
/usr/bin/kingai desktop set "$selected"
rm -f "$choice_file"
