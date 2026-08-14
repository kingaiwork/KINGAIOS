#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-server}"
ARCH="${2:-amd64}"
OUT="${3:-dist}"
SUITE="${KINGAI_UBUNTU_SUITE:-resolute}"
ROOT="${OUT}/rootfs-${PROFILE}-${ARCH}"
PKG_FILE="distro/packages/${PROFILE}.txt"
VERSION="$(tr -d '[:space:]' < VERSION)"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(git log -1 --format=%ct 2>/dev/null || echo 0)}"
export SOURCE_DATE_EPOCH

case "$PROFILE" in
  server|desktop|iot|recovery) ;;
  *) echo "invalid profile: $PROFILE" >&2; exit 2 ;;
esac
case "$ARCH" in
  amd64|arm64) ;;
  *) echo "invalid arch: $ARCH" >&2; exit 2 ;;
esac
if [[ "$PROFILE" == "recovery" && "$ARCH" != "amd64" ]]; then
  echo "recovery profile is currently reviewed only for amd64" >&2
  exit 2
fi

command -v mmdebstrap >/dev/null || { echo "mmdebstrap is required" >&2; exit 1; }
command -v go >/dev/null || { echo "go is required" >&2; exit 1; }
[[ -f "$PKG_FILE" ]] || { echo "missing $PKG_FILE" >&2; exit 1; }

mkdir -p "$OUT"
rm -rf "$ROOT"
PACKAGES=$(grep -Ev '^\s*(#|$)' "$PKG_FILE" | paste -sd, -)

if [[ "$ARCH" == "amd64" && ( "$PROFILE" == "server" || "$PROFILE" == "desktop" ) ]]; then
  for extra in distro/packages/installer-common.txt distro/packages/installer-amd64.txt; do
    [[ -f "$extra" ]] || { echo "missing $extra" >&2; exit 1; }
    extra_pkgs=$(grep -Ev '^\s*(#|$)' "$extra" | paste -sd, -)
    [[ -n "$extra_pkgs" ]] && PACKAGES="${PACKAGES},${extra_pkgs}"
  done
fi

if [[ "$ARCH" == "arm64" ]]; then
  MIRRORS=(
    "deb http://ports.ubuntu.com/ubuntu-ports ${SUITE} main universe restricted multiverse"
    "deb http://ports.ubuntu.com/ubuntu-ports ${SUITE}-updates main universe restricted multiverse"
    "deb http://ports.ubuntu.com/ubuntu-ports ${SUITE}-security main universe restricted multiverse"
  )
else
  MIRRORS=(
    "deb http://archive.ubuntu.com/ubuntu ${SUITE} main universe restricted multiverse"
    "deb http://archive.ubuntu.com/ubuntu ${SUITE}-updates main universe restricted multiverse"
    "deb http://security.ubuntu.com/ubuntu ${SUITE}-security main universe restricted multiverse"
  )
fi

mmdebstrap \
  --variant=minbase \
  --architectures="$ARCH" \
  --aptopt='APT::Install-Recommends "false"' \
  --include="$PACKAGES" \
  "$SUITE" "$ROOT" "${MIRRORS[@]}"
chown 0:0 "$ROOT"

build_binary() {
  local cmd="$1" pkg
  case "$cmd" in
    kingai) pkg=./cmd/kingai ;;
    kingaid) pkg=./cmd/kingaid ;;
    kingai-execd) pkg=./cmd/kingai-execd ;;
    kingai-update) pkg=./cmd/kingai-update ;;
    kingai-installer) pkg=./cmd/kingai-installer ;;
    kingai-recovery) pkg=./cmd/kingai-recovery ;;
    *) echo "unknown KINGAI binary: $cmd" >&2; exit 2 ;;
  esac
  CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build \
    -trimpath -tags osusergo \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o "$OUT/$cmd-$ARCH" "$pkg"
}

case "$PROFILE" in
  server|desktop)
    KINGAI_BINARIES=(kingai kingaid kingai-execd kingai-update kingai-installer kingai-recovery)
    ;;
  iot)
    # Edge keeps the governed non-root Runtime but does not ship the generic
    # privileged execution broker. Device-specific privileged operations must
    # arrive later through explicit Device Pack capability handlers.
    KINGAI_BINARIES=(kingai kingaid kingai-update)
    ;;
  recovery)
    KINGAI_BINARIES=(kingai kingai-update kingai-recovery)
    ;;
esac

for cmd in "${KINGAI_BINARIES[@]}"; do
  build_binary "$cmd"
done

install -Dm755 "$OUT/kingai-$ARCH" "$ROOT/usr/bin/kingai"
if [[ "$PROFILE" != "recovery" ]]; then
  install -Dm755 "$OUT/kingaid-$ARCH" "$ROOT/usr/lib/kingai/kingaid"
fi
install -Dm755 "$OUT/kingai-update-$ARCH" "$ROOT/usr/lib/kingai/kingai-update"
ln -sfn /usr/lib/kingai/kingai-update "$ROOT/usr/bin/kingai-update"

