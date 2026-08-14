# KINGAI OS Edge Provisioning

Provisioning creates the local trust facts used by Device Pack loading, Fleet policy and update targeting. It must be performed by a trusted factory, administrator or Fleet enrollment workflow; it is not a self-registration API for an untrusted agent.

## Device identity

Provision `/etc/kingai/device.json` as a root-owned regular file that is not group/world writable. Schema: `iot/device-identity.schema.json`.

Required facts:

- `device_id`: stable per-device identifier. Do not use a mutable hostname as the security identity.
- `board_id`: exact canonical board identifier used by Device Pack binding.
- `class`: gateway, robot, industrial-pc, embedded, appliance or developer.
- `update_channel`: stable, beta, dev or pinned.
- `provisioning`: factory, manual or fleet.
- `attestation`: none, tpm2, secure-element or tee. This records the expected mode; it does not by itself prove attestation.

Optional Fleet/labels are metadata and must not override central Policy.

## Device Pack trust

Public signing keys are provisioned under `/etc/kingai/trust/device-pack-keys/<key-id>.pub` as base64-encoded Ed25519 public keys. Keys and the directory must be root-owned and not group/world writable.

Private signing keys never belong in the device image, Device Pack, repository, logs or Fleet metadata.

## Install layout

- Manifests: `/etc/kingai/device-packs/<pack>.json`
- Detached signatures: `/etc/kingai/device-packs/<pack>.json.sig`
- Immutable artifacts: `/usr/lib/kingai/device-packs/<pack-id>/<artifact>`
- Runtime handler sockets: `/run/kingai-device/<handler>.sock`
- Device identity: `/etc/kingai/device.json`

At `kingaid` startup every release Device Pack must pass manifest schema checks, detached Ed25519 verification, exact artifact size/SHA-256 verification, architecture/board binding and runtime policy registration. Failure is fail-closed.

## Attestation boundary

`attestation` in the identity and OTA manifest is a policy selector, not an attestation proof. TPM2/secure-element/TEE quote generation, verifier infrastructure, enrollment certificates and key rotation require a separate device-specific implementation and HIL/security review before they are marketed as active remote attestation.

## Re-provisioning

Changing `board_id`, Fleet, update channel or trust keys is an administrative trust-root operation. It should be performed out of band or through a dedicated privileged service with explicit owner approval and audit; do not expose it as a generic `device.*` handler.
