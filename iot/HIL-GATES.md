# KINGAI OS IoT / Edge Hardware-in-the-Loop Release Gates

A platform remains `hardware_verified:false` and `bootable_release:false` until the applicable gates below pass on real hardware. VM/QEMU tests supplement these gates but do not replace them.

## Gate 1 — provenance and identity

- Exact board/revision is recorded.
- Device identity is root-owned, immutable to normal agents and matches Device Pack `board_ids`.
- Every firmware/bootloader/driver artifact has reviewed source/license metadata.
- Detached Device Pack signature verifies with the provisioned release key.
- Any manifest, signature or artifact byte tamper causes fail-closed startup.

## Gate 2 — boot and lifecycle

- Cold boot from powered-off state.
- 20 consecutive reboot cycles without filesystem or state corruption.
- Clean shutdown/restart of `kingaid` and every board handler.
- Recovery behavior after a handler is missing, unhealthy or returns malformed data.
- Clock loss/RTC reset and network-unavailable boot do not bypass local policy.

## Gate 3 — capability isolation

For every declared capability:

- exact allowed resources succeed;
- undeclared resource is denied;
- undeclared capability is denied;
- wildcard/path/shell-like targets are denied;
- normal `main` Agent cannot inherit `system-ops` device authority;
- required Approval is enforced before the handler receives the request;
- audit records the decision and result.

## Gate 4 — physical I/O

Where applicable test GPIO, I2C, SPI, UART, camera, microphone, GPU/NPU/accelerator, relays, power controllers, motors and actuators.

- Read operations stay within declared buses/addresses/lines/devices.
- Mutating resources respect risk floors.
- Motor/actuator/power emergency-stop or safe-state behavior is tested independently of the AI process.
- Handler crash/restart must not leave outputs in an unsafe latched state.

## Gate 5 — load and thermal behavior

- Sustained inference/compute workload.
- Concurrent telemetry + model inference + task execution.
- Memory pressure and disk-full behavior.
- Thermal throttling and accelerator reset where applicable.
- Watchdog behavior if the platform uses one.

## Gate 6 — OTA and rollback

- Correctly targeted signed update succeeds.
- Wrong arch, board, class, Device Pack or attestation target is rejected before writes.
- Corrupt artifact and metadata are rejected.
- Power removal during download, staging, slot selection and first boot is tested.
- Failed first boot returns to the prior known-good slot.
- Successful health reconciliation commits the new slot exactly once.
- Repeated failed updates cannot erase the known-good recovery path.

## Gate 7 — network and Fleet failure

- Loss of WAN/Fleet connection does not disable local Policy/Approval/Audit.
- DNS/TLS/TUF failures fail closed for update operations.
- Device continues permitted local-first tasks when cloud is unavailable.
- Fleet metadata cannot overwrite root trust keys or local board identity without a privileged provisioning path.

## Gate 8 — release evidence

Store with the release:

- hardware model/revision and serial redaction policy;
- Device Pack ID/version and manifest digest;
- OS version/build digest;
- test runner version;
- pass/fail results and logs;
- update/rollback evidence;
- known limitations.

Only after the evidence is reviewed may `iot/support-matrix.json` mark the platform hardware verified or bootable.