if [[ "$PROFILE" == "server" || "$PROFILE" == "desktop" ]]; then
  install -Dm755 "$OUT/kingai-execd-$ARCH" "$ROOT/usr/lib/kingai/kingai-execd"
  install -Dm755 "$OUT/kingai-installer-$ARCH" "$ROOT/usr/lib/kingai/kingai-installer"
  install -Dm755 "$OUT/kingai-recovery-$ARCH" "$ROOT/usr/lib/kingai/kingai-recovery"
  ln -sfn /usr/lib/kingai/kingai-installer "$ROOT/usr/bin/kingai-installer"
  ln -sfn /usr/lib/kingai/kingai-recovery "$ROOT/usr/bin/kingai-recovery"
fi
if [[ "$PROFILE" == "recovery" ]]; then
  install -Dm755 "$OUT/kingai-recovery-$ARCH" "$ROOT/usr/lib/kingai/kingai-recovery"
  ln -sfn /usr/lib/kingai/kingai-recovery "$ROOT/usr/bin/kingai-recovery"
fi

if [[ "$PROFILE" != "recovery" ]]; then
  install -Dm644 systemd/kingaid.service "$ROOT/usr/lib/systemd/system/kingaid.service"
  install -Dm644 systemd/kingai-update-health.service "$ROOT/usr/lib/systemd/system/kingai-update-health.service"
fi
if [[ "$PROFILE" == "server" || "$PROFILE" == "desktop" ]]; then
  install -Dm644 systemd/kingai-execd.service "$ROOT/usr/lib/systemd/system/kingai-execd.service"
  install -Dm644 systemd/kingaid-execd.conf "$ROOT/usr/lib/systemd/system/kingaid.service.d/10-execd.conf"
fi
if [[ "$PROFILE" == "iot" ]]; then
  install -Dm644 systemd/kingaid-iot.conf "$ROOT/usr/lib/systemd/system/kingaid.service.d/10-iot.conf"
fi

install -Dm644 sysusers/kingai.conf "$ROOT/usr/lib/sysusers.d/kingai.conf"
install -Dm644 configs/policy.json "$ROOT/etc/kingai/policy.json"
install -Dm644 configs/system.json "$ROOT/etc/kingai/system.json"
install -Dm644 configs/models.json "$ROOT/etc/kingai/models.json"
install -Dm644 configs/agents.json "$ROOT/etc/kingai/agents.json"

cp -a --no-preserve=ownership distro/overlay/. "$ROOT/"
rm -f "$ROOT/etc/os-release" "$ROOT/usr/lib/os-release"
install -Dm644 distro/overlay/etc/os-release "$ROOT/usr/lib/os-release"
ln -s ../usr/lib/os-release "$ROOT/etc/os-release"
mkdir -p "$ROOT/etc/systemd/system/multi-user.target.wants"

if [[ "$PROFILE" == "server" || "$PROFILE" == "desktop" ]]; then
  ln -sfn /usr/lib/systemd/system/kingai-execd.service "$ROOT/etc/systemd/system/multi-user.target.wants/kingai-execd.service"
  ln -sfn /usr/lib/systemd/system/kingaid.service "$ROOT/etc/systemd/system/multi-user.target.wants/kingaid.service"
  ln -sfn /usr/lib/systemd/system/kingai-update-health.service "$ROOT/etc/systemd/system/multi-user.target.wants/kingai-update-health.service"
elif [[ "$PROFILE" == "iot" ]]; then
  ln -sfn /usr/lib/systemd/system/kingaid.service "$ROOT/etc/systemd/system/multi-user.target.wants/kingaid.service"
  ln -sfn /usr/lib/systemd/system/kingai-update-health.service "$ROOT/etc/systemd/system/multi-user.target.wants/kingai-update-health.service"
fi

if [[ "$PROFILE" == "recovery" ]]; then
  rm -f \
    "$ROOT/etc/systemd/system/multi-user.target.wants/kingai-execd.service" \
    "$ROOT/etc/systemd/system/multi-user.target.wants/kingaid.service" \
    "$ROOT/etc/systemd/system/multi-user.target.wants/kingai-update-health.service"
  printf 'KINGAI OS Recovery %s\nOffline A/B inspection, rollback and boot repair. No network trust is implied.\n' "$VERSION" > "$ROOT/etc/motd"
fi

install -d -m0755 "$ROOT/var/crash" "$ROOT/etc/xdg/autostart"
mkdir -p "$ROOT/usr/share/doc/kingai-os" "$ROOT/usr/share/kingai/legal"
install -Dm644 LICENSE "$ROOT/usr/share/doc/kingai-os/LICENSE"
install -Dm644 NOTICE "$ROOT/usr/share/doc/kingai-os/NOTICE"
install -Dm644 legal/THIRD_PARTY.md "$ROOT/usr/share/kingai/legal/THIRD_PARTY.md"
install -Dm644 legal/models.json "$ROOT/usr/share/kingai/legal/models.json"

