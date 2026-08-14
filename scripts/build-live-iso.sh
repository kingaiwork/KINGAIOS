#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-server}"
OUT="${2:-dist}"
ARCH="amd64"
VERSION="$(tr -d '[:space:]' < VERSION)"
RELEASE_CHANNEL="${KINGAI_RELEASE_CHANNEL:-dev}"
case "$RELEASE_CHANNEL" in dev|beta|rc|stable) ;; *) echo "invalid release channel: $RELEASE_CHANNEL" >&2; exit 2;; esac
BASE_VERSION="$(bash scripts/release-base-version.sh "$VERSION")"
case "$RELEASE_CHANNEL" in
  dev) ARTIFACT_VERSION="$VERSION" ;;
  beta|rc) ARTIFACT_VERSION="${BASE_VERSION}-${RELEASE_CHANNEL}" ;;
  stable) ARTIFACT_VERSION="$BASE_VERSION" ;;
esac
ROOT="${OUT}/rootfs-${PROFILE}-${ARCH}"
WORK="${OUT}/iso-work-${PROFILE}"
ISO_ROOT="${WORK}/iso"
ISO="${OUT}/KINGAI-OS-${PROFILE^}-${ARTIFACT_VERSION}-${ARCH}.iso"
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
# The embedded squashfs is also the installer source. Keep /boot kernel and
# initramfs inside it even though Casper carries boot copies at /casper/ too.
# This small duplication makes the ISO self-contained for A/B disk installs.
mksquashfs "$ROOT" "$ISO_ROOT/casper/filesystem.squashfs" -comp xz -b 1M -noappend -no-xattrs
du -sx --block-size=1 "$ROOT" | cut -f1 > "$ISO_ROOT/casper/filesystem.size"
chroot "$ROOT" dpkg-query -W -f='${binary:Package} ${Version}\n' | LC_ALL=C sort > "$ISO_ROOT/casper/filesystem.manifest"
printf 'KINGAI OS %s %s amd64 (%s)\n' "$ARTIFACT_VERSION" "${PROFILE^}" "$RELEASE_CHANNEL" > "$ISO_ROOT/.disk/info"
if [[ "$PROFILE" == "recovery" ]]; then
  MENU="KINGAI OS Recovery"
  EXTRA="systemd.unit=multi-user.target"
elif [[ "$PROFILE" == "server" ]]; then
  MENU="Try / Install KINGAI OS Server"
  EXTRA="quiet splash"
else
  MENU="Try KINGAI OS ${PROFILE^}"
  EXTRA="quiet splash"
fi
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
python3 - "$ISO" "$PROFILE" "$ARTIFACT_VERSION" "$VERSION" "$RELEASE_CHANNEL" "$SIZE" "$SBOM_SHA" > "$ISO.manifest.json" <<'PY'
import hashlib, json, os, sys
path, profile, version, source_version, channel, size, sbom_sha = sys.argv[1:]
with open(path, "rb") as f:
    sha = hashlib.file_digest(f, "sha256").hexdigest()
installable = profile == "server"
stage = "installable-preview" if installable else "developer-foundation"
manifest = {
    "product": "KINGAI OS",
    "version": version,
    "source_version": source_version,
    "release_channel": channel,
    "profile": profile,
    "arch": "amd64",
    "artifact": os.path.basename(path),
    "size_bytes": int(size),
    "sha256": sha,
    "sbom": {
        "artifact": os.path.basename(path) + ".spdx.json",
        "sha256": sbom_sha,
        "format": "SPDX-2.3",
    },
    "stage": stage,
    "installable": installable,
    "secure_boot": False,
    "boot_mode": ["UEFI", "BIOS"],
    "offline_recovery": profile == "recovery",
}
if installable:
    manifest["installer"] = {
        "entrypoint": "kingai-install",
        "layout": "GPT+EFI+ROOT_A+ROOT_B+LUKS_STATE",
        "confirmation": "ERASE:<target>",
    }
print(json.dumps(manifest, indent=2))
PY
echo "Built $ISO ($(numfmt --to=iec "$SIZE"))"
if [[ "$PROFILE" == "recovery" ]]; then
  echo "Offline Recovery ISO: no SSH service, no automatic agent/update daemon."
elif [[ "$PROFILE" == "server" ]]; then
  echo "Installable Server Preview: use sudo kingai-install from the live session; Beta/Stable release gates remain enforced."
else
  echo "Developer Live ISO: production release gates remain enforced."
fi
