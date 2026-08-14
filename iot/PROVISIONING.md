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

A Device Pack release is produced off-device with `tools/device-pack-release`. The release tool requires an explicit redistribution-review acknowledgement and immutable `--review-ref`; it does not silently convert a template into an approved vendor redistribution.

## Install layout

- Manifests: `/etc/kingai/device-packs/<pack-id>.json`
- Detached signatures: `/etc/kingai/device-packs/<pack-id>.json.sig`
- Immutable artifacts: `/usr/lib/kingai/device-packs/<pack-id>/<artifact>`
- Runtime handler sockets: `/run/kingai-device/<handler>.sock`
- Device identity: `/etc/kingai/device.json`
- Root lifecycle audit: `/var/log/kingai/device-pack-admin.jsonl`

At `kingaid` startup every release Device Pack must pass manifest schema checks, detached Ed25519 verification, exact artifact size/SHA-256 verification, architecture/board binding and runtime policy registration. Failure is fail-closed.

## Root-only lifecycle administration

`/usr/sbin/kingai-devicepack` is installed only in the IoT / Edge image. It is an administrative trust-root tool, not an Agent capability and not a generic execution service.

A release bundle is supplied as a signed manifest plus its detached signature and an artifact directory. Installation requires a trusted device identity first:

```bash
sudo kingai-devicepack install \
  --manifest /media/release/kingai.raspberry-pi-5.json \
  --artifacts /media/release/artifacts \
  --confirm INSTALL:kingai.raspberry-pi-5
```

The signature defaults to `<manifest>.sig`. Custom trust paths exist for factory/test workflows but production should use the standard root-owned directories.

The installer performs these steps before activation:

1. obtains an exclusive lifecycle lock;
2. parses the signed release manifest;
3. checks local architecture and trusted Board ID;
4. copies manifest/signature/artifacts into protected staging directories;
5. recomputes every artifact size and SHA-256 while copying;
6. verifies the detached Ed25519 signature from the root-provisioned keyring;
7. verifies the staged artifacts again using the production Device Pack verifier;
8. records a root administrative audit event;
9. atomically replaces the artifact directory while retaining the previous one as a hidden backup;
10. moves the signature into place and activates the manifest **last**;
11. returns `restart_required:true` instead of unexpectedly restarting a robot/gateway process.

This ordering means an interruption prefers an inactive or verification-failing state instead of activating unsigned/partial bytes.

List installed packs and their verification state with:

```bash
kingai-devicepack list
```

Deactivate a pack with an explicit trust-root confirmation:

```bash
sudo kingai-devicepack deactivate \
  --pack kingai.raspberry-pi-5 \
  --confirm DEACTIVATE:kingai.raspberry-pi-5
```

Deactivation moves the active manifest out of the `.json` namespace first, then retains the signature/artifacts as hidden backups. Restart `kingaid` during an appropriate maintenance/safe-state window to apply installation, upgrade or deactivation. The lifecycle CLI deliberately does not call `systemctl restart` automatically.

Board-handler executables/services remain board-specific integration components. The Device Pack lifecycle tool does not dynamically install arbitrary systemd units or turn signed artifact metadata into a generic root-code launcher.

## Attestation boundary

`attestation` in the identity and OTA manifest is a policy selector, not an attestation proof. TPM2/secure-element/TEE quote generation, verifier infrastructure, enrollment certificates and key rotation require a separate device-specific implementation and HIL/security review before they are marketed as active remote attestation.

## Re-provisioning

Changing `board_id`, Fleet, update channel or trust keys is an administrative trust-root operation. It should be performed out of band or through a dedicated privileged service with explicit owner authorization and audit; do not expose it as a generic `device.*` handler.
