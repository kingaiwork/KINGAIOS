#!/usr/bin/env bash
set -euo pipefail

OUT="${1:-dist}"
ROOT="${OUT}/rootfs-iot-amd64"
BASE_PKG_FILE="distro/packages/iot.txt"
UEFI_PKG_FILE="distro/packages/iot-uefi-amd64.txt"
VERSION="$(tr -d '[:space:]' < VERSION)"

[[ $EUID -eq 0 ]] || { echo "build-iot-uefi-amd64-rootfs.sh must run as root" >&2; exit 1; }
[[ -f "$BASE_PKG_FILE" ]] || { echo "missing $BASE_PKG_FILE" >&2; exit 1; }
[[ -f "$UEFI_PKG_FILE" ]] || { echo "missing $UEFI_PKG_FILE" >&2; exit 1; }
command -v python3 >/dev/null || { echo "python3 is required" >&2; exit 1; }
command -v chroot >/dev/null || { echo "chroot is required" >&2; exit 1; }
command -v sha256sum >/dev/null || { echo "sha256sum is required" >&2; exit 1; }
base_sha_before=$(sha256sum "$BASE_PKG_FILE" | awk '{print $1}')

# The generic IoT foundation intentionally remains minbase and does not carry
# apt or a kernel. For this explicit board-release build only, construct a
# temporary combined mmdebstrap package manifest so kernel/boot packages are
# installed in the original rootfs transaction instead of mutating a finished
# trusted root filesystem with chroot apt-get.
tmp=$(mktemp -d)
backup="$tmp/iot.txt.original"
combined="$tmp/iot.txt.combined"
cp -a "$BASE_PKG_FILE" "$backup"

restore_package_manifest() {
  cp -a "$backup" "$BASE_PKG_FILE" 2>/dev/null || true
  rm -rf "$tmp"
}
trap restore_package_manifest EXIT

python3 - "$BASE_PKG_FILE" "$UEFI_PKG_FILE" "$combined" <<'PY'
from pathlib import Path
import sys

base, extra, out = map(Path, sys.argv[1:])
seen = set()
lines = []
for source in (base, extra):
    for raw in source.read_text(encoding="utf-8").splitlines():
        value = raw.strip()
        if not value or value.startswith("#") or value in seen:
            continue
        seen.add(value)
        lines.append(value)
out.write_text("\n".join(lines) + "\n", encoding="utf-8")
PY
install -m0644 "$combined" "$BASE_PKG_FILE"

KINGAI_SKIP_ARCHIVE=1 bash scripts/build-rootfs.sh iot amd64 "$OUT"
restore_package_manifest
trap - EXIT

base_sha_after=$(sha256sum "$BASE_PKG_FILE" | awk '{print $1}')
[[ "$base_sha_after" == "$base_sha_before" ]] || {
  echo "Generic IoT package manifest was not restored after board build" >&2
  exit 1
}

[[ -d "$ROOT" ]] || { echo "missing IoT rootfs: $ROOT" >&2; exit 1; }

# Linux package hooks normally create the initramfs during mmdebstrap. Refresh
# it once more from the final package set without requiring apt or host mounts.
chroot "$ROOT" update-initramfs -u -k all

# Bootable IoT/Edge systems are headless by default.
mkdir -p "$ROOT/etc/systemd/system"
ln -sfn /usr/lib/systemd/system/multi-user.target "$ROOT/etc/systemd/system/default.target"

mkdir -p "$ROOT/usr/share/kingai/iot"
python3 - "$ROOT/usr/share/kingai/iot/board-release.json" "$VERSION" <<'PY'
import json, sys
path, version = sys.argv[1:]
data = {
    "schema": 1,
    "product": "KINGAI OS IoT / Edge",
    "board_release": "generic-uefi-amd64",
    "version": version,
    "arch": "amd64",
    "boot_method": "uefi",
    "partition_model": "gpt-efi-root_a-root_b-luks_state",
    "update_model": "grub-a-b",
    "device_pack_required": True,
    "vm_boot_candidate": True,
    "hardware_verified": False,
    "production_release": False
}
with open(path, "w", encoding="utf-8") as fh:
    json.dump(data, fh, indent=2)
    fh.write("\n")
PY

kernel=$(find "$ROOT/boot" -maxdepth 1 -name 'vmlinuz-*' -print -quit)
initrd=$(find "$ROOT/boot" -maxdepth 1 -name 'initrd.img-*' -print -quit)
[[ -n "$kernel" && -s "$kernel" ]] || { echo "UEFI Edge kernel missing" >&2; exit 1; }
[[ -n "$initrd" && -s "$initrd" ]] || { echo "UEFI Edge initramfs missing" >&2; exit 1; }
for tool in cryptsetup blkid grub-editenv; do
  chroot "$ROOT" sh -c "command -v $tool >/dev/null" || { echo "UEFI Edge runtime missing: $tool" >&2; exit 1; }
done
[[ "$(readlink "$ROOT/etc/systemd/system/default.target")" == "/usr/lib/systemd/system/multi-user.target" ]]
[[ ! -e "$ROOT/usr/lib/kingai/kingai-execd" ]]
[[ ! -e "$ROOT/usr/lib/kingai/kingai-installer" ]]

echo "Built KINGAI OS IoT Generic UEFI amd64 board rootfs: $ROOT"
