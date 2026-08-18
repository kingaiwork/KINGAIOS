#!/usr/bin/env bash
set -euo pipefail

fail() { echo "cross-check: $*" >&2; exit 1; }
need() { [[ -f "$1" ]] || fail "missing required file: $1"; }
contains() { grep -Fq -- "$2" "$1" || fail "$1 missing invariant: $2"; }
not_contains() { ! grep -Fq -- "$2" "$1" || fail "$1 contains forbidden invariant: $2"; }

for f in \
  go.mod \
  release/gates.json \
  .github/workflows/release.yml \
  .github/workflows/stability-security-crosscheck.yml \
  .github/workflows/codeql.yml \
  .github/workflows/govulncheck.yml \
  .github/workflows/smoke-runtime-persistence-migration-vm.yml \
  .github/workflows/smoke-installable-desktop-iso.yml \
  .github/workflows/smoke-desktop-ab-update.yml \
  scripts/check-release-gate-freshness.sh \
  scripts/build-live-iso.sh \
  scripts/kingai-install-live.sh \
  cmd/kingai-update/main.go \
  internal/installer/executor.go \
  internal/update/executor.go \
  internal/update/persistence.go \
  distro/overlay/usr/lib/kingai/kingai-configure-desktop-install \
  distro/overlay/usr/lib/kingai/kingai-installer-gui \
  distro/overlay/usr/share/applications/kingai-installer.desktop \
  systemd/kingaid.service \
  systemd/kingai-execd.service \
  container/Dockerfile; do
  need "$f"
done

python3 - <<'PY'
import json
import re
from pathlib import Path

p = Path('release/gates.json')
g = json.loads(p.read_text())
required_bool = [
    'd5_runtime_ci','container_image_ci','container_multiarch_ci',
    'container_security_scan_ci','container_oci_archive_ci',
    'container_release_pipeline','container_signed_release','installer_vm',
    'installable_live_iso_vm','desktop_live_vm','desktop_install_vm',
    'desktop_ab_update_vm','server_bios_uefi_vm','ab_update_vm','recovery_vm',
    'tuf_client','secure_boot_vm','tuf_repository','production_signing_ready',
    'protected_branch','rollback_drill','recovery_drill','r2_delivery',
]
for key in required_bool:
    if key not in g or not isinstance(g[key], bool):
        raise SystemExit(f'cross-check: gate {key!r} must exist and be boolean')

for key in (
    'installer_vm','installable_live_iso_vm','desktop_live_vm','desktop_install_vm',
    'desktop_ab_update_vm','ab_update_vm','recovery_vm','tuf_client',
):
    if g[key] is not True:
        raise SystemExit(f'cross-check: installable Desktop/Beta foundation requires {key}=true')

if g['production_signing_ready'] and not g['tuf_repository']:
    raise SystemExit('cross-check: production signing cannot be ready before production TUF repository')
if g['r2_delivery'] and not g['production_signing_ready']:
    raise SystemExit('cross-check: production delivery cannot precede production signing readiness')

# Keep host CI and container builds on the same security-patched Go line.
gomod = Path('go.mod').read_text()
m = re.search(r'^go\s+(\d+\.\d+\.\d+)\s*$', gomod, re.M)
if not m:
    raise SystemExit('cross-check: go.mod must pin an exact patch Go version')
go_version = m.group(1)
if go_version != '1.25.13':
    raise SystemExit(f'cross-check: expected security toolchain go1.25.13, found go{go_version}')
docker = Path('container/Dockerfile').read_text()
if f'ARG GO_IMAGE=golang:{go_version}-bookworm' not in docker:
    raise SystemExit('cross-check: container Go builder must match the go.mod security patch version')

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

# Installed runtime data must live on encrypted persistent STATE, never silently
# fall back to an A/B root slot when STATE is unavailable.
contains internal/installer/executor.go '/dev/mapper/KINGAI_STATE /var/lib/kingai-state ext4 defaults 0 2'
contains internal/installer/executor.go '/var/lib/kingai-state/kingai/runtime/lib /var/lib/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state'
contains internal/installer/executor.go '/var/lib/kingai-state/kingai/runtime/log /var/log/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state'
contains internal/installer/executor.go 'kingai/runtime/.layout-v1-ready'
contains internal/installer/executor.go 'origin=fresh-install'
not_contains internal/installer/executor.go 'KINGAI_STATE /var/lib/kingai-state ext4 defaults,nofail'

