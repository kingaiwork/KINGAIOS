#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-server}"
OUT="${2:-dist}"
ARCH="amd64"
VERSION="$(tr -d '[:space:]' < VERSION)"
ROOT="${OUT}/rootfs-${PROFILE}-${ARCH}"
WORK="${OUT}/iso-work-${PROFILE}"
ISO_ROOT="${WORK}/iso"
ISO="${OUT}/KINGAI-OS-${PROFILE^}-${VERSION}-${ARCH}.iso"
SBOM="$ROOT/usr/share/kingai/legal/KINGAI-OS.spdx.json"

case "$PROFILE" in server|desktop|recovery) ;; *) echo "live ISO supports server, desktop or recovery" >&2; exit 2;; esac
for tool in mksquashfs grub-mkrescue xorriso md5sum; do command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 1; }; done
[[ -d "$ROOT" ]] || { echo "missing $ROOT; run scripts/build-rootfs.sh $PROFILE amd64 $OUT first" >&2; exit 1; }
[[ -f "$SBOM" ]] || { echo "missing SBOM: $SBOM" >&2; exit 1; }
KERNEL=$(find "$ROOT/boot" -maxdepth 1 -name 'vmlinuz-*' -printf '%p\n' | sort -V | tail -1)
INITRD=$(find "$ROOT/boot" -maxdepth 1 -name 'initrd.img-*' -printf '%p\n' | sort -V | tail -1)
[[ -n "$KERNEL" && -n "$INITRD" ]] || { echo "kernel/initrd missing" >&2; exit 1; }
rm -rf "$WORK";mkdir -p "$ISO_ROOT/casper" "$ISO_ROOT/boot/grub" "$ISO_ROOT/.disk" "$ISO_ROOT/sbom"
cp "$KERNEL" "$ISO_ROOT/casper/vmlinuz";cp "$INITRD" "$ISO_ROOT/casper/initrd";cp "$SBOM" "$ISO_ROOT/sbom/KINGAI-OS.spdx.json"
mksquashfs "$ROOT" "$ISO_ROOT/casper/filesystem.squashfs" -comp xz -b 1M -noappend -no-xattrs -wildcards -e 'boot/vmlinuz-*' 'boot/initrd.img-*'
du -sx --block-size=1 "$ROOT" | cut -f1 > "$ISO_ROOT/casper/filesystem.size"
chroot "$ROOT" dpkg-query -W -f='${binary:Package} ${Version}\n' | LC_ALL=C sort > "$ISO_ROOT/casper/filesystem.manifest"
printf 'KINGAI OS %s %s amd64\n' "$VERSION" "${PROFILE^}" > "$ISO_ROOT/.disk/info"
if [[ "$PROFILE" == "recovery" ]]; then MENU="KINGAI OS Recovery"; EXTRA="systemd.unit=multi-user.target"; else MENU="Try KINGAI OS ${PROFILE^}"; EXTRA="quiet splash"; fi
cat > "$ISO_ROOT/boot/grub/grub.cfg" <<EOF
set timeout=5
set default=0
if serial --unit=0 --speed=115200 --word=8 --parity=no --stop=1; then
  terminal_input console serial
  terminal_output console serial
fi
menuentry "$MENU" {
    linux /casper/vmlinuz boot=casper $EXTRA username=kingai hostname=kingai console=tty0 console=ttyS0,115200n8 ---
    initrd /casper/initrd
}
menuentry "$MENU (safe graphics)" {
    linux /casper/vmlinuz boot=casper nomodeset systemd.unit=multi-user.target username=kingai hostname=kingai console=tty0 console=ttyS0,115200n8 ---
    initrd /casper/initrd
}
EOF
(cd "$ISO_ROOT";find . -type f ! -name md5sum.txt -print0 | LC_ALL=C sort -z | xargs -0 md5sum > md5sum.txt)
grub-mkrescue -o "$ISO" "$ISO_ROOT";sha256sum "$ISO" > "$ISO.sha256";cp "$SBOM" "$ISO.spdx.json"
SIZE=$(stat -c '%s' "$ISO");SBOM_SHA=$(sha256sum "$ISO.spdx.json" | awk '{print $1}')
python3 - "$ISO" "$PROFILE" "$VERSION" "$SIZE" "$SBOM_SHA" > "$ISO.manifest.json" <<'PY'
import hashlib, json, os, sys
path, profile, version, size, sbom_sha = sys.argv[1:]
with open(path, "rb") as f: sha = hashlib.file_digest(f, "sha256").hexdigest()
print(json.dumps({"product":"KINGAI OS","version":version,"profile":profile,"arch":"amd64","artifact":os.path.basename(path),"size_bytes":int(size),"sha256":sha,"sbom":{"artifact":os.path.basename(path)+".spdx.json","sha256":sbom_sha,"format":"SPDX-2.3"},"stage":"developer-foundation","installable":False,"secure_boot":False,"boot_mode":["UEFI","BIOS"],"offline_recovery":profile=="recovery"}, indent=2))
PY
echo "Built $ISO ($(numfmt --to=iec "$SIZE"))"
if [[ "$PROFILE" == "recovery" ]]; then echo "Offline Recovery ISO: no SSH service, no automatic agent/update daemon."; else echo "Developer Live ISO: production release gates remain enforced."; fi
