#!/usr/bin/env bash
set -euo pipefail

validate_release() {
  local version=$1 channel=$2
  case "$channel" in
    beta) [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+-beta\.[0-9]+$ ]] ;;
    rc) [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$ ]] ;;
    stable) [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ;;
    *) return 1 ;;
  esac
}

for pair in \
  '0.1.0-beta.1 beta' \
  '1.2.3-beta.42 beta' \
  '0.1.0-rc.1 rc' \
  '9.8.7 stable'
do
  read -r version channel <<<"$pair"
  validate_release "$version" "$channel" || {
    echo "expected valid release identity: $version / $channel" >&2
    exit 1
  }
done

for pair in \
  '0.1.0 beta' \
  '0.1.0-beta.1 stable' \
  '0.1.0-rc.1 beta' \
  '0.1.0-rc rc' \
  'v0.1.0 stable' \
  '0.1 stable' \
  '0.1.0 nightly'
do
  read -r version channel <<<"$pair"
  if validate_release "$version" "$channel"; then
    echo "expected invalid release identity: $version / $channel" >&2
    exit 1
  fi
done

python3 - <<'PY'
import json
from pathlib import Path

gates = json.loads(Path('release/gates.json').read_text(encoding='utf-8'))
assert gates['container_release_pipeline'] is True
assert gates['container_signed_release'] is False
assert gates['production_signing_ready'] is False

request = json.loads(Path('.release/container-beta.json').read_text(encoding='utf-8'))
assert request['profile'] == 'container'
assert request['channel'] == 'beta'
assert request['version'] == '0.1.0-beta.1'
PY

echo "KINGAI OS Container release policy regression tests passed."
