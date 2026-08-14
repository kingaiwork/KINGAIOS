# KINGAI OS IoT / Edge

KINGAI OS IoT / Edge targets **ARM64 and x86-64 gateways, robots, industrial computers and embedded intelligent systems**. The common intelligence/governance core stays independent from board-specific firmware and privileged hardware code.

The generic artifact remains a compressed ext4 root filesystem image. It deliberately reports `bootable:false`: a generic ARM64 userspace build is not a Raspberry Pi, Jetson or universal ARM boot image. A board release needs an exact Device Pack, boot layout and hardware-in-the-loop evidence.

## Runtime footprint

The generic IoT image carries:

```text
kingai
kingaid
kingai-update
kingai-devicepack   # root-only trust lifecycle administration
```

It intentionally does not carry the generic privileged execution daemon, installer or recovery binary:

```text
kingai-execd
kingai-installer
kingai-recovery
```

`kingaid` remains `_kingai`, non-root, `PrivateDevices=yes`, AF_UNIX only and with an empty Linux capability bounding set. `kingai-devicepack` is a separate root-only administrative utility and is never exposed as an Agent/device execution capability.

The IoT drop-in selects the governed Device Broker:

```text
KINGAI_REQUIRE_EXECD=true
KINGAI_TASK_RUN_BUDGET=16
KINGAI_DEVICE_RUNTIME_ENABLED=true
KINGAI_DEVICE_IDENTITY=/etc/kingai/device.json
KINGAI_DEVICE_PACK_DIR=/etc/kingai/device-packs
KINGAI_DEVICE_ARTIFACT_ROOT=/usr/lib/kingai/device-packs
KINGAI_DEVICE_TRUST_DIR=/etc/kingai/trust/device-pack-keys
KINGAI_DEVICE_HANDLER_ROOT=/run/kingai-device
KINGAI_DEVICE_BROKER_SOCKET=/run/kingai/device-broker.sock
KINGAI_EXECD_SOCKET=/run/kingai/device-broker.sock
```

`KINGAI_REQUIRE_EXECD` is a legacy environment name used by the common readiness path; on IoT it means the selected execution broker is required to be healthy, not that the generic privileged ExecD is installed. `KINGAI_EXECD_SOCKET` is likewise protocol compatibility with the existing constrained execution client. The IoT socket is served by the non-privileged in-process Device Broker; `kingai-execd` is not installed. If an installed Device Pack references a board handler that is missing or unhealthy, Device Broker `/healthz` returns unavailable and KINGAI readiness fails instead of reporting a false healthy state.

## Execution chain

```text
Agent / Task
  -> local Peer Identity
  -> Agent capability eligibility
  -> central Policy
  -> Approval when required
  -> exact signed Device Pack capability/resource
  -> non-root Device Broker
  -> private AF_UNIX board handler
  -> hardware operation
  -> Audit / Task result
```

`system-ops` may be eligible for the `device.*` family, but a concrete capability remains default-deny until an installed verified Device Pack declares the exact capability and exact resource. Device Pack runtime rules can never weaken a stricter static Policy rule.

## Device Pack trust

Production Device Pack loading is fail-closed. Before a capability is registered, KINGAI verifies:

- strict Device Pack v2 schema;
- CPU architecture;
- exact Board ID when declared;
- root-owned/non-writable manifest and trust directories;
- detached Ed25519 signature over the exact manifest bytes;
- root-provisioned Device Pack public key;
- immutable installed artifact size and SHA-256;
- duplicate pack/capability ownership;
- physical operation/resource risk floors;
- exact resource allowlists.

The runtime layout is:

```text
/etc/kingai/device.json
/etc/kingai/device-packs/<pack-id>.json
/etc/kingai/device-packs/<pack-id>.json.sig
/etc/kingai/trust/device-pack-keys/<key-id>.pub
/usr/lib/kingai/device-packs/<pack-id>/<artifact>
/run/kingai-device/<handler>.sock
```

A Boolean `signed_manifest:true` is not proof. The detached Ed25519 signature and artifact bytes are verified at startup.

## Device Pack release and lifecycle

`tools/device-pack-release` turns an integration template into a release manifest only after it receives real artifacts, their computed hashes/sizes, an Ed25519 release key and an explicit redistribution review acknowledgement plus immutable review reference. Private signing keys remain off-device.

On-device `/usr/sbin/kingai-devicepack` provides root-only `install`, `list` and `deactivate` operations. Installation requires the trusted device identity and an exact `INSTALL:<pack-id>` confirmation. It acquires an exclusive lifecycle lock, stages the bytes under protected destination directories, verifies signature/architecture/Board ID/hash again, retains prior files as hidden backups, moves the signature into place, and activates the manifest **last**. Deactivation requires `DEACTIVATE:<pack-id>` and removes the active manifest namespace first. Both return `restart_required:true`; the tool deliberately does not auto-restart a robot or gateway.

