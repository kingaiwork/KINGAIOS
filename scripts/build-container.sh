#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

version=$(tr -d '[:space:]' < VERSION)
image=${KINGAI_CONTAINER_IMAGE:-kingai-os}
tag=${KINGAI_CONTAINER_TAG:-$version}
platforms=${KINGAI_CONTAINER_PLATFORMS:-linux/amd64,linux/arm64}
output=${KINGAI_CONTAINER_OUTPUT:-}

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }

docker buildx version >/dev/null 2>&1 || { echo "docker buildx is required" >&2; exit 1; }

args=(buildx build --file container/Dockerfile --platform "$platforms" --tag "$image:$tag")

if [[ -n "$output" ]]; then
  mkdir -p "$(dirname "$output")"
  args+=(--output "type=oci,dest=$output")
elif [[ "$platforms" == *,* ]]; then
  args+=(--output type=cacheonly)
else
  args+=(--load)
fi

args+=(.)

echo "Building KINGAI OS Container $image:$tag for $platforms"
docker "${args[@]}"

echo "KINGAI OS Container build completed."
