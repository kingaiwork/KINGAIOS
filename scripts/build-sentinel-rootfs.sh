#!/usr/bin/env bash
set -euo pipefail

ARCH="${1:-amd64}"
OUT="${2:-dist}"
VERSION="$(tr -d '[:space:]' < VERSION)"
BASE_ROOT="${OUT}/rootfs-server-${ARCH}"
ROOT="${OUT}/rootfs-sentinel-${ARCH}"
PKG_FILE="distro/packages/sentinel.txt"

case "$ARCH" in
  amd64|arm64) ;;
  *) echo "invalid arch: $ARCH" >&2; exit 2 ;;
esac

for tool in go tar zstd python3; do
  command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 1; }
done
[[ -f "$PKG_FILE" ]] || { echo "missing $PKG_FILE" >&2; exit 1; }

rm -rf "$BASE_ROOT" "$ROOT"
KINGAI_SKIP_ARCHIVE=1 bash scripts/build-rootfs.sh server "$ARCH" "$OUT"
mv "$BASE_ROOT" "$ROOT"

mapfile -t PACKAGES < <(grep -Ev '^\s*(#|$)' "$PKG_FILE")
if (( ${#PACKAGES[@]} > 0 )); then
  chroot "$ROOT" env DEBIAN_FRONTEND=noninteractive apt-get update
  chroot "$ROOT" env DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "${PACKAGES[@]}"
  chroot "$ROOT" apt-get clean
  rm -rf "$ROOT/var/lib/apt/lists/"*
fi

CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build \
  -trimpath -tags osusergo \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o "$OUT/kingai-sentinel-$ARCH" ./cmd/kingai-sentinel

install -Dm755 "$OUT/kingai-sentinel-$ARCH" "$ROOT/usr/lib/kingai/kingai-sentinel"
install -Dm755 scripts/sentinel-intel-sync.sh "$ROOT/usr/lib/kingai/sentinel-intel-sync"
install -Dm644 configs/sentinel.json "$ROOT/etc/kingai/sentinel.json"
install -Dm640 configs/sentinel-scope.example.json "$ROOT/etc/kingai/sentinel-scope.example.json"
install -Dm644 sentinel/feeds/feeds.json "$ROOT/usr/share/kingai/sentinel/feeds/feeds.json"
install -Dm644 sentinel/packs/catalog.json "$ROOT/usr/share/kingai/sentinel/packs/catalog.json"
install -Dm644 sentinel/web/index.html "$ROOT/usr/share/kingai/sentinel/web/index.html"
install -Dm644 systemd/kingai-sentinel.service "$ROOT/usr/lib/systemd/system/kingai-sentinel.service"
install -Dm644 systemd/kingai-threat-intel.service "$ROOT/usr/lib/systemd/system/kingai-threat-intel.service"
install -Dm644 systemd/kingai-threat-intel.timer "$ROOT/usr/lib/systemd/system/kingai-threat-intel.timer"

mkdir -p "$ROOT/etc/systemd/system/multi-user.target.wants" "$ROOT/etc/systemd/system/timers.target.wants"
ln -sfn /usr/lib/systemd/system/kingai-sentinel.service "$ROOT/etc/systemd/system/multi-user.target.wants/kingai-sentinel.service"
ln -sfn /usr/lib/systemd/system/kingai-threat-intel.timer "$ROOT/etc/systemd/system/timers.target.wants/kingai-threat-intel.timer"

cat >> "$ROOT/usr/lib/os-release" <<EOF
VARIANT="Sentinel"
VARIANT_ID=sentinel
EOF
printf 'KINGAI OS Sentinel %s\nProactive Defense Intelligence · console: http://127.0.0.1:9443\nActive validation is default-deny until an authorized scope is installed.\n' "$VERSION" > "$ROOT/etc/motd"

chown -R 0:0 \
  "$ROOT/etc/kingai" \
  "$ROOT/usr/lib/kingai" \
  "$ROOT/usr/share/kingai/sentinel"
chmod 0755 "$ROOT/usr/lib/kingai/kingai-sentinel" "$ROOT/usr/lib/kingai/sentinel-intel-sync"

chroot "$ROOT" dpkg-query -W -f='${binary:Package}\t${Version}\t${Architecture}\n' | LC_ALL=C sort > "$ROOT/usr/share/kingai/legal/packages.tsv"
python3 scripts/generate-sbom.py \
  "$ROOT/usr/share/kingai/legal/packages.tsv" \
  "$ROOT/usr/share/kingai/legal/KINGAI-OS.spdx.json" \
  "$VERSION" sentinel "$ARCH"

for required in \
  usr/lib/kingai/kingai-sentinel \
  usr/lib/kingai/sentinel-intel-sync \
  etc/kingai/sentinel.json \
  usr/share/kingai/sentinel/web/index.html \
  usr/lib/systemd/system/kingai-sentinel.service \
  usr/lib/systemd/system/kingai-threat-intel.timer; do
  test -e "$ROOT/$required" || { echo "missing sentinel runtime: $required" >&2; exit 1; }
done

test -L "$ROOT/etc/systemd/system/multi-user.target.wants/kingai-sentinel.service"
test -L "$ROOT/etc/systemd/system/timers.target.wants/kingai-threat-intel.timer"
grep -q '^Environment=KINGAI_SENTINEL_BIND=127.0.0.1:9443$' "$ROOT/usr/lib/systemd/system/kingai-sentinel.service"

SIZE_MIB=$(du -sm "$ROOT" | awk '{print $1}')
if (( SIZE_MIB > 3072 )); then
  echo "Sentinel rootfs exceeds hard budget: ${SIZE_MIB} MiB > 3072 MiB" >&2
  exit 1
elif (( SIZE_MIB > 2560 )); then
  echo "warning: Sentinel rootfs exceeds warning budget: ${SIZE_MIB} MiB" >&2
fi

ARTIFACT="$OUT/KINGAI-OS-sentinel-${ARCH}-rootfs.tar.zst"
tar --numeric-owner --xattrs --acls -C "$ROOT" -I 'zstd -19 -T0' -cf "$ARTIFACT" .
sha256sum "$ARTIFACT" > "$ARTIFACT.sha256"
cp "$ROOT/usr/share/kingai/legal/KINGAI-OS.spdx.json" "$ARTIFACT.spdx.json"
echo "Built $ARTIFACT"