# Personal Desktop data must be encrypted and survive A/B slot changes. First
# user secrets must be file-based/private rather than command-line password data.
contains distro/overlay/usr/lib/kingai/kingai-configure-desktop-install '/var/lib/kingai-state/home /home none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state'
contains distro/overlay/usr/lib/kingai/kingai-configure-desktop-install 'install -d -m0700 -o "$uid_a" -g "$gid_a" "$state/home/$USERNAME"'
contains distro/overlay/usr/lib/kingai/kingai-configure-desktop-install 'UID/GID 1000'
contains scripts/kingai-install-live.sh '--user-password-file'
contains scripts/kingai-install-live.sh 'Caller-provided secret files are copied into a root-only'
contains distro/overlay/usr/lib/kingai/kingai-installer-gui 'select(.type == "disk" and (.ro|not) and (.rm|not) and (.size >= 42949672960)'
contains distro/overlay/usr/lib/kingai/kingai-installer-gui 'sudo -n /usr/sbin/kingai-install'
contains distro/overlay/usr/share/applications/kingai-installer.desktop 'Name=Install KINGAI OS'

# A/B updates must rewrite the target persistence layout, preserve Desktop human
# identity from the active slot, migrate pre-layout Betas before kingaid starts,
# and durably persist slot state across power loss.
contains internal/update/executor.go 'prepareTargetRuntimePersistence(targetRoot, activeRoot, stateMnt, state.ActiveSlot, aUUID, bUUID)'
contains internal/update/executor.go 'activeRoot := rootA'
contains internal/update/executor.go '"--exclude=/home/*"'
contains internal/update/executor.go 'required update tool missing: %s'
contains internal/update/executor.go '"findmnt"'
contains internal/update/executor.go 'atomicWriteDurable(path, append(b, '\''\n'\''), 0o600)'
contains internal/update/persistence.go 'preserveDesktopIdentity(activeRoot, targetRoot, stateRoot)'
contains internal/update/persistence.go 'etc/NetworkManager/system-connections'
contains internal/update/persistence.go 'var/lib/bluetooth'
contains internal/update/persistence.go 'var/lib/fprint'
contains internal/update/persistence.go 'ExecStart=/usr/lib/kingai/kingai-update migrate-state'
contains internal/update/persistence.go 'Requires=kingai-state-migrate.service'
contains internal/update/persistence.go 'mount", "-o", "ro"'
contains internal/update/persistence.go 'migration source unexpectedly resolves to the current root'
contains internal/update/persistence.go 'tmp.Sync()'
contains internal/update/persistence.go 'os.Rename(tmpName, path)'
contains internal/update/persistence.go 'syncDir(filepath.Dir(path))'
contains cmd/kingai-update/main.go 'case "migrate-state"'
contains cmd/kingai-update/main.go 'kingupdate.MigrateRuntimeState()'

# Exact Desktop install and A/B upgrade paths are first-class release evidence.
contains .github/workflows/smoke-installable-desktop-iso.yml 'KINGAI_DESKTOP_ENCRYPTED_HOME_OK'
contains .github/workflows/smoke-installable-desktop-iso.yml 'KINGAI_DESKTOP_FIRST_USER_OK'
contains .github/workflows/smoke-installable-desktop-iso.yml 'truncate -s 48G'
contains .github/workflows/smoke-desktop-ab-update.yml 'KINGAI_DESKTOP_AB_USER_OK'
contains .github/workflows/smoke-desktop-ab-update.yml 'KINGAI_DESKTOP_AB_HOME_OK'
contains .github/workflows/smoke-desktop-ab-update.yml 'KINGAI_DESKTOP_AB_WIFI_OK'
contains .github/workflows/smoke-desktop-ab-update.yml 'actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803'
contains .github/workflows/smoke-desktop-ab-update.yml 'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e'

# The exact legacy migration path is a release evidence requirement and its VM
# workflow must use immutable GitHub Action revisions.
contains scripts/check-release-gate-freshness.sh "require_fresh runtime-persistence 'smoke-runtime-persistence-migration-vm.yml'"
contains .github/workflows/smoke-runtime-persistence-migration-vm.yml 'actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803'
contains .github/workflows/smoke-runtime-persistence-migration-vm.yml 'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e'
contains .github/workflows/smoke-runtime-persistence-migration-vm.yml 'KINGAI_RUNTIME_STATE_MIGRATION_OK'
contains .github/workflows/smoke-runtime-persistence-migration-vm.yml 'defaults,nofail'
contains .github/workflows/smoke-runtime-persistence-migration-vm.yml 'source_slot=A'

