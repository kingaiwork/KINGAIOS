#!/usr/bin/env bash
set -euo pipefail
# Fail-closed live-media installer; dedicated VM gates verify this exact entrypoint.

PROFILE=server
TARGET=""
SOURCE_IMAGE=""
STATE_KEY=""
CONFIRM=""
LIST_ONLY=0
USERNAME=""
HOSTNAME_VALUE="kingai-pc"
USER_PASSWORD_FILE=""
LOCALE_VALUE="en_US.UTF-8"
TIMEZONE_VALUE="UTC"

usage() {
  cat >&2 <<'EOF'
usage: kingai-install [--list] --target /dev/DEVICE [--profile server|desktop]
                      [--source-image /path/to/filesystem.squashfs]
                      [--state-key /path/to/private-key]
                      [--confirm ERASE:/dev/DEVICE]
                      [--username NAME] [--hostname NAME]
                      [--user-password-file /path/to/private-password-file]
                      [--locale en_US.UTF-8] [--timezone UTC]

Installs KINGAI OS from the read-only squashfs embedded in the live ISO.
The target disk is erased and provisioned with EFI + A/B roots + encrypted STATE.
Desktop installs additionally provision the first local human account and place
/home on encrypted STATE so personal files survive A/B system-slot changes.

Without --state-key, a STATE unlock passphrase of at least 32 characters is
requested securely from /dev/tty. Desktop account passwords are also requested
securely when --user-password-file is omitted. Password values are never placed
on the command line.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --list) LIST_ONLY=1; shift ;;
    --target) [[ $# -ge 2 ]] || { usage; exit 2; }; TARGET="$2"; shift 2 ;;
    --profile) [[ $# -ge 2 ]] || { usage; exit 2; }; PROFILE="$2"; shift 2 ;;
    --source-image) [[ $# -ge 2 ]] || { usage; exit 2; }; SOURCE_IMAGE="$2"; shift 2 ;;
    --state-key) [[ $# -ge 2 ]] || { usage; exit 2; }; STATE_KEY="$2"; shift 2 ;;
    --confirm) [[ $# -ge 2 ]] || { usage; exit 2; }; CONFIRM="$2"; shift 2 ;;
    --username) [[ $# -ge 2 ]] || { usage; exit 2; }; USERNAME="$2"; shift 2 ;;
    --hostname) [[ $# -ge 2 ]] || { usage; exit 2; }; HOSTNAME_VALUE="$2"; shift 2 ;;
    --user-password-file) [[ $# -ge 2 ]] || { usage; exit 2; }; USER_PASSWORD_FILE="$2"; shift 2 ;;
    --locale) [[ $# -ge 2 ]] || { usage; exit 2; }; LOCALE_VALUE="$2"; shift 2 ;;
    --timezone) [[ $# -ge 2 ]] || { usage; exit 2; }; TIMEZONE_VALUE="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

INSTALLER_BIN="${KINGAI_INSTALLER_BIN:-/usr/lib/kingai/kingai-installer}"
[[ -x "$INSTALLER_BIN" ]] || { echo "KINGAI installer runtime is missing: $INSTALLER_BIN" >&2; exit 1; }

if (( LIST_ONLY )); then
  exec "$INSTALLER_BIN" list
fi

[[ $(id -u) -eq 0 ]] || { echo "kingai-install must run as root (use sudo)." >&2; exit 1; }
case "$PROFILE" in server|desktop) ;; *) echo "profile must be server or desktop" >&2; exit 2;; esac
[[ -n "$TARGET" ]] || { echo "--target is required" >&2; usage; exit 2; }

# Production use is intentionally limited to an actual Casper live session.
# CI may bypass this only for disposable /dev/nbd* devices.
if [[ "${KINGAI_INSTALLER_LIVE_CI:-0}" == "1" ]]; then
  [[ "$TARGET" == /dev/nbd* ]] || { echo "live-installer CI bypass is restricted to /dev/nbd*" >&2; exit 1; }
else
  grep -Eq '(^| )boot=casper( |$)' /proc/cmdline || {
    echo "kingai-install is only enabled from a KINGAI OS live session." >&2
    exit 1
  }
fi

if [[ -z "$SOURCE_IMAGE" ]]; then
  for candidate in \
    /cdrom/casper/filesystem.squashfs \
    /run/live/medium/casper/filesystem.squashfs \
    /isodevice/casper/filesystem.squashfs; do
    if [[ -f "$candidate" ]]; then SOURCE_IMAGE="$candidate"; break; fi
  done
fi
[[ -f "$SOURCE_IMAGE" ]] || {
  echo "KINGAI live source image not found; use --source-image explicitly." >&2
  exit 1
}

runtime_dir=/run/kingai-installer
mkdir -p "$runtime_dir"
chmod 0700 "$runtime_dir"
source_root=$(mktemp -d "$runtime_dir/source.XXXXXX")
generated_key=""
generated_password=""
result_file=$(mktemp "$runtime_dir/result.XXXXXX")
chmod 0600 "$result_file"

cleanup() {
  set +e
  mountpoint -q "$source_root" && umount "$source_root"
  [[ -n "$generated_key" && -f "$generated_key" ]] && rm -f "$generated_key"
  [[ -n "$generated_password" && -f "$generated_password" ]] && rm -f "$generated_password"
  [[ -f "$result_file" ]] && rm -f "$result_file"
  rmdir "$source_root" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

if [[ -z "$STATE_KEY" ]]; then
  [[ -r /dev/tty ]] || { echo "interactive passphrase entry requires /dev/tty" >&2; exit 1; }
  generated_key=$(mktemp "$runtime_dir/state-key.XXXXXX")
  chmod 0600 "$generated_key"
  printf 'Create encrypted STATE unlock passphrase (32+ characters): ' >/dev/tty
  IFS= read -r -s pass1 </dev/tty
  printf '\nRepeat encrypted STATE unlock passphrase: ' >/dev/tty
  IFS= read -r -s pass2 </dev/tty
  printf '\n' >/dev/tty
  [[ "$pass1" == "$pass2" ]] || { echo "passphrases do not match" >&2; exit 1; }
  (( ${#pass1} >= 32 )) || { echo "STATE passphrase must be at least 32 characters" >&2; exit 1; }
  printf '%s' "$pass1" > "$generated_key"
  unset pass1 pass2
  STATE_KEY="$generated_key"
fi

if [[ "$PROFILE" == "desktop" ]]; then
  if [[ "${KINGAI_INSTALLER_CI:-0}" == "1" ]]; then
    [[ -n "$USERNAME" ]] || USERNAME=kingai
    [[ -n "$HOSTNAME_VALUE" ]] || HOSTNAME_VALUE=kingai-ci
    LOCALE_VALUE="${LOCALE_VALUE:-C.UTF-8}"
    TIMEZONE_VALUE="${TIMEZONE_VALUE:-UTC}"
    if [[ -z "$USER_PASSWORD_FILE" ]]; then
      generated_password=$(mktemp "$runtime_dir/user-password.XXXXXX")
      chmod 0600 "$generated_password"
      printf 'KINGAI-CI-Desktop-Only-%s' "$$-$(date +%s)" > "$generated_password"
      USER_PASSWORD_FILE="$generated_password"
    fi
  else
    [[ -r /dev/tty ]] || { echo "Desktop account setup requires /dev/tty" >&2; exit 1; }
    if [[ -z "$USERNAME" ]]; then
      printf 'Desktop username [kingai]: ' >/dev/tty
      IFS= read -r USERNAME </dev/tty
      USERNAME=${USERNAME:-kingai}
    fi
    if [[ "$HOSTNAME_VALUE" == "kingai-pc" ]]; then
      printf 'Computer name [kingai-pc]: ' >/dev/tty
      IFS= read -r host_input </dev/tty
      HOSTNAME_VALUE=${host_input:-kingai-pc}
      unset host_input
    fi
    if [[ -z "$USER_PASSWORD_FILE" ]]; then
      generated_password=$(mktemp "$runtime_dir/user-password.XXXXXX")
      chmod 0600 "$generated_password"
      printf 'Create Desktop login password (12+ characters): ' >/dev/tty
      IFS= read -r -s user_pass1 </dev/tty
      printf '\nRepeat Desktop login password: ' >/dev/tty
      IFS= read -r -s user_pass2 </dev/tty
      printf '\n' >/dev/tty
      [[ "$user_pass1" == "$user_pass2" ]] || { echo "Desktop passwords do not match" >&2; exit 1; }
      (( ${#user_pass1} >= 12 )) || { echo "Desktop password must be at least 12 characters" >&2; exit 1; }
      printf '%s' "$user_pass1" > "$generated_password"
      unset user_pass1 user_pass2
      USER_PASSWORD_FILE="$generated_password"
    fi
  fi
fi

if [[ -z "$CONFIRM" ]]; then
  printf 'This will ERASE %s. Type exactly ERASE:%s to continue: ' "$TARGET" "$TARGET" >/dev/tty
  IFS= read -r CONFIRM </dev/tty
fi
[[ "$CONFIRM" == "ERASE:$TARGET" ]] || { echo "confirmation mismatch; installation cancelled" >&2; exit 1; }

mount -t squashfs -o loop,ro "$SOURCE_IMAGE" "$source_root"
grep -q '^NAME="KINGAI OS"' "$source_root/usr/lib/os-release" || {
  echo "embedded source is not a KINGAI OS filesystem" >&2
  exit 1
}

echo "Installing KINGAI OS $PROFILE from verified live payload onto $TARGET ..." >&2
if ! KINGAI_INSTALLER_ALLOW_WRITE=1 "$INSTALLER_BIN" execute \
  --target "$TARGET" \
  --profile "$PROFILE" \
  --source-root "$source_root" \
  --state-key "$STATE_KEY" \
  --confirm "$CONFIRM" > "$result_file"; then
  echo "KINGAI OS disk installation failed." >&2
  exit 1
fi

if [[ "$PROFILE" == "desktop" ]]; then
  configurator=/usr/lib/kingai/kingai-configure-desktop-install
  [[ -r "$configurator" ]] || { echo "Desktop post-install configurator is missing: $configurator" >&2; exit 1; }
  bash "$configurator" \
    --target "$TARGET" \
    --state-key "$STATE_KEY" \
    --username "$USERNAME" \
    --hostname "$HOSTNAME_VALUE" \
    --user-password-file "$USER_PASSWORD_FILE" \
    --locale "$LOCALE_VALUE" \
    --timezone "$TIMEZONE_VALUE"
fi

cat "$result_file"
echo "KINGAI OS installation completed. Remove the live media and reboot." >&2
