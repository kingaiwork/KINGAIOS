#!/usr/bin/env bash
set -euo pipefail

# KINGAI OS artifact router
# - Files below GitHub's 2 GiB per-release-asset ceiling may be uploaded to GitHub Releases.
# - Files at/above 2 GiB are sent to Cloudflare R2 using the S3-compatible API.
#
# Required for R2:
#   R2_ACCOUNT_ID
#   R2_BUCKET
#   AWS_ACCESS_KEY_ID
#   AWS_SECRET_ACCESS_KEY
# Optional:
#   R2_PREFIX=stable
#
# Required for GitHub Release upload:
#   GH_TOKEN
#   GH_RELEASE_TAG

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <artifact>" >&2
  exit 2
fi

artifact=$1
[[ -f "$artifact" ]] || { echo "artifact not found: $artifact" >&2; exit 1; }

bytes=$(stat -c '%s' "$artifact")
name=$(basename "$artifact")
version=$(tr -d '[:space:]' < VERSION 2>/dev/null || echo unknown)
sha_file="${artifact}.sha256"
sha256sum "$artifact" > "$sha_file"

# GitHub documents a per-file Release Asset limit of <2 GiB.
# Route the boundary itself to R2 to avoid ambiguous failures.
limit=$((2 * 1024 * 1024 * 1024))

printf 'artifact=%s\nsize_bytes=%s\nversion=%s\n' "$name" "$bytes" "$version"

if (( bytes >= limit )); then
  : "${R2_ACCOUNT_ID:?R2_ACCOUNT_ID is required for >=2GiB artifacts}"
  : "${R2_BUCKET:?R2_BUCKET is required for >=2GiB artifacts}"
  : "${AWS_ACCESS_KEY_ID:?AWS_ACCESS_KEY_ID is required for R2}"
  : "${AWS_SECRET_ACCESS_KEY:?AWS_SECRET_ACCESS_KEY is required for R2}"

  command -v aws >/dev/null 2>&1 || { echo "aws CLI is required" >&2; exit 1; }

  prefix=${R2_PREFIX:-dev}
  key="${prefix}/${version}/${name}"
  endpoint="https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"

  export AWS_DEFAULT_REGION=auto
  export AWS_EC2_METADATA_DISABLED=true

  echo "routing=cloudflare-r2"
  echo "r2_key=$key"

  # aws s3 cp automatically uses multipart upload for large objects.
  aws s3 cp "$artifact" "s3://${R2_BUCKET}/${key}" \
    --endpoint-url "$endpoint" \
    --only-show-errors
  aws s3 cp "$sha_file" "s3://${R2_BUCKET}/${key}.sha256" \
    --endpoint-url "$endpoint" \
    --only-show-errors

  echo "published=r2://${R2_BUCKET}/${key}"
  exit 0
fi

if [[ -n "${GH_RELEASE_TAG:-}" && -n "${GH_TOKEN:-}" ]]; then
  command -v gh >/dev/null 2>&1 || { echo "gh CLI is required for GitHub Release upload" >&2; exit 1; }
  echo "routing=github-release"
  gh release upload "$GH_RELEASE_TAG" "$artifact" "$sha_file" --clobber
  echo "published=github-release:${GH_RELEASE_TAG}/${name}"
  exit 0
fi

echo "routing=local-staging"
echo "GitHub upload skipped because GH_RELEASE_TAG/GH_TOKEN are not set."
echo "SHA256: $(cat "$sha_file")"
