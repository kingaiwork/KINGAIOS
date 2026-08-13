#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
normalize="$root/scripts/release-base-version.sh"

for v in 0.1.0 0.1.0-dev 0.1.0-beta 0.1.0-rc 0.1.0-stable 0.1.0-rc.7; do
  got=$(bash "$normalize" "$v")
  [[ "$got" == 0.1.0 ]] || { echo "$v normalized to $got" >&2; exit 1; }
done

for bad in '' latest 0.1 0.1.0-alpha 0.1.0-dev.foo; do
  if bash "$normalize" "$bad" >/dev/null 2>&1; then
    echo "invalid VERSION was accepted: ${bad:-<empty>}" >&2
    exit 1
  fi
done

echo 'KINGAI release version normalization tests passed.'
