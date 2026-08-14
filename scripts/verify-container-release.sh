#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 ghcr.io/kingaiwork/kingai-os@sha256:<digest>" >&2
  exit 2
fi

image=$1
case "$image" in
  ghcr.io/kingaiwork/kingai-os@sha256:*) ;;
  *) echo "verification requires the canonical KINGAI OS image pinned by sha256 digest" >&2; exit 2 ;;
esac

command -v cosign >/dev/null 2>&1 || { echo "cosign is required" >&2; exit 1; }

identity='^https://github\.com/kingaiwork/KINGAIOS/\.github/workflows/container-release\.yml@refs/(heads/main|tags/container-v[0-9]+\.[0-9]+\.[0-9]+(-(beta|rc)\.[0-9]+)?)$'
issuer='https://token.actions.githubusercontent.com'

cosign verify \
  --certificate-identity-regexp "$identity" \
  --certificate-oidc-issuer "$issuer" \
  "$image"

echo "KINGAI OS Container signature verified: $image"
