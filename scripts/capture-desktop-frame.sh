#!/usr/bin/env bash
set -euo pipefail
iso=${1:?usage: capture-desktop-frame.sh ISO [OUT.png]}
out=${2:-kingai-desktop-real.png}
log=${out%.png}.log
mon=$(mktemp -u /tmp/kingai-qemu-monitor.XXXXXX.sock)
ppm=$(mktemp /tmp/kingai-desktop.XXXXXX.ppm)
cleanup(){ rm -f "$mon" "$ppm"; if [[ -n "${pid:-}" ]]; then kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; fi; }
trap cleanup EXIT
qemu-system-x86_64 -machine q35 -m 4096 -smp 4 -vga virtio \
  -cdrom "$iso" -boot d -display vnc=:1 \
  -monitor "unix:$mon,server,nowait" -serial "file:$log" \
  -no-reboot -snapshot &
pid=$!
ready=0
for _ in $(seq 1 240); do
  if grep -Fq 'KINGAI_DESKTOP_GRAPHICAL_READY' "$log" 2>/dev/null; then ready=1; break; fi
  kill -0 "$pid" 2>/dev/null || break
  sleep 1
done
if [[ $ready -ne 1 ]]; then cat "$log" 2>/dev/null || true; echo 'Desktop did not reach graphical ready state.' >&2; exit 1; fi
sleep 8
printf 'screendump %s\nquit\n' "$ppm" | socat - "UNIX-CONNECT:$mon"
wait "$pid" 2>/dev/null || true
pid=
if command -v magick >/dev/null 2>&1; then magick "$ppm" "$out"; else convert "$ppm" "$out"; fi
test -s "$out"
identify "$out" || true
echo "Captured real KINGAI OS Desktop framebuffer: $out"
