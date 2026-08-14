#!/usr/bin/env bash
set -euo pipefail

qml_bin=""
for candidate in /usr/lib/qt6/bin/qml qml6 qml; do
  if [[ "$candidate" == /* && -x "$candidate" ]]; then
    qml_bin="$candidate"
    break
  fi
  if command -v "$candidate" >/dev/null 2>&1; then
    qml_bin=$(command -v "$candidate")
    break
  fi
done

[[ -n "$qml_bin" ]] || {
  echo "Qt 6 QML viewer is not available" >&2
  exit 1
}

center="home"
if [[ $# -ge 1 ]]; then
  case "$1" in
    kingai://*) center="${1#kingai://}"; center="${center%%/*}" ;;
    --center)
      [[ $# -ge 2 ]] || { echo "--center requires a value" >&2; exit 2; }
      center="$2"
      ;;
    *) center="$1" ;;
  esac
fi

case "$center" in
  home|agents|tasks|approvals|memory|models|automations|health) ;;
  *) center="home" ;;
esac

exec "$qml_bin" /usr/share/kingai/desktop/intelligence/Main.qml --center "$center"