# Container default must remain fixed non-root with Unix-socket management only.
contains container/Dockerfile 'ARG GO_IMAGE=golang:1.25.13-bookworm'
contains container/Dockerfile 'ARG KINGAI_UID=10001'
contains container/Dockerfile 'ARG KINGAI_GID=10001'
contains container/Dockerfile 'KINGAI_SOCKET=/run/kingai/kingaid.sock'
contains container/Dockerfile 'USER _kingai:kingai'
contains container/Dockerfile 'VOLUME ["/var/lib/kingai", "/var/log/kingai"]'
not_contains container/Dockerfile 'EXPOSE '

# Release policy must keep Developer/Beta/RC/Stable materially different.
contains .github/workflows/release.yml '.installer_vm==true and .installable_live_iso_vm==true and .ab_update_vm==true and .recovery_vm==true and .tuf_client==true'
contains .github/workflows/release.yml '.desktop_live_vm==true and .desktop_install_vm==true and .desktop_ab_update_vm==true'
contains .github/workflows/release.yml '.secure_boot_vm==true and .tuf_repository==true and .production_signing_ready==true'
contains .github/workflows/release.yml '.protected_branch==true and .rollback_drill==true and .recovery_drill==true and .r2_delivery==true'
contains .github/workflows/release.yml '.secure_boot==true'
contains .github/workflows/release.yml 'Reject stale verification evidence'
contains .github/workflows/release.yml 'kingai/release-run'
contains .github/workflows/release.yml 'actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a'

# Non-dev publishing must be coupled to fresh cross-component and vulnerability evidence.
contains scripts/check-release-gate-freshness.sh "require_fresh stability-security 'stability-security-crosscheck.yml'"
contains scripts/check-release-gate-freshness.sh "require_fresh go-vulnerability 'govulncheck.yml'"
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
contains .github/workflows/govulncheck.yml 'actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803'
contains .github/workflows/govulncheck.yml 'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e'
contains .github/workflows/govulncheck.yml 'golang.org/x/vuln/cmd/govulncheck@v1.6.0'
contains .github/workflows/govulncheck.yml 'GOTOOLCHAIN: local'
not_contains .github/workflows/release.yml 'actions/checkout@v6'
not_contains .github/workflows/release.yml 'actions/setup-go@v7'
not_contains .github/workflows/stability-security-crosscheck.yml 'actions/checkout@v6'
not_contains .github/workflows/stability-security-crosscheck.yml 'actions/setup-go@v7'
not_contains .github/workflows/codeql.yml 'actions/checkout@v6'
not_contains .github/workflows/codeql.yml 'actions/setup-go@v7'
not_contains .github/workflows/codeql.yml 'github/codeql-action/init@v4'
not_contains .github/workflows/codeql.yml 'github/codeql-action/analyze@v4'
not_contains .github/workflows/govulncheck.yml 'actions/checkout@v6'
not_contains .github/workflows/govulncheck.yml 'actions/setup-go@v7'

# Installable profiles must remain explicit and production Secure Boot must stay
# false until the production signing chain is implemented and proven.
contains scripts/build-live-iso.sh 'installable = profile in {"server", "desktop", "sentinel"}'
contains scripts/build-live-iso.sh '"secure_boot": False'
contains scripts/build-live-iso.sh 'GPT+EFI+ROOT_A+ROOT_B+LUKS_STATE'
contains scripts/build-live-iso.sh 'confirmation": "ERASE:<target>"'
contains scripts/build-live-iso.sh 'MENU="Try / Install KINGAI OS Server"'
contains scripts/build-live-iso.sh 'MENU="Try / Install KINGAI OS Desktop"'
contains scripts/build-live-iso.sh 'MENU="Try / Install KINGAI OS Sentinel"'

if grep -REn '(^|[[:space:]])(0\.0\.0\.0|\[::\]|127\.0\.0\.1):[0-9]+|ListenStream=[0-9]+' systemd; then
  fail 'systemd management surface contains a TCP listener'
fi

python3 -m json.tool release/gates.json >/dev/null
for f in configs/*.json; do python3 -m json.tool "$f" >/dev/null; done
bash -n scripts/*.sh distro/overlay/usr/lib/kingai/kingai-configure-desktop-install distro/overlay/usr/lib/kingai/kingai-installer-gui

echo 'KINGAI OS cross-component stability/security invariants passed.'
