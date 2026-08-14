#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
checker="$repo_root/scripts/check-release-gate-freshness.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bin" "$tmp/repo"

cat > "$tmp/bin/gh" <<'EOF'
#!/bin/sh
set -eu
: "${FAKE_EVIDENCE_SHA:?}"
printf '%s\n' "$FAKE_EVIDENCE_SHA"
EOF
chmod 0755 "$tmp/bin/gh"

cd "$tmp/repo"
git init -q
git config user.name KINGAI-CI
git config user.email ci@kingai.invalid
mkdir -p internal/update
echo base > README.md
echo 'package update' > internal/update/base.go
git add .
git commit -qm base
evidence=$(git rev-parse HEAD)

# Unrelated documentation changes must not invalidate binary/update gates.
echo docs >> README.md
git add README.md
git commit -qm docs
current=$(git rev-parse HEAD)
PATH="$tmp/bin:$PATH" FAKE_EVIDENCE_SHA="$evidence" GITHUB_REPOSITORY=kingaiwork/KINGAIOS GITHUB_SHA="$current" \
  bash "$checker" beta server >/dev/null
PATH="$tmp/bin:$PATH" FAKE_EVIDENCE_SHA="$evidence" GITHUB_REPOSITORY=kingaiwork/KINGAIOS GITHUB_SHA="$current" \
  bash "$checker" beta iot >/dev/null

# A relevant A/B/Edge update control-plane change must invalidate previously green evidence.
echo '// changed' >> internal/update/base.go
git add internal/update/base.go
git commit -qm update-change
current=$(git rev-parse HEAD)
if PATH="$tmp/bin:$PATH" FAKE_EVIDENCE_SHA="$evidence" GITHUB_REPOSITORY=kingaiwork/KINGAIOS GITHUB_SHA="$current" \
  bash "$checker" beta server >/dev/null 2>&1; then
  echo 'freshness checker accepted stale server update evidence' >&2
  exit 1
fi
if PATH="$tmp/bin:$PATH" FAKE_EVIDENCE_SHA="$evidence" GITHUB_REPOSITORY=kingaiwork/KINGAIOS GITHUB_SHA="$current" \
  bash "$checker" beta iot >/dev/null 2>&1; then
  echo 'freshness checker accepted stale IoT update evidence' >&2
  exit 1
fi

# Developer builds deliberately do not require release evidence.
env -u GITHUB_REPOSITORY -u GITHUB_SHA bash "$checker" dev server >/dev/null
env -u GITHUB_REPOSITORY -u GITHUB_SHA bash "$checker" dev iot >/dev/null

# Release identity must never leak the source channel suffix into promoted tags.
bash "$repo_root/scripts/test-release-base-version.sh" >/dev/null

echo 'KINGAI release control regression tests passed.'
