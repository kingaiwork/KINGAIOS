# KINGAI OS IoT / Edge Board Porting

A board is not considered supported because the generic ARM64 or amd64 userspace builds. Hardware support is a release claim and requires a verified Device Pack plus hardware-in-the-loop gates.

## Porting layers

1. **Architecture layer** — `amd64` or `arm64` KINGAI IoT root filesystem.
2. **Boot layer** — exact UEFI/U-Boot/vendor boot chain, kernel, initrd, DTB and firmware provenance.
3. **Device Pack** — signed manifest, exact board IDs, immutable artifact hashes and capability/resource allowlists.
4. **Handler layer** — one or more least-privilege AF_UNIX services for GPIO, I2C, SPI, UART, camera, accelerator, motor, power or vendor-specific functions.
5. **Identity layer** — root-owned `/etc/kingai/device.json` with device, board, class, fleet, update-channel and attestation policy.
6. **Update layer** — signed/TUF target compatibility, boot-slot integration and rollback behavior appropriate to the board.
7. **HIL layer** — real hardware boot, capability, failure and update tests.

## Required porting sequence

1. Copy the closest file in `iot/templates/`.
2. Assign a stable canonical `board_id`; do not derive authorization from a marketing name supplied by an untrusted client.
3. Replace every zero-size/zero-hash artifact placeholder with the exact reviewed artifact.
4. Record license and source for firmware, bootloader and driver artifacts.
5. Declare only resources physically exposed to KINGAI. Do not use wildcards.
6. Implement handlers with `sdk/edgehandler` or an equivalent protocol implementation. A handler must repeat the exact capability/resource allowlist.
7. Provision a Device Pack Ed25519 public key under `/etc/kingai/trust/device-pack-keys/` and keep signing keys off-device.
8. Sign the exact manifest bytes and install the detached `.json.sig` beside the manifest.
9. Install immutable artifacts beneath `/usr/lib/kingai/device-packs/<pack-id>/`.
10. Provision `/etc/kingai/device.json` and verify board identity agreement.
11. Run the mandatory gates in `HIL-GATES.md`.
12. Only after passing the gates may `hardware_verified` and `bootable_release` be changed to `true` in `support-matrix.json`.

## Boot image rule

`scripts/build-iot-image.sh` produces a generic compressed ext4 root filesystem image and intentionally declares `bootable:false`. A board release must supply an exact boot layout or board-specific flashing flow. Do not change the generic image metadata to `bootable:true` merely because it contains a kernel artifact in a Device Pack.

## Physical safety rule

KINGAI applies operation and resource risk floors. A Device Pack may raise risk but cannot lower it. Motor, actuator, relay and power control are at least L4. Firmware, boot and flash changes are at least L5. High-risk mutations require Approval before the Device Broker reaches a board handler.
