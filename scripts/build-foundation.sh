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
  echo "building KINGAI D5 core $version for linux/$goarch"

  local cmd pkg
  for cmd in kingai kingaid kingai-execd kingai-update kingai-installer kingai-recovery; do
    case "$cmd" in
      kingai) pkg=./cmd/kingai ;;
      kingaid) pkg=./cmd/kingaid ;;
      kingai-execd) pkg=./cmd/kingai-execd ;;
      kingai-update) pkg=./cmd/kingai-update ;;
      kingai-installer) pkg=./cmd/kingai-installer ;;
      kingai-recovery) pkg=./cmd/kingai-recovery ;;
    esac
    CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
      go build -trimpath -tags osusergo \
        -ldflags "-s -w -buildid= -X main.version=$version" \
        -o "$dir/$cmd" "$pkg"
    sha256sum "$dir/$cmd" > "$dir/$cmd.sha256"
  done

  cat > "$dir/build-metadata.txt" <<META
product=KINGAI OS
version=$version
channel=dev
platform=linux/$goarch
architecture=D5 Alpha Runtime Foundation
source_commit=${GITHUB_SHA:-local}
core_binaries=kingai,kingaid,kingai-execd,kingai-update,kingai-installer,kingai-recovery
platform_profiles=server,desktop,iot,container
META
}

build_one amd64
build_one arm64

echo "foundation outputs: $out"
find "$out" -maxdepth 2 -type f -print | sort
