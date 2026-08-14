#!/bin/sh
set -eu

CONF=/etc/kingai/state-migration.conf
STATE_ROOT=/var/lib/kingai-state
MARKER="$STATE_ROOT/kingai/runtime/.layout-v1-ready"
SOURCE_MOUNT=/run/kingai-state-migrate/source

# Fresh installs create the marker during installation. Updated legacy installs
# receive CONF in the staged target slot and migrate once before kingaid starts.
if [ -e "$MARKER" ]; then
  rm -f "$CONF"
  exit 0
fi

if [ ! -r "$CONF" ]; then
  echo "KINGAI STATE migration metadata is missing" >&2
  exit 1
fi
if ! mountpoint -q "$STATE_ROOT"; then
  echo "encrypted KINGAI_STATE is not mounted" >&2
  exit 1
fi
if ! mountpoint -q /var/lib/kingai || ! mountpoint -q /var/log/kingai; then
  echo "KINGAI runtime bind mounts are not active" >&2
  exit 1
fi

SOURCE_ROOT_UUID=$(sed -n 's/^SOURCE_ROOT_UUID=//p' "$CONF" | head -n1)
SOURCE_SLOT=$(sed -n 's/^SOURCE_SLOT=//p' "$CONF" | head -n1)
case "$SOURCE_ROOT_UUID" in
  ''|*[!0-9A-Fa-f-]*) echo "invalid migration source root UUID" >&2; exit 1 ;;
esac
case "$SOURCE_SLOT" in
  A|B) ;;
  *) echo "invalid migration source slot" >&2; exit 1 ;;
esac

SOURCE_DEVICE=$(blkid -U "$SOURCE_ROOT_UUID" 2>/dev/null || true)
if [ -z "$SOURCE_DEVICE" ] || [ ! -b "$SOURCE_DEVICE" ]; then
  echo "migration source root device cannot be resolved" >&2
  exit 1
fi

mkdir -p "$SOURCE_MOUNT"
cleanup() {
  if mountpoint -q "$SOURCE_MOUNT"; then
    umount "$SOURCE_MOUNT" || true
  fi
}
trap cleanup EXIT INT TERM HUP

mount -o ro "$SOURCE_DEVICE" "$SOURCE_MOUNT"
if ! grep -q '^NAME="KINGAI OS"' "$SOURCE_MOUNT/usr/lib/os-release" 2>/dev/null; then
  echo "migration source is not a KINGAI OS root" >&2
  exit 1
fi

mkdir -p /var/lib/kingai /var/log/kingai "$STATE_ROOT/kingai/runtime"
chmod 0700 "$STATE_ROOT/kingai/runtime"

if [ -d "$SOURCE_MOUNT/var/lib/kingai" ]; then
  rsync -aHAX --numeric-ids "$SOURCE_MOUNT/var/lib/kingai/" /var/lib/kingai/
fi
if [ -d "$SOURCE_MOUNT/var/log/kingai" ]; then
  rsync -aHAX --numeric-ids "$SOURCE_MOUNT/var/log/kingai/" /var/log/kingai/
fi

sync
printf 'layout=1\nsource_slot=%s\nsource_root_uuid=%s\n' "$SOURCE_SLOT" "$SOURCE_ROOT_UUID" > "$MARKER.tmp"
chmod 0600 "$MARKER.tmp"
mv -f "$MARKER.tmp" "$MARKER"
sync
rm -f "$CONF"

echo "KINGAI encrypted STATE runtime migration completed from slot $SOURCE_SLOT"
