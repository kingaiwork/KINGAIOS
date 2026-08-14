#!/usr/bin/env bash
set -euo pipefail

fail() { echo "cross-check: $*" >&2; exit 1; }
need() { [[ -f "$1" ]] || fail "missing required file: $1"; }
contains() { grep -Fq -- "$2" "$1" || fail "$1 missing invariant: $2"; }
not_contains() { ! grep -Fq -- "$2" "$1" || fail "$1 contains forbidden invariant: $2"; }

for f in \
  release/gates.json \
  .github/workflows/release.yml \
  .github/workflows/stability-security-crosscheck.yml \
  .github/workflows/codeql.yml \
  scripts/check-release-gate-freshness.sh \
  scripts/build-live-iso.sh \
  systemd/kingaid.service \
  systemd/kingai-execd.service \
  container/Dockerfile; do
  need "$f"
done

python3 - <<'PY'
import json
from pathlib import Path

p = Path('release/gates.json')
g = json.loads(p.read_text())
required_bool = [
    'd5_runtime_ci','container_image_ci','container_multiarch_ci',
    'container_security_scan_ci','container_oci_archive_ci',
    'container_release_pipeline','container_signed_release','installer_vm',
    'installable_live_iso_vm','desktop_live_vm','server_bios_uefi_vm',
    'ab_update_vm','recovery_vm','tuf_client','secure_boot_vm',
    'tuf_repository','production_signing_ready','protected_branch',
    'rollback_drill','recovery_drill','r2_delivery',
]
for key in required_bool:
    if key not in g or not isinstance(g[key], bool):
        raise SystemExit(f'cross-check: gate {key!r} must exist and be boolean')

# Functional Beta evidence must remain complete once an installable Beta is published.
for key in ('installer_vm','installable_live_iso_vm','ab_update_vm','recovery_vm','tuf_client'):
    if g[key] is not True:
        raise SystemExit(f'cross-check: published installable Beta requires {key}=true')

# Never infer production readiness from test-only Secure Boot evidence.
if g['production_signing_ready'] and not g['tuf_repository']:
    raise SystemExit('cross-check: production signing cannot be ready before production TUF repository')
if g['r2_delivery'] and not g['production_signing_ready']:
    raise SystemExit('cross-check: production delivery cannot precede production signing readiness')

# CodeQL tracing must initialize only after the exact Go toolchain is installed.
codeql = Path('.github/workflows/codeql.yml').read_text()
setup = codeql.find('actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e')
init = codeql.find('github/codeql-action/init@ff2f1c621b7f889edc0d3c761ac2e6a3f8cdb0dd')
build = codeql.find('go build ./...')
analyze = codeql.find('github/codeql-action/analyze@ff2f1c621b7f889edc0d3c761ac2e6a3f8cdb0dd')
if min(setup, init, build, analyze) < 0 or not (setup < init < build < analyze):
    raise SystemExit('cross-check: CodeQL must run setup-go -> init -> traced build -> analyze')
PY

# Core daemon: non-root, no TCP/IP address families, no privilege gain.
contains systemd/kingaid.service 'User=_kingai'
contains systemd/kingaid.service 'NoNewPrivileges=yes'
contains systemd/kingaid.service 'ProtectSystem=strict'
contains systemd/kingaid.service 'ProtectHome=yes'
contains systemd/kingaid.service 'RestrictAddressFamilies=AF_UNIX'
contains systemd/kingaid.service 'CapabilityBoundingSet='
contains systemd/kingaid.service 'AmbientCapabilities='
contains systemd/kingaid.service 'UMask=0077'
not_contains systemd/kingaid.service 'User=root'
not_contains systemd/kingaid.service 'AF_INET'
not_contains systemd/kingaid.service 'AF_INET6'

# ExecD is intentionally privileged but must stay isolated and capability-empty.
contains systemd/kingai-execd.service 'User=root'
contains systemd/kingai-execd.service 'PrivateNetwork=yes'
contains systemd/kingai-execd.service 'NoNewPrivileges=yes'
contains systemd/kingai-execd.service 'ProtectSystem=strict'
contains systemd/kingai-execd.service 'ProtectProc=invisible'
contains systemd/kingai-execd.service 'RestrictNamespaces=yes'
contains systemd/kingai-execd.service 'RestrictAddressFamilies=AF_UNIX'
contains systemd/kingai-execd.service 'CapabilityBoundingSet='
contains systemd/kingai-execd.service 'AmbientCapabilities='
contains systemd/kingai-execd.service 'MemoryMax=128M'
contains systemd/kingai-execd.service 'TasksMax=32'
not_contains systemd/kingai-execd.service 'AF_INET'
not_contains systemd/kingai-execd.service 'AF_INET6'

