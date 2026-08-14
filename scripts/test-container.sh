#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

version=$(tr -d '[:space:]' < VERSION)
image=${KINGAI_CONTAINER_TEST_IMAGE:-kingai-os:smoke}
platform=${KINGAI_CONTAINER_TEST_PLATFORM:-}
name="kingai-container-smoke-$$"
state_volume="kingai-container-state-$$"
log_volume="kingai-container-logs-$$"
status_file=$(mktemp)

cleanup() {
  docker rm -f "$name" >/dev/null 2>&1 || true
  docker volume rm -f "$state_volume" "$log_volume" >/dev/null 2>&1 || true
  rm -f "$status_file"
}
trap cleanup EXIT

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 1; }

revision=$(git rev-parse HEAD 2>/dev/null || printf unknown)
build_args=(
  --file container/Dockerfile
  --build-arg "VERSION=$version"
  --build-arg "REVISION=$revision"
  --tag "$image"
)
if [[ -n "$platform" ]]; then
  docker buildx version >/dev/null 2>&1 || { echo "docker buildx is required for platform smoke tests" >&2; exit 1; }
  docker buildx build --platform "$platform" --load "${build_args[@]}" .
else
  docker build "${build_args[@]}" .
fi

image_user=$(docker image inspect "$image" --format '{{.Config.User}}')
[[ "$image_user" == "_kingai:kingai" ]] || { echo "expected image user _kingai:kingai, got $image_user" >&2; exit 1; }

exposed_ports=$(docker image inspect "$image" --format '{{json .Config.ExposedPorts}}')
[[ "$exposed_ports" == "null" ]] || { echo "container must not expose management TCP ports: $exposed_ports" >&2; exit 1; }

entrypoint=$(docker image inspect "$image" --format '{{json .Config.Entrypoint}}')
[[ "$entrypoint" == '["/usr/lib/kingai/kingaid"]' ]] || { echo "unexpected entrypoint: $entrypoint" >&2; exit 1; }

docker volume create "$state_volume" >/dev/null
docker volume create "$log_volume" >/dev/null

start_container() {
  local run_args=(
    -d
    --name "$name"
    --read-only
    --security-opt no-new-privileges=true
    --cap-drop ALL
    --pids-limit 256
    --tmpfs /run/kingai:rw,nosuid,nodev,noexec,mode=0750,uid=10001,gid=10001
    --tmpfs /tmp:rw,nosuid,nodev,noexec,mode=1777
    --mount "type=volume,src=$state_volume,dst=/var/lib/kingai"
    --mount "type=volume,src=$log_volume,dst=/var/log/kingai"
  )
  if [[ -n "$platform" ]]; then
    run_args+=(--platform "$platform")
  fi
  docker run "${run_args[@]}" "$image" >/dev/null
}

wait_healthy() {
  local state
  for _ in $(seq 1 45); do
    state=$(docker inspect "$name" --format '{{.State.Health.Status}}')
    case "$state" in
      healthy) return 0 ;;
      unhealthy)
        docker logs "$name" >&2 || true
        return 1
        ;;
    esac
    sleep 1
  done
  docker logs "$name" >&2 || true
  echo "container did not become healthy" >&2
  return 1
}

start_container
wait_healthy

[[ "$(docker exec "$name" id -u)" == "10001" ]] || { echo "container UID is not 10001" >&2; exit 1; }
[[ "$(docker exec "$name" id -g)" == "10001" ]] || { echo "container GID is not 10001" >&2; exit 1; }
# Under cross-architecture binfmt/QEMU emulation /proc/1/exe may resolve to the
# host-side QEMU interpreter. Verify the configured entrypoint above and the
# PID 1 command line here so the smoke test checks the container contract
# rather than a host emulation implementation detail.
pid1_cmdline=$(docker exec "$name" sh -eu -c 'tr "\000" "\n" < /proc/1/cmdline')
grep -Fxq '/usr/lib/kingai/kingaid' <<<"$pid1_cmdline" || {
  echo "PID 1 command line does not contain /usr/lib/kingai/kingaid: $pid1_cmdline" >&2
  exit 1
}
docker exec "$name" sh -eu -c 'test "$(wc -l < /proc/net/tcp)" -eq 1; test "$(wc -l < /proc/net/tcp6)" -eq 1'

if [[ -n "$platform" ]]; then
  machine=$(docker exec "$name" uname -m)
  case "$platform" in
    linux/amd64) [[ "$machine" == "x86_64" ]] || { echo "expected x86_64 runtime, got $machine" >&2; exit 1; } ;;
    linux/arm64) [[ "$machine" == "aarch64" || "$machine" == "arm64" ]] || { echo "expected arm64 runtime, got $machine" >&2; exit 1; } ;;
    *) echo "unsupported smoke-test platform: $platform" >&2; exit 2 ;;
  esac
fi

docker exec "$name" /usr/bin/kingai status --json > "$status_file"
python3 - "$status_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    status = json.load(fh)

expected = {
    "policy": "enabled",
    "approval_broker": "enabled",
    "task_graph": "enabled",
    "memory_service": "enabled",
    "model_router": "enabled",
    "audit": "enabled",
}
for key, value in expected.items():
    if status.get(key) != value:
        raise SystemExit(f"{key}: expected {value!r}, got {status.get(key)!r}")
PY

docker exec "$name" /usr/bin/kingai memory put smoke '{"marker":"container-persistence-smoke"}' >/dev/null
docker exec "$name" /usr/bin/kingai task create main container-persistence-task >/dev/null

docker rm -f "$name" >/dev/null
start_container
wait_healthy

docker exec "$name" /usr/bin/kingai memory search container-persistence-smoke | grep -q container-persistence-smoke
docker exec "$name" /usr/bin/kingai task list | grep -q container-persistence-task

echo "KINGAI OS Container smoke test passed${platform:+ for $platform}."
