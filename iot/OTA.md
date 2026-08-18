# KINGAI OS IoT / Edge OTA

KINGAI OS already has signed update manifests, TUF metadata support, A/B slot state, boot health and rollback primitives. Edge adds target compatibility so a cryptographically valid artifact is not automatically valid for every device.

## Trust chain

1. TUF metadata selects the trusted target.
2. The KINGAI update envelope verifies the signed manifest.
3. Artifact size and SHA-256 are verified.
4. Edge target compatibility checks profile, architecture, channel, board ID, device class, required Device Packs and accepted attestation mode.
5. The board-specific update implementation stages the inactive boot/root slot.
6. Boot health commits a successful slot or rolls back a failed boot.

All six layers are required for a release-grade Edge OTA path.

## Compatibility fields

Signed update manifests may contain:

- `board_ids`
- `device_classes`
- `required_device_packs`
- `attestation_modes`

These fields are part of the signed manifest payload. Changing them invalidates the signature.

## Current execution boundary

The repository's direct A/B write executor is currently reviewed only for `amd64` and a GRUB-oriented KINGAI disk layout. That limitation is deliberate.

ARM64 and vendor boards may use the signed/TUF verification and target-compatibility code now, but writing boot firmware, DTBs or vendor boot slots requires the corresponding verified Device Pack and board-specific update handler. Do not advertise ARM64 A/B flashing as complete until HIL power-loss and rollback gates pass on each board family.

## Failure rules

An update must stop before writes when any of these are true:

- signature or TUF verification fails;
- artifact hash/size fails;
- product/profile/architecture/channel mismatch;
- board or device class mismatch;
- a required Device Pack is absent;
- attestation mode is not accepted;
- target slot/layout cannot be proven inactive and correct;
- confirmation/owner authorization is missing for privileged update execution.

## Rollout policy

Use staged channels and Fleet cohorts. `pinned` devices should not move automatically. A Fleet orchestrator can decide *when* to offer an update, but the device still performs local cryptographic and compatibility verification before accepting it.
