#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

version=$(tr -d '[:space:]' < VERSION)
image=${KINGAI_CONTAINER_TEST_IMAGE:-kingai-os:smoke}
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
docker build \
  --file container/Dockerfile \
  --build-arg "VERSION=$version" \
  --build-arg "REVISION=$revision" \
  --tag "$image" \
  .

image_user=$(docker image inspect "$image" --format '{{.Config.User}}')
[[ "$image_user" == "_kingai:kingai" ]] || { echo "expected image user _kingai:kingai, got $image_user" >&2; exit 1; }

exposed_ports=$(docker image inspect "$image" --format '{{json .Config.ExposedPorts}}')
[[ "$exposed_ports" == "null" ]] || { echo "container must not expose management TCP ports: $exposed_ports" >&2; exit 1; }

entrypoint=$(docker image inspect "$image" --format '{{json .Config.Entrypoint}}')
[[ "$entrypoint" == '["/usr/lib/kingai/kingaid"]' ]] || { echo "unexpected entrypoint: $entrypoint" >&2; exit 1; }

docker volume create "$state_volume" >/dev/null
docker volume create "$log_volume" >/dev/null

start_container() {
  docker run -d \
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
  for _ in $(seq 1 30); do
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

[[ "$(docker exec "$name" id -u)" == "10001" ]]
[[ "$(docker exec "$name" id -g)" == "10001" ]]
docker exec "$name" sh -eu -c 'test "$(basename "$(readlink /proc/1/exe)")" = kingaid'
docker exec "$name" sh -eu -c 'test "$(wc -l < /proc/net/tcp)" -eq 1; test "$(wc -l < /proc/net/tcp6)" -eq 1'

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

echo "KINGAI OS Container smoke test passed."
