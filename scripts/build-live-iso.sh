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

case "$PROFILE" in server|desktop) ;; *) echo "live ISO currently supports server or desktop" >&2; exit 2;; esac
for tool in mksquashfs xorriso grub-mkstandalone mformat mmd mcopy; do command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 1; }; done
[[ -d "$ROOT" ]] || { echo "missing $ROOT; run scripts/build-rootfs.sh $PROFILE amd64 $OUT first" >&2; exit 1; }

KERNEL=$(find "$ROOT/boot" -maxdepth 1 -name 'vmlinuz-*' -printf '%p\n' | sort -V | tail -1)
INITRD=$(find "$ROOT/boot" -maxdepth 1 -name 'initrd.img-*' -printf '%p\n' | sort -V | tail -1)
[[ -n "$KERNEL" && -n "$INITRD" ]] || { echo "kernel/initrd missing" >&2; exit 1; }

rm -rf "$WORK"
mkdir -p "$ISO_ROOT/casper" "$ISO_ROOT/boot/grub" "$ISO_ROOT/EFI/BOOT"
cp "$KERNEL" "$ISO_ROOT/casper/vmlinuz"
cp "$INITRD" "$ISO_ROOT/casper/initrd"

mksquashfs "$ROOT" "$ISO_ROOT/casper/filesystem.squashfs" -comp xz -b 1M -noappend -wildcards \
  -e 'boot/vmlinuz-*' 'boot/initrd.img-*'
du -sx --block-size=1 "$ROOT" | cut -f1 > "$ISO_ROOT/casper/filesystem.size"

cat > "$ISO_ROOT/boot/grub/grub.cfg" <<EOF
set timeout=5
set default=0

menuentry "Try KINGAI OS ${PROFILE^}" {
    linux /casper/vmlinuz boot=casper quiet splash username=kingai hostname=kingai ---
    initrd /casper/initrd
}

menuentry "Try KINGAI OS ${PROFILE^} (safe graphics)" {
    linux /casper/vmlinuz boot=casper nomodeset username=kingai hostname=kingai ---
    initrd /casper/initrd
}
EOF

# UEFI boot image.
grub-mkstandalone -O x86_64-efi -o "$ISO_ROOT/EFI/BOOT/BOOTX64.EFI" \
  --locales="" --fonts="" "boot/grub/grub.cfg=$ISO_ROOT/boot/grub/grub.cfg"
EFI_IMG="$ISO_ROOT/boot/grub/efi.img"
dd if=/dev/zero of="$EFI_IMG" bs=1M count=64 status=none
mformat -i "$EFI_IMG" -F ::
mmd -i "$EFI_IMG" ::/EFI ::/EFI/BOOT
mcopy -i "$EFI_IMG" "$ISO_ROOT/EFI/BOOT/BOOTX64.EFI" ::/EFI/BOOT/BOOTX64.EFI

# Legacy BIOS boot image.
grub-mkstandalone -O i386-pc -o "$WORK/core.img" --locales="" --fonts="" \
  "boot/grub/grub.cfg=$ISO_ROOT/boot/grub/grub.cfg"
cat /usr/lib/grub/i386-pc/cdboot.img "$WORK/core.img" > "$ISO_ROOT/boot/grub/bios.img"

xorriso -as mkisofs \
  -iso-level 3 -full-iso9660-filenames -volid "KINGAI_OS" \
  -output "$ISO" \
  -eltorito-boot boot/grub/bios.img -no-emul-boot -boot-load-size 4 -boot-info-table \
  -eltorito-catalog boot/grub/boot.cat \
  -eltorito-alt-boot -e boot/grub/efi.img -no-emul-boot \
  -isohybrid-gpt-basdat "$ISO_ROOT"

sha256sum "$ISO" > "$ISO.sha256"
SIZE=$(stat -c '%s' "$ISO")
python3 - "$ISO" "$PROFILE" "$VERSION" "$SIZE" > "$ISO.manifest.json" <<'PY'
import json, os, sys, hashlib
path, profile, version, size = sys.argv[1:]
with open(path, 'rb') as f:
    sha = hashlib.file_digest(f, 'sha256').hexdigest()
print(json.dumps({
    'product':'KINGAI OS', 'version':version, 'profile':profile, 'arch':'amd64',
    'artifact':os.path.basename(path), 'size_bytes':int(size), 'sha256':sha,
    'stage':'developer-foundation', 'installable':False, 'boot_mode':['UEFI','BIOS']
}, indent=2))
PY

echo "Built $ISO"
echo "NOTE: This is a bootable Developer Live ISO. Installer enablement remains gated until destructive-disk tests pass."
