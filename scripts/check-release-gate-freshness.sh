#!/usr/bin/env bash
set -euo pipefail

channel="${1:-}"
profile="${2:-server}"
case "$channel" in dev|beta|rc|stable) ;; *) echo "usage: $0 <dev|beta|rc|stable> <server|desktop>" >&2; exit 2;; esac
case "$profile" in server|desktop) ;; *) echo "invalid profile: $profile" >&2; exit 2;; esac

[[ "$channel" == dev ]] && { echo 'Developer channel: freshness gates are advisory only.'; exit 0; }
: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
: "${GITHUB_SHA:?GITHUB_SHA is required}"
command -v gh >/dev/null 2>&1 || { echo 'gh CLI is required for release evidence verification' >&2; exit 1; }
command -v git >/dev/null 2>&1 || { echo 'git is required for release evidence verification' >&2; exit 1; }

latest_success_sha() {
  local workflow=$1 sha
  sha=$(gh api -H 'Accept: application/vnd.github+json' \
    "repos/${GITHUB_REPOSITORY}/actions/workflows/${workflow}/runs?branch=main&status=success&per_page=1" \
    --jq '.workflow_runs[0].head_sha // empty')
  [[ -n "$sha" ]] || { echo "No successful evidence run exists for ${workflow}" >&2; return 1; }
  printf '%s\n' "$sha"
}

require_fresh() {
  local gate=$1 workflow=$2 regex=$3 sha changed
  sha=$(latest_success_sha "$workflow")
  git cat-file -e "${sha}^{commit}" 2>/dev/null || git fetch --no-tags origin main >/dev/null 2>&1 || true
  git cat-file -e "${sha}^{commit}" 2>/dev/null || { echo "${gate}: evidence commit ${sha} is not available locally" >&2; return 1; }
  git merge-base --is-ancestor "$sha" "$GITHUB_SHA" || { echo "${gate}: evidence ${sha} is not an ancestor of ${GITHUB_SHA}" >&2; return 1; }
  changed=$(git diff --name-only "${sha}..${GITHUB_SHA}" -- | grep -E "$regex" || true)
  if [[ -n "$changed" ]]; then
    echo "${gate}: stale verification. Relevant files changed after successful ${workflow} run ${sha}:" >&2
    printf '%s\n' "$changed" >&2
    return 1
  fi
  echo "${gate}: fresh (${workflow} @ ${sha})"
}

# Every non-dev release first requires a fresh cross-component regression pass.
# Deliberately exclude .release request markers so a release request can be the
# commit after the evidence without invalidating it. Any actual code, policy,
# service, build/release script, workflow or module change makes the evidence stale.
require_fresh stability-security 'stability-security-crosscheck.yml' \
  '^(cmd/|internal/|configs/|container/|systemd/|scripts/|release/|\.github/workflows/|go\.(mod|sum)$)'

if [[ "$profile" == desktop ]]; then
  require_fresh installer 'smoke-installer-desktop-vm.yml' \
    '^(cmd/kingai-installer/|internal/installer/|internal/update/|desktop/|distro/packages/(desktop|installer-[^/]+)\.txt$|distro/overlay/|scripts/build-rootfs\.sh$|systemd/kingai-update-health\.service$|go\.(mod|sum)$)'
else
  require_fresh installer 'smoke-installer-vm.yml' \
    '^(cmd/kingai-installer/|internal/installer/|internal/update/|distro/packages/(server|installer-[^/]+)\.txt$|distro/overlay/|scripts/build-rootfs\.sh$|systemd/kingai-update-health\.service$|go\.(mod|sum)$)'
  require_fresh installable-live-iso 'smoke-installable-live-iso.yml' \
    '^(cmd/kingai-installer/|internal/installer/|scripts/(kingai-install-live|build-rootfs|build-live-iso)\.sh$|distro/packages/(server|installer-[^/]+)\.txt$|distro/overlay/|systemd/|go\.(mod|sum)$)'
fi

require_fresh ab-update 'smoke-update-ab-vm.yml' \
  '^(cmd/kingai-update/|cmd/kingai-installer/|internal/update/|internal/installer/|systemd/kingai-update-health\.service$|scripts/build-rootfs\.sh$|distro/packages/(server|installer-[^/]+)\.txt$|distro/overlay/|go\.(mod|sum)$)'
require_fresh recovery 'smoke-recovery-vm.yml' \
  '^(cmd/kingai-recovery/|internal/recovery/|internal/update/|internal/installer/|distro/packages/recovery\.txt$|distro/overlay/|scripts/build-(rootfs|live-iso)\.sh$|go\.(mod|sum)$)'
require_fresh tuf-client 'ci.yml' \
  '^(internal/tufclient/|cmd/kingai-update/|go\.(mod|sum)$)'

if [[ "$channel" == rc || "$channel" == stable ]]; then
  require_fresh secure-boot 'smoke-secure-boot-vm.yml' \
    '^(internal/installer/|distro/packages/installer-amd64\.txt$|scripts/build-rootfs\.sh$|go\.(mod|sum)$)'
fi

echo "Release evidence freshness passed for ${profile}/${channel}."
