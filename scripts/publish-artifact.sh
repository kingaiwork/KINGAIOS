#!/usr/bin/env bash
set -euo pipefail

# KINGAI OS artifact router
# <= 2 GiB: GitHub Release when a release tag/token is supplied.
# >= 2 GiB: Cloudflare R2 using the official Wrangler v4 remote R2 command.
#
# R2 environment:
#   CLOUDFLARE_API_TOKEN
#   CLOUDFLARE_ACCOUNT_ID
#   R2_BUCKET
# Optional: R2_PREFIX=dev
#
# GitHub environment:
#   GH_TOKEN
#   GH_RELEASE_TAG

WRANGLER_VERSION="${KINGAI_WRANGLER_VERSION:-4.94.0}"

if [[ $# -ne 1 ]]; then echo "usage: $0 <artifact>" >&2; exit 2; fi
artifact=$1
[[ -f "$artifact" ]] || { echo "artifact not found: $artifact" >&2; exit 1; }
bytes=$(stat -c '%s' "$artifact")
name=$(basename "$artifact")
version=$(tr -d '[:space:]' < VERSION 2>/dev/null || echo unknown)
sha_file="${artifact}.sha256"
[[ -f "$sha_file" ]] || sha256sum "$artifact" > "$sha_file"
limit=$((2 * 1024 * 1024 * 1024))
printf 'artifact=%s\nsize_bytes=%s\nversion=%s\n' "$name" "$bytes" "$version"

r2_put() {
  local file=$1 key=$2 content_type=$3
  npx --yes "wrangler@${WRANGLER_VERSION}" r2 object put "${R2_BUCKET}/${key}" \
    --file "$file" --content-type "$content_type" --remote
}

if (( bytes >= limit )); then
  : "${CLOUDFLARE_API_TOKEN:?CLOUDFLARE_API_TOKEN is required for >=2GiB artifacts}"
  : "${CLOUDFLARE_ACCOUNT_ID:?CLOUDFLARE_ACCOUNT_ID is required for >=2GiB artifacts}"
  : "${R2_BUCKET:?R2_BUCKET is required for >=2GiB artifacts}"
  command -v node >/dev/null 2>&1 || { echo "Node.js is required for Wrangler" >&2; exit 1; }
  command -v npx >/dev/null 2>&1 || { echo "npx is required for Wrangler" >&2; exit 1; }
  prefix=${R2_PREFIX:-dev}; key="${prefix}/${version}/${name}"
  echo "routing=cloudflare-r2"; echo "r2_key=$key"; echo "wrangler_version=$WRANGLER_VERSION"
  npx --yes "wrangler@${WRANGLER_VERSION}" --version
  r2_put "$artifact" "$key" application/octet-stream
  r2_put "$sha_file" "$key.sha256" text/plain
  [[ -f "$artifact.manifest.json" ]] && r2_put "$artifact.manifest.json" "$key.manifest.json" application/json
  [[ -f "$artifact.spdx.json" ]] && r2_put "$artifact.spdx.json" "$key.spdx.json" application/spdx+json
  echo "published=r2://${R2_BUCKET}/${key}"
  exit 0
fi

if [[ -n "${GH_RELEASE_TAG:-}" && -n "${GH_TOKEN:-}" ]]; then
  command -v gh >/dev/null 2>&1 || { echo "gh CLI is required for GitHub Release upload" >&2; exit 1; }
  assets=("$artifact" "$sha_file")
  [[ -f "$artifact.manifest.json" ]] && assets+=("$artifact.manifest.json")
  [[ -f "$artifact.spdx.json" ]] && assets+=("$artifact.spdx.json")
  echo "routing=github-release"
  gh release upload "$GH_RELEASE_TAG" "${assets[@]}" --clobber
  echo "published=github-release:${GH_RELEASE_TAG}/${name}"
  exit 0
fi

echo "routing=local-staging"
echo "Upload skipped because release credentials were not supplied."
echo "SHA256: $(cat "$sha_file")"
