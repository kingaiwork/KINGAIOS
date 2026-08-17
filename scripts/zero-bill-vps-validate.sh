#!/usr/bin/env bash
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
mode="${1:-core}"
root="${KINGAI_VPS_ROOT:-/srv/kingai}"
sha="${GITHUB_SHA:-$(git rev-parse HEAD)}"
out="$root/qa/KINGAIOS/$sha/$mode"
mkdir -p "$out"

fail_paid() { echo "ZERO-BILL BLOCK: $*" >&2; exit 2; }

# No paid-provider credentials may be present in automatic execution.
for name in OPENAI_API_KEY ANTHROPIC_API_KEY DEEPSEEK_API_KEY GEMINI_API_KEY CLOUDFLARE_API_TOKEN; do
  [[ -z "${!name:-}" ]] || fail_paid "$name is forbidden in automatic KINGAI OS validation"
done

# The only workflow file allowed is this repository's self-hosted zero-bill workflow.
if find .github/workflows -maxdepth 1 -type f ! -name 'zero-bill-vps.yml' -print -quit | grep -q .; then
  fail_paid 'legacy GitHub Actions workflows still exist'
fi
if grep -RInE 'runs-on:[[:space:]]*(ubuntu|windows|macos)|actions/upload-artifact|github/codeql-action|gh release (create|upload)|docker/build-push-action' .github/workflows; then
  fail_paid 'billable or externally stored CI path detected'
fi

core_checks() {
  go version
  python3 --version
  docker --version || true
  mkdir -p dist
  go build -trimpath -o dist/kingai ./cmd/kingai
  go build -trimpath -o dist/kingaid ./cmd/kingaid
  go build -trimpath -o dist/kingai-update ./cmd/kingai-update
  go build -trimpath -o dist/kingai-installer ./cmd/kingai-installer
  go build -trimpath -o dist/kingai-recovery ./cmd/kingai-recovery
  go test ./...
  go vet ./...
  test -z "$(gofmt -l .)"
  for f in scripts/*.sh desktop/welcome/*.sh; do bash -n "$f"; done
  python3 - <<'PY'
import json
from pathlib import Path
for p in ['configs/policy.json','configs/agents.json','configs/models.json','configs/system.json','release/gates.json']:
    json.loads(Path(p).read_text())
PY
  bash scripts/test-release-gate-freshness.sh
  bash scripts/cross-check-security.sh
  if command -v govulncheck >/dev/null 2>&1; then govulncheck ./...; fi
  ./dist/kingai version
  ./dist/kingai-installer version
  cp -a dist "$out/"
  (cd "$out/dist" && sha256sum * > SHA256SUMS)
}

case "$mode" in
  core)
    core_checks
    ;;
  images)
    core_checks
    sudo -n true >/dev/null 2>&1 || { echo 'VPS sudo without interactive prompt is required for image builders.' >&2; exit 3; }
    bash scripts/build-rootfs.sh
    bash scripts/build-live-iso.sh
    bash scripts/build-iot-image.sh
    bash scripts/build-sentinel-rootfs.sh
    find . -maxdepth 4 -type f \( -name '*.iso' -o -name '*.img' -o -name '*.tar*' -o -name '*.qcow2' -o -name 'SHA256SUMS*' \) -print0 | xargs -0 -r cp --parents -t "$out"
    ;;
  vm)
    core_checks
    command -v qemu-system-x86_64 >/dev/null 2>&1 || { echo 'qemu-system-x86_64 is required on VPS.' >&2; exit 3; }
    [[ -e /dev/kvm && -r /dev/kvm && -w /dev/kvm ]] || { echo '/dev/kvm is not usable by the VPS runner account.' >&2; exit 3; }
    cat > "$out/README.txt" <<'EOF'
KVM readiness passed. Historical VM smoke workflows were intentionally retired to remove hosted-runner dependence.
OpenClaw/Codex must migrate each historical smoke workflow into local scripts before release gating is declared parity-complete.
EOF
    ;;
  release)
    core_checks
    bash scripts/check-release-gate-freshness.sh
    python3 scripts/generate-sbom.py || true
    mkdir -p "$root/releases/KINGAIOS/${GITHUB_REF_NAME:-manual}"
    cp -a "$out/." "$root/releases/KINGAIOS/${GITHUB_REF_NAME:-manual}/"
    echo 'Local release candidate only. No GitHub Release, registry push, artifact upload, or paid storage publication is allowed.'
    ;;
  *) echo 'usage: zero-bill-vps-validate.sh core|images|vm|release' >&2; exit 2;;
esac
