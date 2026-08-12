#!/usr/bin/env bash
set -euo pipefail

ARCH="${1:-arm64}"
OUT="${2:-dist}"
PROFILE="iot"
VERSION="$(tr -d '[:space:]' < VERSION)"
ROOT="${OUT}/rootfs-${PROFILE}-${ARCH}"
RAW="${OUT}/KINGAI-OS-IoT-${VERSION}-${ARCH}.img"
XZ="${RAW}.xz"
MOUNT="${OUT}/iot-mnt-${ARCH}"

case "$ARCH" in amd64|arm64) ;; *) echo "invalid arch: $ARCH" >&2; exit 2;; esac
for tool in truncate mkfs.ext4 mount umount xz e2fsck python3; do
  command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 1; }
done
[[ $EUID -eq 0 ]] || { echo "build-iot-image.sh must run as root" >&2; exit 1; }
[[ -d "$ROOT" ]] || { echo "missing $ROOT; build the IoT rootfs first" >&2; exit 1; }

used=$(du -sx --block-size=1 "$ROOT" | cut -f1)
# Give the generic edge filesystem 40% growth room plus 256 MiB, rounded up to 64 MiB.
size=$(( used + used * 40 / 100 + 256 * 1024 * 1024 ))
step=$((64 * 1024 * 1024))
size=$(( (size + step - 1) / step * step ))
minimum=$((1024 * 1024 * 1024))
if (( size < minimum )); then size=$minimum; fi

rm -f "$RAW" "$XZ"
rm -rf "$MOUNT"
mkdir -p "$MOUNT"
truncate -s "$size" "$RAW"
mkfs.ext4 -q -F -L KINGAI_EDGE "$RAW"

mounted=0
cleanup() {
  if (( mounted == 1 )); then umount "$MOUNT" || true; fi
  rmdir "$MOUNT" 2>/dev/null || true
}
trap cleanup EXIT
mount -o loop "$RAW" "$MOUNT"
mounted=1
cp -a "$ROOT"/. "$MOUNT"/
sync
umount "$MOUNT"
mounted=0
rmdir "$MOUNT"
trap - EXIT

e2fsck -f -n "$RAW" >/dev/null
xz -T0 -9e -c "$RAW" > "$XZ"
sha256sum "$XZ" > "$XZ.sha256"
compressed=$(stat -c '%s' "$XZ")
rawbytes=$(stat -c '%s' "$RAW")
python3 - "$XZ" "$ARCH" "$VERSION" "$compressed" "$rawbytes" > "$XZ.manifest.json" <<'PY'
import hashlib, json, os, sys
path, arch, version, compressed, rawbytes = sys.argv[1:]
with open(path, 'rb') as f:
    sha = hashlib.file_digest(f, 'sha256').hexdigest()
print(json.dumps({
    'product': 'KINGAI OS',
    'edition': 'IoT / Edge',
    'version': version,
    'arch': arch,
    'artifact': os.path.basename(path),
    'format': 'ext4-rootfs.img.xz',
    'size_bytes': int(compressed),
    'raw_size_bytes': int(rawbytes),
    'sha256': sha,
    'bootable': False,
    'device_pack_required': True,
    'stage': 'developer-foundation'
}, indent=2))
PY
rm -f "$RAW"
echo "Built $XZ ($(numfmt --to=iec "$compressed"))"
echo "Generic Edge root filesystem image: board-specific kernel/bootloader Device Pack required."
