#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

version=$(tr -d '[:space:]' < VERSION)
image=${KINGAI_CONTAINER_IMAGE:-kingai-os}
tag=${KINGAI_CONTAINER_TAG:-$version}
platforms=${KINGAI_CONTAINER_PLATFORMS:-linux/amd64,linux/arm64}
output=${KINGAI_CONTAINER_OUTPUT:-}
push=${KINGAI_CONTAINER_PUSH:-0}
revision=${KINGAI_CONTAINER_REVISION:-$(git rev-parse HEAD 2>/dev/null || printf unknown)}

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
docker buildx version >/dev/null 2>&1 || { echo "docker buildx is required" >&2; exit 1; }

if [[ -n "$output" && "$push" == "1" ]]; then
  echo "KINGAI_CONTAINER_OUTPUT and KINGAI_CONTAINER_PUSH=1 are mutually exclusive" >&2
  exit 2
fi

args=(
  buildx build
  --file container/Dockerfile
  --platform "$platforms"
  --tag "$image:$tag"
  --build-arg "VERSION=$version"
  --build-arg "REVISION=$revision"
)

if [[ "$push" == "1" ]]; then
  args+=(--provenance=true --sbom=true --push)
elif [[ -n "$output" ]]; then
  mkdir -p "$(dirname "$output")"
  args+=(--output "type=oci,dest=$output")
elif [[ "$platforms" == *,* ]]; then
  output="dist/kingai-os-${tag}.oci.tar"
  mkdir -p "$(dirname "$output")"
  args+=(--output "type=oci,dest=$output")
else
  args+=(--load)
fi

args+=(.)

echo "Building KINGAI OS Container $image:$tag for $platforms"
docker "${args[@]}"

if [[ "$push" == "1" ]]; then
  echo "Published $image:$tag"
elif [[ -n "$output" ]]; then
  echo "OCI archive: $output"
else
  echo "Loaded local image: $image:$tag"
fi