# Container default must remain fixed non-root with Unix-socket management only.
contains container/Dockerfile 'ARG KINGAI_UID=10001'
contains container/Dockerfile 'ARG KINGAI_GID=10001'
contains container/Dockerfile 'KINGAI_SOCKET=/run/kingai/kingaid.sock'
contains container/Dockerfile 'USER _kingai:kingai'
contains container/Dockerfile 'VOLUME ["/var/lib/kingai", "/var/log/kingai"]'
not_contains container/Dockerfile 'EXPOSE '

# Release policy must keep Beta and Stable materially different.
contains .github/workflows/release.yml '.installer_vm==true and .installable_live_iso_vm==true and .ab_update_vm==true and .recovery_vm==true and .tuf_client==true'
contains .github/workflows/release.yml '.secure_boot_vm==true and .tuf_repository==true and .production_signing_ready==true'
contains .github/workflows/release.yml '.protected_branch==true and .rollback_drill==true and .recovery_drill==true and .r2_delivery==true'
contains .github/workflows/release.yml '.secure_boot==true'
contains .github/workflows/release.yml 'Reject stale verification evidence'

# Non-dev publishing must be coupled to a fresh successful cross-component audit.
contains scripts/check-release-gate-freshness.sh "require_fresh stability-security 'stability-security-crosscheck.yml'"
contains scripts/check-release-gate-freshness.sh "'^(cmd/|internal/|configs/|container/|systemd/|scripts/|release/|\\.github/workflows/|go\\.(mod|sum)$)'"

# Critical release/security-chain Actions must be immutable rather than movable major tags.
contains .github/workflows/release.yml 'actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803'
contains .github/workflows/release.yml 'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e'
contains .github/workflows/stability-security-crosscheck.yml 'actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803'
contains .github/workflows/stability-security-crosscheck.yml 'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e'
contains .github/workflows/codeql.yml 'actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803'
contains .github/workflows/codeql.yml 'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e'
contains .github/workflows/codeql.yml 'github/codeql-action/init@ff2f1c621b7f889edc0d3c761ac2e6a3f8cdb0dd'
contains .github/workflows/codeql.yml 'github/codeql-action/analyze@ff2f1c621b7f889edc0d3c761ac2e6a3f8cdb0dd'
contains .github/workflows/codeql.yml 'build-mode: manual'
not_contains .github/workflows/release.yml 'actions/checkout@v6'
not_contains .github/workflows/release.yml 'actions/setup-go@v7'
not_contains .github/workflows/stability-security-crosscheck.yml 'actions/checkout@v6'
not_contains .github/workflows/stability-security-crosscheck.yml 'actions/setup-go@v7'
not_contains .github/workflows/codeql.yml 'actions/checkout@v6'
not_contains .github/workflows/codeql.yml 'actions/setup-go@v7'
not_contains .github/workflows/codeql.yml 'github/codeql-action/init@v4'
not_contains .github/workflows/codeql.yml 'github/codeql-action/analyze@v4'

# Installable profiles must remain explicit and production Secure Boot must stay false
# until the production signing chain is implemented and proven.
contains scripts/build-live-iso.sh 'installable = profile in {"server", "sentinel"}'
contains scripts/build-live-iso.sh '"secure_boot": False'
contains scripts/build-live-iso.sh 'GPT+EFI+ROOT_A+ROOT_B+LUKS_STATE'
contains scripts/build-live-iso.sh 'confirmation": "ERASE:<target>"'
contains scripts/build-live-iso.sh 'MENU="Try / Install KINGAI OS Server"'
contains scripts/build-live-iso.sh 'MENU="Try / Install KINGAI OS Sentinel"'

# Fail on accidental obvious TCP management binding in system service definitions.
if grep -REn '(^|[[:space:]])(0\.0\.0\.0|\[::\]|127\.0\.0\.1):[0-9]+|ListenStream=[0-9]+' systemd; then
  fail 'systemd management surface contains a TCP listener'
fi

# Basic syntax/data sanity across release-critical files.
python3 -m json.tool release/gates.json >/dev/null
for f in configs/*.json; do python3 -m json.tool "$f" >/dev/null; done
bash -n scripts/*.sh

echo 'KINGAI OS cross-component stability/security invariants passed.'
