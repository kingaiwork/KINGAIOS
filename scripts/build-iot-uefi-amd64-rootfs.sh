#!/usr/bin/env bash
set -euo pipefail

OUT="${1:-dist}"
ROOT="${OUT}/rootfs-iot-amd64"
PKG_FILE="distro/packages/iot-uefi-amd64.txt"
VERSION="$(tr -d '[:space:]' < VERSION)"

[[ $EUID -eq 0 ]] || { echo "build-iot-uefi-amd64-rootfs.sh must run as root" >&2; exit 1; }
[[ -f "$PKG_FILE" ]] || { echo "missing $PKG_FILE" >&2; exit 1; }
command -v chroot >/dev/null || { echo "chroot is required" >&2; exit 1; }

KINGAI_SKIP_ARCHIVE=1 bash scripts/build-rootfs.sh iot amd64 "$OUT"
[[ -d "$ROOT" ]] || { echo "missing IoT rootfs: $ROOT" >&2; exit 1; }

mapfile -t packages < <(grep -Ev '^\s*(#|$)' "$PKG_FILE")
((${#packages[@]} > 0)) || { echo "empty UEFI package set" >&2; exit 1; }

backup="$ROOT/etc/resolv.conf.kingai-backup"
rm -f "$backup"
if [[ -e "$ROOT/etc/resolv.conf" || -L "$ROOT/etc/resolv.conf" ]]; then
  cp -a --no-dereference "$ROOT/etc/resolv.conf" "$backup"
fi
rm -f "$ROOT/etc/resolv.conf"
cp -L /etc/resolv.conf "$ROOT/etc/resolv.conf"
chmod 0644 "$ROOT/etc/resolv.conf"

mounted_dev=0
mounted_proc=0
mounted_sys=0
cleanup() {
  if (( mounted_sys == 1 )); then umount -R "$ROOT/sys" 2>/dev/null || true; fi
  if (( mounted_proc == 1 )); then umount -R "$ROOT/proc" 2>/dev/null || true; fi
  if (( mounted_dev == 1 )); then umount -R "$ROOT/dev" 2>/dev/null || true; fi
  rm -f "$ROOT/etc/resolv.conf"
  if [[ -e "$backup" || -L "$backup" ]]; then
    mv "$backup" "$ROOT/etc/resolv.conf"
  fi
}
trap cleanup EXIT

mount --rbind /dev "$ROOT/dev"
mount --make-rslave "$ROOT/dev"
mounted_dev=1
mount -t proc proc "$ROOT/proc"
mounted_proc=1
mount --rbind /sys "$ROOT/sys"
mount --make-rslave "$ROOT/sys"
mounted_sys=1

chroot "$ROOT" apt-get update
DEBIAN_FRONTEND=noninteractive chroot "$ROOT" apt-get install -y --no-install-recommends "${packages[@]}"
chroot "$ROOT" update-initramfs -u -k all
rm -rf "$ROOT/var/lib/apt/lists"/*

# Bootable IoT/Edge systems are headless by default.
mkdir -p "$ROOT/etc/systemd/system"
ln -sfn /usr/lib/systemd/system/multi-user.target "$ROOT/etc/systemd/system/default.target"

# Refresh the SBOM after adding the board-release kernel/runtime packages.
chroot "$ROOT" dpkg-query -W -f='${binary:Package}\t${Version}\t${Architecture}\n' | LC_ALL=C sort > "$ROOT/usr/share/kingai/legal/packages.tsv"
python3 scripts/generate-sbom.py \
  "$ROOT/usr/share/kingai/legal/packages.tsv" \
  "$ROOT/usr/share/kingai/legal/KINGAI-OS.spdx.json" \
  "$VERSION" "iot-uefi-amd64" "amd64"

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

# Drop host mounts before validating final on-disk layout.
cleanup
mounted_dev=mounted_proc=mounted_sys=0
trap - EXIT

kernel=$(find "$ROOT/boot" -maxdepth 1 -name 'vmlinuz-*' -print -quit)
initrd=$(find "$ROOT/boot" -maxdepth 1 -name 'initrd.img-*' -print -quit)
[[ -n "$kernel" && -s "$kernel" ]] || { echo "UEFI Edge kernel missing" >&2; exit 1; }
[[ -n "$initrd" && -s "$initrd" ]] || { echo "UEFI Edge initramfs missing" >&2; exit 1; }
for tool in cryptsetup blkid grub-editenv; do
  chroot "$ROOT" sh -c "command -v $tool >/dev/null" || { echo "UEFI Edge runtime missing: $tool" >&2; exit 1; }
done
[[ "$(readlink "$ROOT/etc/systemd/system/default.target")" == "/usr/lib/systemd/system/multi-user.target" ]]
[[ ! -e "$ROOT/usr/lib/kingai/kingai-execd" ]]

echo "Built KINGAI OS IoT Generic UEFI amd64 board rootfs: $ROOT"
