#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

version=$(tr -d '[:space:]' < VERSION)
out=${OUT_DIR:-out/foundation}
mkdir -p "$out"

build_one() {
  local arch=$1
  local goarch
  case "$arch" in
    amd64) goarch=amd64 ;;
    arm64) goarch=arm64 ;;
    *) echo "unsupported arch: $arch" >&2; exit 2 ;;
  esac

  local dir="$out/linux-$arch"
  mkdir -p "$dir"
  echo "building KINGAI core $version for linux/$goarch"
  CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
    go build -trimpath \
      -ldflags "-s -w -buildid= -X main.version=$version" \
      -o "$dir/kingai" ./cmd/kingai

  sha256sum "$dir/kingai" > "$dir/kingai.sha256"
  cat > "$dir/build-metadata.txt" <<META
product=KINGAI OS
version=$version
channel=dev
platform=linux/$goarch
architecture=D4 Sovereign Distributed Intelligence
source_commit=${GITHUB_SHA:-local}
META
}

build_one amd64
build_one arm64

echo "foundation outputs: $out"
find "$out" -maxdepth 2 -type f -print | sort
