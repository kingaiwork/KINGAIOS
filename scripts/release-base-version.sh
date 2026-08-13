#!/usr/bin/env bash
set -euo pipefail

raw="${1:-}"
[[ -n "$raw" ]] || { echo 'usage: release-base-version.sh <VERSION>' >&2; exit 2; }

# VERSION records the source channel (for example 0.1.0-dev), while release
# workflows may promote the same audited source to beta/rc/stable. Remove only
# a recognized terminal channel suffix; never silently rewrite arbitrary text.
base=$(printf '%s\n' "$raw" | sed -E 's/-(dev|beta|rc|stable)(\.[0-9]+)?$//')
if [[ ! "$base" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "VERSION does not normalize to an x.y.z release base: $raw" >&2
  exit 1
fi
printf '%s\n' "$base"
