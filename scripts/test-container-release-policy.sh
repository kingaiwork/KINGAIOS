#!/usr/bin/env bash
set -euo pipefail

validate_release() {
  local version=$1 channel=$2
  case "$channel" in
    beta) [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+-beta\.[0-9]+$ ]] ;;
    rc) [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$ ]] ;;
    stable) [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ;;
    *) return 1 ;;
  esac
}

for pair in \
  '0.1.0-beta.1 beta' \
  '1.2.3-beta.42 beta' \
  '0.1.0-rc.1 rc' \
  '9.8.7 stable'
do
  read -r version channel <<<"$pair"
  validate_release "$version" "$channel" || {
    echo "expected valid release identity: $version / $channel" >&2
    exit 1
  }
done

for pair in \
  '0.1.0 beta' \
  '0.1.0-beta.1 stable' \
  '0.1.0-rc.1 beta' \
  '0.1.0-rc rc' \
  'v0.1.0 stable' \
  '0.1 stable' \
  '0.1.0 nightly'
do
  read -r version channel <<<"$pair"
  if validate_release "$version" "$channel"; then
    echo "expected invalid release identity: $version / $channel" >&2
    exit 1
  fi
done

python3 - <<'PY'
import json
import re
from pathlib import Path


def load(path: Path):
    if not path.is_file():
        raise AssertionError(f'missing required release policy file: {path}')
    return json.loads(path.read_text(encoding='utf-8'))


gates = load(Path('release/gates.json'))
assert gates['container_release_pipeline'] is True
assert gates['production_signing_ready'] is False

request = load(Path('.release/container-beta.json'))
assert request['profile'] == 'container'
assert request['channel'] == 'beta'
assert request['version'] == '0.1.0-beta.1'

# container_signed_release is a lifecycle state, not a permanent false gate.
# Before the first signed release it may be false. Once true, durable evidence
# must exist and prove the published artifact, architecture coverage and the
# stable fail-closed policy rather than merely trusting the boolean flag.
if gates['container_signed_release'] is True:
    evidence_path = Path('release') / f"container-beta-{request['version']}.json"
    evidence = load(evidence_path)

    assert evidence['schema'] == 1
    assert evidence['product'] == 'KINGAI OS Container'
    assert evidence['profile'] == request['profile']
    assert evidence['channel'] == request['channel']
    assert evidence['version'] == request['version']
    assert evidence['image'] == 'ghcr.io/kingaiwork/kingai-os'
    assert set(evidence['platforms']) == {'linux/amd64', 'linux/arm64'}
    assert re.fullmatch(r'sha256:[0-9a-f]{64}', evidence['digest'])
    assert evidence['release_run_id'] > 0
    assert evidence['release_artifact_id'] > 0
    assert evidence['management_transport'] == 'unix-socket-only'
    assert evidence['runtime_uid'] == 10001
    assert evidence['runtime_gid'] == 10001

    supply_chain = evidence['supply_chain']
    assert supply_chain['buildkit_sbom'] is True
    assert supply_chain['buildkit_provenance'] is True
    assert supply_chain['cyclonedx_sbom'] is True
    assert supply_chain['github_provenance_attestation'] is True
    assert supply_chain['github_sbom_attestation'] is True
    assert supply_chain['signature'] == 'sigstore-keyless-cosign'
    assert supply_chain['signature_verified'] is True

    validation = evidence['post_release_validation']
    assert validation['amd64_runtime_persistence_smoke'] is True
    assert validation['arm64_runtime_persistence_smoke'] is True
    assert validation['amd64_fixable_critical_scan'] is True
    assert validation['arm64_fixable_critical_scan'] is True
    assert validation['multiarch_oci_archive_validation'] is True

    promotion = evidence['promotion']
    assert promotion['beta_signed_release'] is True
    assert promotion['stable_release_allowed'] is False
    assert promotion['stable_block_reason'] == 'production_signing_ready=false'
elif gates['container_signed_release'] is not False:
    raise AssertionError('container_signed_release must be a boolean')
PY

echo "KINGAI OS Container release policy regression tests passed."
