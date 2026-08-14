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

fail() {
  echo "KINGAI OS Container smoke failure: $*" >&2
  docker ps -a --filter "name=^/${name}$" >&2 || true
  docker logs "$name" >&2 2>/dev/null || true
  exit 1
}

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
[[ "$image_user" == "_kingai:kingai" ]] || fail "expected image user _kingai:kingai, got $image_user"

exposed_ports=$(docker image inspect "$image" --format '{{json .Config.ExposedPorts}}')
[[ "$exposed_ports" == "null" ]] || fail "container must not expose management TCP ports: $exposed_ports"

entrypoint=$(docker image inspect "$image" --format '{{json .Config.Entrypoint}}')
[[ "$entrypoint" == '["/usr/lib/kingai/kingaid"]' ]] || fail "unexpected image entrypoint: $entrypoint"

docker volume create "$state_volume" >/dev/null
docker volume create "$log_volume" >/dev/null

start_container() {
  local platform_args=()
  if [[ -n "$platform" ]]; then
    platform_args=(--platform "$platform")
  fi

  docker run -d \
    "${platform_args[@]}" \
    --name "$name" \
    --read-only \
    --security-opt no-new-privileges=true \
    --cap-drop ALL \
    --pids-limit 256 \
    --tmpfs /run/kingai:rw,nosuid,nodev,noexec,mode=0750,uid=10001,gid=10001 \
    --tmpfs /tmp:rw,nosuid,nodev,noexec,mode=1777 \
    --mount "type=volume,src=$state_volume,dst=/var/lib/kingai" \
    --mount "type=volume,src=$log_volume,dst=/var/log/kingai" \
    "$image" >/dev/null
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

verify_runtime_contract() {
  local runtime_uid runtime_gid runtime_path running tcp4_lines tcp6_lines machine

  runtime_uid=$(docker exec "$name" id -u) || fail "could not read runtime UID"
  [[ "$runtime_uid" == "10001" ]] || fail "expected runtime UID 10001, got $runtime_uid"

  runtime_gid=$(docker exec "$name" id -g) || fail "could not read runtime GID"
  [[ "$runtime_gid" == "10001" ]] || fail "expected runtime GID 10001, got $runtime_gid"

  # Docker's runtime Path is architecture-independent. /proc/1/exe can resolve to
  # the QEMU/binfmt emulator when an arm64 image is exercised on an amd64 runner.
  runtime_path=$(docker inspect "$name" --format '{{.Path}}') || fail "could not inspect runtime PID 1 path"
  [[ "$runtime_path" == "/usr/lib/kingai/kingaid" ]] || fail "expected PID 1 runtime path /usr/lib/kingai/kingaid, got $runtime_path"

  running=$(docker inspect "$name" --format '{{.State.Running}}') || fail "could not inspect running state"
  [[ "$running" == "true" ]] || fail "container is not running"

  tcp4_lines=$(docker exec "$name" sh -c 'wc -l < /proc/net/tcp') || fail "could not inspect IPv4 TCP listeners"
  tcp6_lines=$(docker exec "$name" sh -c 'wc -l < /proc/net/tcp6') || fail "could not inspect IPv6 TCP listeners"
  [[ "$tcp4_lines" -eq 1 ]] || fail "unexpected IPv4 TCP socket entries: $tcp4_lines"
  [[ "$tcp6_lines" -eq 1 ]] || fail "unexpected IPv6 TCP socket entries: $tcp6_lines"

  if [[ -n "$platform" ]]; then
    machine=$(docker exec "$name" uname -m) || fail "could not inspect runtime architecture"
    case "$platform" in
      linux/amd64) [[ "$machine" == "x86_64" ]] || fail "expected x86_64 runtime, got $machine" ;;
      linux/arm64) [[ "$machine" == "aarch64" || "$machine" == "arm64" ]] || fail "expected arm64 runtime, got $machine" ;;
      *) echo "unsupported smoke-test platform: $platform" >&2; exit 2 ;;
    esac
  fi
}

start_container
wait_healthy || fail "container failed health check"
verify_runtime_contract

docker exec "$name" /usr/bin/kingai status --json > "$status_file" || fail "kingai status failed"
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

docker exec "$name" /usr/bin/kingai memory put smoke '{"marker":"container-persistence-smoke"}' >/dev/null || fail "memory write failed"
docker exec "$name" /usr/bin/kingai task create main container-persistence-task >/dev/null || fail "task creation failed"

docker rm -f "$name" >/dev/null
start_container
wait_healthy || fail "recreated container failed health check"
verify_runtime_contract

memory_result=$(docker exec "$name" /usr/bin/kingai memory search container-persistence-smoke) || fail "memory search failed after recreation"
grep -q container-persistence-smoke <<<"$memory_result" || fail "memory persistence marker missing after recreation"

task_result=$(docker exec "$name" /usr/bin/kingai task list) || fail "task list failed after recreation"
grep -q container-persistence-task <<<"$task_result" || fail "task persistence marker missing after recreation"

echo "KINGAI OS Container smoke test passed${platform:+ for $platform}."
