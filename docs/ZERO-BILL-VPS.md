# KINGAI OS Zero-Bill VPS Migration

KINGAI OS previously had many independent GitHub Actions workflows for core CI, CodeQL, containers, rootfs, ISO builds, desktop screenshots, Sentinel, VM smoke tests and release publication. In Zero-Bill mode those workflows are retired on the migration branch and replaced by one owned-VPS workflow: `.github/workflows/zero-bill-vps.yml`.

Git history preserves every retired workflow. Do not restore an old workflow unless it uses only the owned self-hosted runner and no external artifact/release/registry paid path.

## VPS requirements

Register a self-hosted runner with labels:

```text
self-hosted
linux
x64
kingai-vps
```

Prepare local storage:

```bash
sudo mkdir -p /srv/kingai/{qa/KINGAIOS,releases/KINGAIOS,builds/KINGAIOS,archives/KINGAIOS}
sudo chown -R "$USER":"$USER" /srv/kingai
```

The runner should have the distro/package prerequisites already required by KINGAI OS image builders, plus Go, Python, Docker/Podman, QEMU and KVM access. Heavy image builders may require reviewed passwordless sudo for narrowly scoped local build commands. Do not give the runner Cloudflare, Supabase, paid AI or infrastructure-billing credentials.

## Modes

Core automatic validation:

```bash
bash scripts/zero-bill-vps-validate.sh core
```

Image build:

```bash
bash scripts/zero-bill-vps-validate.sh images
```

KVM readiness / VM migration gate:

```bash
bash scripts/zero-bill-vps-validate.sh vm
```

Local release candidate:

```bash
bash scripts/zero-bill-vps-validate.sh release
```

Outputs remain under `/srv/kingai` and are not uploaded automatically to GitHub Releases, Actions artifacts, container registries or other metered storage.

## VM smoke parity migration

The historical VM workflows contained valuable install/recovery/A-B update/secure-boot/state-unlock/runtime-persistence coverage. They are preserved in Git history but intentionally not treated as migrated merely because hosted Actions were removed.

OpenClaw/Codex on the VPS must migrate them, in order, into local scripts under `qa/vps/`:

1. installable live ISO boot
2. installer VM
3. installer desktop VM
4. recovery VM
5. secure boot VM
6. state passphrase unlock VM
7. A/B update VM
8. runtime persistence migration VM
9. desktop ISO/rootfs screenshots
10. server/rootfs/IoT smoke

Each local test must save command logs, exit status, VM serial output and relevant screenshots under `/srv/kingai/qa/KINGAIOS/<sha>/vm/`.

Release parity is not considered complete until all migrated local scripts pass on the real VPS.

## OpenClaw / Codex deployment instruction

```text
Operate kingaiwork/KINGAIOS in ZERO-BILL mode. Read .kingai/zero-bill.json and the central KINGAIASE policy. Use only the owned VPS for CI, ISO/rootfs/container builds, KVM smoke, desktop screenshots, security scans and release candidates. Never restore GitHub-hosted runners, Actions artifact uploads, CodeQL/GHAS, GitHub Release auto-upload, registry pushes, paid AI APIs, paid cloud storage or automatic publication. Migrate each historical VM workflow from git history into qa/vps local scripts, preserving its assertions. Store all evidence under /srv/kingai. Do not claim parity until every local VM test passes. Do not merge or publish releases automatically.
```

## Truth boundary

This GitHub branch retires hosted automation and defines the VPS replacement. It does not prove that the VPS currently has KVM, image-build packages, runner registration or sufficient local storage. Those must be verified on the machine with logs before this migration is merged as release-ready.
