#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-server}"
ARCH="${2:-amd64}"
OUT="${3:-dist}"
SUITE="${KINGAI_UBUNTU_SUITE:-resolute}"
ROOT="${OUT}/rootfs-${PROFILE}-${ARCH}"
PKG_FILE="distro/packages/${PROFILE}.txt"

case "$PROFILE" in server|desktop|iot) ;; *) echo "invalid profile: $PROFILE" >&2; exit 2;; esac
case "$ARCH" in amd64|arm64) ;; *) echo "invalid arch: $ARCH" >&2; exit 2;; esac
command -v mmdebstrap >/dev/null || { echo "mmdebstrap is required" >&2; exit 1; }
command -v go >/dev/null || { echo "go is required" >&2; exit 1; }
[[ -f "$PKG_FILE" ]] || { echo "missing $PKG_FILE" >&2; exit 1; }
mkdir -p "$OUT"
rm -rf "$ROOT"

PACKAGES=$(grep -Ev '^\s*(#|$)' "$PKG_FILE" | paste -sd, -)
if [[ "$ARCH" == "arm64" ]]; then
  MIRROR="deb http://ports.ubuntu.com/ubuntu-ports ${SUITE} main universe restricted multiverse"
else
  MIRROR="deb http://archive.ubuntu.com/ubuntu ${SUITE} main universe restricted multiverse"
fi

mmdebstrap --variant=minbase --architectures="$ARCH" \
  --aptopt='APT::Install-Recommends "false"' \
  --include="$PACKAGES" "$SUITE" "$ROOT" "$MIRROR"

TARGET_GOARCH="$ARCH"; [[ "$ARCH" == "amd64" ]] && TARGET_GOARCH=amd64
CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_GOARCH" go build -trimpath -ldflags "-s -w -X main.version=$(cat VERSION)" -o "$OUT/kingai-$ARCH" ./cmd/kingai
CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_GOARCH" go build -trimpath -ldflags "-s -w -X main.version=$(cat VERSION)" -o "$OUT/kingaid-$ARCH" ./cmd/kingaid

install -Dm755 "$OUT/kingai-$ARCH" "$ROOT/usr/bin/kingai"
install -Dm755 "$OUT/kingaid-$ARCH" "$ROOT/usr/lib/kingai/kingaid"
install -Dm644 systemd/kingaid.service "$ROOT/usr/lib/systemd/system/kingaid.service"
install -Dm644 sysusers/kingai.conf "$ROOT/usr/lib/sysusers.d/kingai.conf"
install -Dm644 configs/policy.json "$ROOT/etc/kingai/policy.json"
install -Dm644 configs/system.json "$ROOT/etc/kingai/system.json"
install -Dm644 configs/models.json "$ROOT/etc/kingai/models.json"
cp -a distro/overlay/. "$ROOT/"
mkdir -p "$ROOT/etc/systemd/system/multi-user.target.wants"
ln -sfn /usr/lib/systemd/system/kingaid.service "$ROOT/etc/systemd/system/multi-user.target.wants/kingaid.service"

if [[ "$PROFILE" == "desktop" ]]; then
  mkdir -p "$ROOT/usr/share/kingai"
  cp -a desktop "$ROOT/usr/share/kingai/"
fi

tar --numeric-owner --xattrs --acls -C "$ROOT" -I 'zstd -19 -T0' -cf "$OUT/KINGAI-OS-${PROFILE}-${ARCH}-rootfs.tar.zst" .
sha256sum "$OUT/KINGAI-OS-${PROFILE}-${ARCH}-rootfs.tar.zst" > "$OUT/KINGAI-OS-${PROFILE}-${ARCH}-rootfs.tar.zst.sha256"
echo "Built $OUT/KINGAI-OS-${PROFILE}-${ARCH}-rootfs.tar.zst"
