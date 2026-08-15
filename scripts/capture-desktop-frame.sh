#!/usr/bin/env bash
set -euo pipefail

iso=${1:?usage: capture-desktop-frame.sh ISO [OUT.png]}
out=${2:-kingai-desktop-real.png}
log=${out%.png}.log
mon=$(mktemp -u /tmp/kingai-qemu-monitor.XXXXXX.sock)
ppm=$(mktemp /tmp/kingai-desktop.XXXXXX.ppm)

cleanup() {
  rm -f "$mon" "$ppm"
  if [[ -n "${pid:-}" ]]; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

qemu-system-x86_64 \
  -machine q35 \
  -m 4096 \
  -smp 4 \
  -device virtio-vga \
  -cdrom "$iso" \
  -boot d \
  -display vnc=:1 \
  -monitor "unix:$mon,server,nowait" \
  -serial "file:$log" \
  -no-reboot \
  -snapshot &
pid=$!

ready=0
for _ in $(seq 1 300); do
  if grep -Fq 'KINGAI_DESKTOP_SESSION_READY' "$log" 2>/dev/null; then
    ready=1
    break
  fi
  kill -0 "$pid" 2>/dev/null || break
  sleep 1
done

if [[ $ready -ne 1 ]]; then
  cat "$log" 2>/dev/null || true
  echo 'Desktop did not reach a real Plasma session.' >&2
  exit 1
fi

capture_is_usable() {
  local image=$1 metrics colors entropy mean
  metrics=$(identify -format '%k %[entropy] %[fx:mean]' "$image" 2>/dev/null || true)
  read -r colors entropy mean <<<"$metrics"
  [[ -n "${colors:-}" && -n "${entropy:-}" && -n "${mean:-}" ]] || return 1
  python3 - "$colors" "$entropy" "$mean" <<'PY'
import sys
colors = int(float(sys.argv[1]))
entropy = float(sys.argv[2])
mean = float(sys.argv[3])
# Reject near-empty / black framebuffer captures. A valid desktop or greeter
# has meaningful chromatic/tonal variation even with a restrained wallpaper.
if colors < 24 or entropy < 0.018 or mean < 0.015:
    raise SystemExit(1)
PY
}

usable=0
for attempt in $(seq 1 12); do
  rm -f "$ppm" "$out"
  printf 'screendump %s\n' "$ppm" | socat - "UNIX-CONNECT:$mon"
  sleep 1
  if [[ -s "$ppm" ]]; then
    if command -v magick >/dev/null 2>&1; then
      magick "$ppm" "$out"
    else
      convert "$ppm" "$out"
    fi
  fi
  if [[ -s "$out" ]] && capture_is_usable "$out"; then
    usable=1
    break
  fi
  echo "Desktop capture attempt $attempt was visually empty; retrying." >&2
  sleep 2
done

printf 'quit\n' | socat - "UNIX-CONNECT:$mon" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=

if [[ $usable -ne 1 ]]; then
  identify "$out" 2>/dev/null || true
  cat "$log" 2>/dev/null || true
  echo 'Refusing to publish an empty or unusable KINGAI OS Desktop capture.' >&2
  exit 1
fi

identify "$out" || true
echo "Captured verified real KINGAI OS Desktop framebuffer: $out"