chroot "$ROOT" dpkg-query -W -f='${binary:Package}\t${Version}\t${Architecture}\n' | LC_ALL=C sort > "$ROOT/usr/share/kingai/legal/packages.tsv"
python3 scripts/generate-sbom.py \
  "$ROOT/usr/share/kingai/legal/packages.tsv" \
  "$ROOT/usr/share/kingai/legal/KINGAI-OS.spdx.json" \
  "$VERSION" "$PROFILE" "$ARCH"

if [[ "$PROFILE" == "desktop" ]]; then
  mkdir -p \
    "$ROOT/usr/share/kingai" \
    "$ROOT/usr/share/plasma/look-and-feel" \
    "$ROOT/usr/share/plasma/plasmoids" \
    "$ROOT/etc/xdg/autostart" \
    "$ROOT/etc/systemd/system"
  cp -a --no-preserve=ownership desktop "$ROOT/usr/share/kingai/"
  cp -a --no-preserve=ownership desktop/look-and-feel/. "$ROOT/usr/share/plasma/look-and-feel/"
  cp -a --no-preserve=ownership desktop/plasmoids/org.kingai.agentcenter "$ROOT/usr/share/plasma/plasmoids/"
  install -Dm755 desktop/welcome/launch.sh "$ROOT/usr/lib/kingai/kingai-welcome"
  install -Dm644 desktop/welcome/kingai-welcome.desktop "$ROOT/etc/xdg/autostart/kingai-welcome.desktop"
  test -f "$ROOT/usr/lib/systemd/system/sddm.service" || { echo "sddm.service missing from desktop rootfs" >&2; exit 1; }
  ln -sfn /usr/lib/systemd/system/graphical.target "$ROOT/etc/systemd/system/default.target"
  ln -sfn /usr/lib/systemd/system/sddm.service "$ROOT/etc/systemd/system/display-manager.service"
fi

chown -R 0:0 "$ROOT/etc/kingai" "$ROOT/usr/lib/kingai" "$ROOT/usr/share/kingai" "$ROOT/usr/share/doc/kingai-os"
if [[ "$PROFILE" == "desktop" ]]; then
  chown -R 0:0 "$ROOT/usr/share/plasma/plasmoids/org.kingai.agentcenter" "$ROOT/usr/share/plasma/look-and-feel/org.kingai."*
fi
chown 0:0 "$ROOT/usr/lib/os-release" "$ROOT/etc/motd" "$ROOT/etc/issue" "$ROOT/etc/issue.net" "$ROOT" 2>/dev/null || true

if [[ "$PROFILE" != "iot" ]]; then
  test -n "$(find "$ROOT/boot" -maxdepth 1 -name 'vmlinuz-*' -print -quit)" || { echo "kernel missing from rootfs" >&2; exit 1; }
  test -n "$(find "$ROOT/boot" -maxdepth 1 -name 'initrd.img-*' -print -quit)" || { echo "initramfs missing from rootfs" >&2; exit 1; }
fi

if [[ "$ARCH" == "amd64" && ( "$PROFILE" == "server" || "$PROFILE" == "desktop" ) ]]; then
  for tool in sgdisk partprobe mkfs.vfat mkfs.ext4 cryptsetup rsync grub-install grub-editenv findmnt blkid systemctl; do
    chroot "$ROOT" sh -c "command -v $tool >/dev/null" || { echo "installer/update/runtime missing: $tool" >&2; exit 1; }
  done
fi
if [[ "$PROFILE" == "recovery" ]]; then
  for tool in cryptsetup mount umount grub-editenv blkid sync; do
    chroot "$ROOT" sh -c "command -v $tool >/dev/null" || { echo "recovery runtime missing: $tool" >&2; exit 1; }
  done
fi

if [[ "$PROFILE" == "iot" ]]; then
  test -x "$ROOT/usr/lib/kingai/kingaid"
  test ! -e "$ROOT/usr/lib/kingai/kingai-execd"
  test ! -e "$ROOT/usr/lib/systemd/system/kingai-execd.service"
  test ! -e "$ROOT/usr/lib/kingai/kingai-installer"
  test ! -e "$ROOT/usr/lib/kingai/kingai-recovery"
  test -L "$ROOT/etc/systemd/system/multi-user.target.wants/kingaid.service"
  grep -q '^Environment=KINGAI_REQUIRE_EXECD=false$' "$ROOT/usr/lib/systemd/system/kingaid.service.d/10-iot.conf"
  grep -q '^Environment=KINGAI_TASK_RUN_BUDGET=16$' "$ROOT/usr/lib/systemd/system/kingaid.service.d/10-iot.conf"
fi

if [[ "${KINGAI_SKIP_ARCHIVE:-0}" == "1" ]]; then
  echo "Built rootfs directory: $ROOT"
  exit 0
fi

ARTIFACT="$OUT/KINGAI-OS-${PROFILE}-${ARCH}-rootfs.tar.zst"
tar --numeric-owner --xattrs --acls -C "$ROOT" -I 'zstd -19 -T0' -cf "$ARTIFACT" .
sha256sum "$ARTIFACT" > "$ARTIFACT.sha256"
cp "$ROOT/usr/share/kingai/legal/KINGAI-OS.spdx.json" "$ARTIFACT.spdx.json"
echo "Built $ARTIFACT"
