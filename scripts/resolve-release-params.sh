#!/usr/bin/env bash
set -euo pipefail

event_name="${1:-}"
input_profile="${2:-}"
input_channel="${3:-}"
push_before="${4:-}"
push_sha="${5:-${GITHUB_SHA:-}}"

emit() {
  local profile="$1" channel="$2"
  case "$profile" in server|desktop) ;; *) echo "invalid release profile: $profile" >&2; exit 2;; esac
  case "$channel" in dev|beta|rc|stable) ;; *) echo "invalid release channel: $channel" >&2; exit 2;; esac
  printf 'profile=%s\nchannel=%s\n' "$profile" "$channel"
}

if [[ "$event_name" != "push" ]]; then
  [[ -n "$input_profile" ]] || input_profile=server
  [[ -n "$input_channel" ]] || input_channel=dev
  emit "$input_profile" "$input_channel"
  exit 0
fi

changed="${KINGAI_RELEASE_CHANGED_FILES:-}"
if [[ -z "$changed" ]]; then
  [[ -n "$push_sha" ]] || { echo 'push SHA is required' >&2; exit 2; }
  if [[ -n "$push_before" && "$push_before" != 0000000000000000000000000000000000000000 ]] && git cat-file -e "$push_before^{commit}" 2>/dev/null; then
    changed=$(git diff --name-only "$push_before" "$push_sha")
  else
    # Merge commits require -m; a plain diff-tree on a merge can be empty and
    # must never silently route a Desktop request to Server.
    changed=$(git diff-tree -m --no-commit-id --name-only -r "$push_sha" | sort -u)
  fi
fi

mapfile -t requests < <(printf '%s\n' "$changed" | grep -E '^\.release/(server|desktop)-(dev|beta)\.json$' | sort -u || true)
if (( ${#requests[@]} == 0 )); then
  echo 'push release routing failed: no explicit release request file changed' >&2
  exit 3
fi
if (( ${#requests[@]} != 1 )); then
  printf 'push release routing failed: ambiguous release requests:' >&2
  printf ' %s' "${requests[@]}" >&2
  printf '\n' >&2
  exit 3
fi

case "${requests[0]}" in
  .release/server-dev.json) emit server dev ;;
  .release/server-beta.json) emit server beta ;;
  .release/desktop-dev.json) emit desktop dev ;;
  .release/desktop-beta.json) emit desktop beta ;;
  *) echo "unsupported release request: ${requests[0]}" >&2; exit 3 ;;
esac