The root lifecycle log is separate from `kingaid`'s normal audit file so a root-created administrative log cannot accidentally change ownership of the daemon audit stream.

## Trusted device identity

`/etc/kingai/device.json` binds the local device to a stable `device_id`, `board_id`, class, Fleet, update channel, provisioning mode and expected attestation mode. It is root-owned and cannot be replaced by an Agent request.

If `KINGAI_DEVICE_BOARD_ID` is supplied and conflicts with the trusted identity file, startup fails. Board-specific Device Packs do not load without a matching Board ID.

The `attestation` field is a policy fact, not a remote-attestation proof. TPM2/secure-element/TEE quote and verifier infrastructure remain separate hardware-specific work.

## Capability and physical-risk model

Canonical resources and risk floors live in `capability-catalog.json`.

Examples:

- telemetry/sensors: at least L0;
- GPIO/I2C/SPI/UART/camera/GPU/NPU: at least L1;
- microphone/location/network/storage: at least L2;
- safety controls: at least L3;
- motors/actuators/relays/power: at least L4;
- firmware/boot/flash: at least L5.

Operations also have floors: read L0, write/compute L1, control L3, reset L4, update L5. The effective minimum is the stricter of operation and resource floors. A Device Pack may raise risk but cannot lower it. Every non-read capability at L3+ requires Approval, so a high-risk physical resource cannot bypass Approval merely by being mislabeled as `compute`.

## Board handlers

Board handlers receive the minimum OS permission needed for hardware. They are not part of `kingaid` and should use `sdk/edgehandler` or the protocol in `HANDLER-PROTOCOL.md`.

The SDK repeats the exact capability/resource allowlist below the Device Broker, refuses shell/path/wildcard targets, binds only below `/run/kingai-device`, requires a private socket, and requires an explicit non-root `kingaid` socket owner.

Board-handler executables/services are board-specific integration components. The Device Pack lifecycle manager does not dynamically install arbitrary systemd units or turn signed metadata into a generic root-code launcher.

See `HANDLER-SDK.md` for the integration contract. Physical safety systems such as emergency stops and hardware interlocks remain independent from AI policy.

## Edge OTA

`kingai-update edge-verify` combines:

1. signed KINGAI update envelope;
2. artifact size/SHA-256;
3. trusted device identity;
4. verified installed Device Packs;
5. profile/architecture/channel targeting;
6. Board ID/device-class targeting;
7. required Device Pack targeting;
8. accepted attestation mode.

The existing direct A/B write executor remains reviewed only for the repository's amd64/GRUB layout. ARM64/vendor-board writes need board-specific boot/update integration and HIL rollback tests. See `OTA.md`.

## Platform status

Machine-readable state is in `support-matrix.json`.

The repository includes integration templates for:

- generic UEFI amd64;
- generic UEFI ARM64;
- Raspberry Pi 5;
- Raspberry Pi CM5;
- NVIDIA Jetson Orin Nano;
- NVIDIA Jetson AGX Orin.

These are **integration templates, not hardware certification**. Current entries remain `hardware_verified:false` and `bootable_release:false` until signed provenance review plus the gates in `HIL-GATES.md` pass on real devices.

## CI and release truth

The IoT smoke workflow covers both amd64 and arm64 and checks Device Pack crypto/runtime/risk/handler-health, lifecycle helpers, trusted identity, Edge OTA compatibility, Agent/Policy rules, Handler SDK allowlists, template validation, support-matrix anti-overclaim assertions, rootfs trust permissions, required Device Broker readiness, non-root daemon sandboxing, absence of generic ExecD/Installer/Recovery, generic image digest and size budget, plus `bootable:false` / `device_pack_required:true` metadata.

A dedicated generic IoT release workflow may publish dev/beta prereleases after software gates; RC/stable generic releases are blocked. Board RC/stable claims remain separately blocked by real HIL evidence.

## Porting and release documents

- `BOARD-PORTING.md` — board integration sequence.
- `PROVISIONING.md` — device identity, trust and Device Pack lifecycle.
- `HANDLER-PROTOCOL.md` — wire protocol.
- `HANDLER-SDK.md` — least-privilege handler SDK.
- `OTA.md` — signed/TUF Edge update model.
- `HIL-GATES.md` — mandatory real-hardware release gates.

---

## 中文摘要

KINGAI OS IoT / Edge 已从“ARM64/x86-64 精简 rootfs”推进为完整的受治理 Edge 软件基线：**设备身份 → 签名 Device Pack → 原子生命周期管理 → artifact 完整性 → 中央 Policy/Approval → 非 root Device Broker → 最小权限硬件 Handler → Audit/Task Result → 受目标约束的 OTA**。

通用镜像仍明确不是万能可启动板卡镜像。Raspberry Pi、Jetson、通用 UEFI ARM64/x86-64 和机器人硬件必须使用具体 Device Pack，并通过真实启动、硬件能力、OTA 掉电和回滚 HIL 测试后，才能在支持矩阵中标记为正式硬件支持。
