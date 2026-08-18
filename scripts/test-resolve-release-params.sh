#!/usr/bin/env bash
set -euo pipefail

resolver="scripts/resolve-release-params.sh"
[[ -x "$resolver" || -f "$resolver" ]] || { echo "missing $resolver" >&2; exit 1; }

expect() {
  local changed="$1" want_profile="$2" want_channel="$3" out
  out=$(KINGAI_RELEASE_CHANGED_FILES="$changed" bash "$resolver" push '' '' deadbeef cafebabe)
  grep -qx "profile=$want_profile" <<<"$out"
  grep -qx "channel=$want_channel" <<<"$out"
}

expect '.release/desktop-dev.json' desktop dev
expect $'scripts/foo.sh\n.release/desktop-beta.json\nREADME.md' desktop beta
expect '.release/server-dev.json' server dev
expect '.release/server-beta.json' server beta

manual=$(bash "$resolver" workflow_dispatch desktop rc '' '')
grep -qx 'profile=desktop' <<<"$manual"
grep -qx 'channel=rc' <<<"$manual"

if KINGAI_RELEASE_CHANGED_FILES='README.md' bash "$resolver" push '' '' deadbeef cafebabe >/dev/null 2>&1; then
  echo 'resolver accepted a push without an explicit release request' >&2
  exit 1
fi
if KINGAI_RELEASE_CHANGED_FILES=$'.release/desktop-dev.json\n.release/server-dev.json' bash "$resolver" push '' '' deadbeef cafebabe >/dev/null 2>&1; then
  echo 'resolver accepted ambiguous release requests' >&2
  exit 1
fi
if bash "$resolver" workflow_dispatch desktop invalid '' '' >/dev/null 2>&1; then
  echo 'resolver accepted an invalid manual channel' >&2
  exit 1
fi

echo 'release parameter resolver tests passed'
