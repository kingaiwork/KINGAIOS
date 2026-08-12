#!/usr/bin/env bash
set -euo pipefail
PROFILE="${1:-server}"; ARCH="${2:-amd64}"; OUT="${3:-dist}"; SUITE="${KINGAI_UBUNTU_SUITE:-resolute}"
ROOT="${OUT}/rootfs-${PROFILE}-${ARCH}"; PKG_FILE="distro/packages/${PROFILE}.txt"; VERSION="$(tr -d '[:space:]' < VERSION)"
case "$PROFILE" in server|desktop|iot);;*)echo "invalid profile: $PROFILE" >&2;exit 2;;esac
case "$ARCH" in amd64|arm64);;*)echo "invalid arch: $ARCH" >&2;exit 2;;esac
command -v mmdebstrap >/dev/null||{ echo "mmdebstrap is required" >&2;exit 1;};command -v go >/dev/null||{ echo "go is required" >&2;exit 1;};[[ -f "$PKG_FILE" ]]||{ echo "missing $PKG_FILE" >&2;exit 1;}
mkdir -p "$OUT";rm -rf "$ROOT";PACKAGES=$(grep -Ev '^\s*(#|$)' "$PKG_FILE"|paste -sd, -)
if [[ "$ARCH" == "arm64" ]];then MIRRORS=("deb http://ports.ubuntu.com/ubuntu-ports ${SUITE} main universe restricted multiverse" "deb http://ports.ubuntu.com/ubuntu-ports ${SUITE}-updates main universe restricted multiverse" "deb http://ports.ubuntu.com/ubuntu-ports ${SUITE}-security main universe restricted multiverse");else MIRRORS=("deb http://archive.ubuntu.com/ubuntu ${SUITE} main universe restricted multiverse" "deb http://archive.ubuntu.com/ubuntu ${SUITE}-updates main universe restricted multiverse" "deb http://security.ubuntu.com/ubuntu ${SUITE}-security main universe restricted multiverse");fi
mmdebstrap --variant=minbase --architectures="$ARCH" --aptopt='APT::Install-Recommends "false"' --include="$PACKAGES" "$SUITE" "$ROOT" "${MIRRORS[@]}"
for cmd in kingai kingaid kingai-update kingai-installer;do case "$cmd" in kingai)pkg=./cmd/kingai;;kingaid)pkg=./cmd/kingaid;;kingai-update)pkg=./cmd/kingai-update;;kingai-installer)pkg=./cmd/kingai-installer;;esac;CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath -tags osusergo -ldflags "-s -w -X main.version=${VERSION}" -o "$OUT/$cmd-$ARCH" "$pkg";done
install -Dm755 "$OUT/kingai-$ARCH" "$ROOT/usr/bin/kingai";install -Dm755 "$OUT/kingaid-$ARCH" "$ROOT/usr/lib/kingai/kingaid";install -Dm755 "$OUT/kingai-update-$ARCH" "$ROOT/usr/lib/kingai/kingai-update";install -Dm755 "$OUT/kingai-installer-$ARCH" "$ROOT/usr/lib/kingai/kingai-installer"
install -Dm644 systemd/kingaid.service "$ROOT/usr/lib/systemd/system/kingaid.service";install -Dm644 sysusers/kingai.conf "$ROOT/usr/lib/sysusers.d/kingai.conf";install -Dm644 configs/policy.json "$ROOT/etc/kingai/policy.json";install -Dm644 configs/system.json "$ROOT/etc/kingai/system.json";install -Dm644 configs/models.json "$ROOT/etc/kingai/models.json";install -Dm644 configs/agents.json "$ROOT/etc/kingai/agents.json";cp -a distro/overlay/. "$ROOT/"
rm -f "$ROOT/etc/os-release" "$ROOT/usr/lib/os-release";install -Dm644 distro/overlay/etc/os-release "$ROOT/usr/lib/os-release";ln -s ../usr/lib/os-release "$ROOT/etc/os-release";mkdir -p "$ROOT/etc/systemd/system/multi-user.target.wants";ln -sfn /usr/lib/systemd/system/kingaid.service "$ROOT/etc/systemd/system/multi-user.target.wants/kingaid.service"
# Casper expects these paths during live-session bottom scripts before normal tmpfiles/autostart setup.
install -d -m0755 "$ROOT/var/crash" "$ROOT/etc/xdg/autostart"
mkdir -p "$ROOT/usr/share/doc/kingai-os" "$ROOT/usr/share/kingai/legal"
install -Dm644 LICENSE "$ROOT/usr/share/doc/kingai-os/LICENSE";install -Dm644 NOTICE "$ROOT/usr/share/doc/kingai-os/NOTICE";install -Dm644 legal/THIRD_PARTY.md "$ROOT/usr/share/kingai/legal/THIRD_PARTY.md";install -Dm644 legal/models.json "$ROOT/usr/share/kingai/legal/models.json"
chroot "$ROOT" dpkg-query -W -f='${binary:Package}\t${Version}\t${Architecture}\n' | LC_ALL=C sort > "$ROOT/usr/share/kingai/legal/packages.tsv"
if [[ "$PROFILE" == "desktop" ]];then mkdir -p "$ROOT/usr/share/kingai" "$ROOT/usr/share/plasma/look-and-feel" "$ROOT/etc/xdg/autostart";cp -a desktop "$ROOT/usr/share/kingai/";cp -a desktop/look-and-feel/. "$ROOT/usr/share/plasma/look-and-feel/";install -Dm755 desktop/welcome/launch.sh "$ROOT/usr/lib/kingai/kingai-welcome";install -Dm644 desktop/welcome/kingai-welcome.desktop "$ROOT/etc/xdg/autostart/kingai-welcome.desktop";fi
# Never leak CI/build-host ownership into published root files copied from the repository.
chown -R 0:0 "$ROOT/etc/kingai" "$ROOT/usr/lib/kingai" "$ROOT/usr/share/kingai" "$ROOT/usr/share/doc/kingai-os"
chown 0:0 "$ROOT/usr/lib/os-release" "$ROOT/etc/motd" "$ROOT/etc/issue" "$ROOT/etc/issue.net" 2>/dev/null || true
if [[ "$PROFILE" != "iot" ]];then test -n "$(find "$ROOT/boot" -maxdepth 1 -name 'vmlinuz-*' -print -quit)"||{ echo "kernel missing from rootfs" >&2;exit 1;};test -n "$(find "$ROOT/boot" -maxdepth 1 -name 'initrd.img-*' -print -quit)"||{ echo "initramfs missing from rootfs" >&2;exit 1;};fi
if [[ "${KINGAI_SKIP_ARCHIVE:-0}" == "1" ]];then echo "Built rootfs directory: $ROOT";exit 0;fi
ARTIFACT="$OUT/KINGAI-OS-${PROFILE}-${ARCH}-rootfs.tar.zst";tar --numeric-owner --xattrs --acls -C "$ROOT" -I 'zstd -19 -T0' -cf "$ARTIFACT" .;sha256sum "$ARTIFACT">"$ARTIFACT.sha256";echo "Built $ARTIFACT"
