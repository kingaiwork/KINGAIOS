# KINGAI OS IoT / Edge

KINGAI OS IoT / Edge targets **ARM64 and x86-64 edge devices, gateways, robots, industrial computers, and embedded intelligent systems** while keeping the common governed intelligence layer independent from board-specific firmware and privileged hardware code.

The generic artifact is an ext4 root filesystem image compressed as `.img.xz`. It deliberately does **not** claim to be a universal bootable ARM image. A flashable hardware release combines the verified common root filesystem with a concrete Device Pack and the board vendor's boot/kernel/firmware assets.

## Generic Edge Runtime

The common IoT image carries only the KINGAI binaries required for a local governed runtime:

```text
kingai
kingaid
kingai-update
```

It intentionally does **not** ship:

```text
kingai-execd
kingai-installer
kingai-recovery
```

The generic `kingaid` process runs as the non-root `_kingai` identity with the same Peer Identity, Policy, Approval, Task, Memory, Model, and Audit foundations used by the other KINGAI OS forms.

IoT applies a platform-specific systemd drop-in:

```text
KINGAI_REQUIRE_EXECD=false
KINGAI_TASK_RUN_BUDGET=16
KINGAI_DEVICE_RUNTIME_ENABLED=true
KINGAI_DEVICE_PACK_DIR=/etc/kingai/device-packs
KINGAI_DEVICE_HANDLER_ROOT=/run/kingai-device
KINGAI_DEVICE_BROKER_SOCKET=/run/kingai/device-broker.sock
KINGAI_EXECD_SOCKET=/run/kingai/device-broker.sock
```

The socket compatibility variable reuses the existing constrained execution client inside `kingaid`; it does **not** install or start the privileged generic `kingai-execd` binary. On IoT, that socket is served by an in-process non-privileged Device Broker that only understands exact Device Pack capabilities.

## Governed device execution is now wired end to end

The Edge execution path is:

```text
Agent / Task
    ↓
local Peer Identity
    ↓
Agent capability eligibility
    ↓
KINGAI central Policy
    ↓
Approval when required
    ↓
exact Device Pack capability + exact resource
    ↓
non-root Device Broker
    ↓
private AF_UNIX handler socket
    ↓
board-specific least-privileged handler
    ↓
hardware operation
    ↓
Audit / Task result
```

The `system-ops` Agent may declare the scoped family `device.*`, but this is only Agent-level eligibility. A concrete capability such as `device.gpio.read` still remains default-deny until a trusted Device Pack declares it and the runtime installs an exact Policy rule for it.

Device Pack rules cannot weaken static system policy. If both define the same capability, KINGAI uses the higher risk level and the stricter approval/owner requirements.

## Device capabilities, not generic root

GPIO, I2C, SPI, cameras, accelerators, power management, motors, robotics control, and firmware operations must be exposed through explicit **Device Pack capability handlers** with their own validation and least-privilege service identity.

A board-specific privileged action must not be implemented by adding a generic root shell to the IoT image.

The Device Broker:

- never interprets a handler ID as a command or executable path;
- never invokes `sh -c`;
- never expands wildcards, environment variables, or shell expressions;
- requires the request target to exactly equal a manifest-declared resource;
- rejects duplicate capability ownership across installed packs;
- rejects packs for the wrong CPU architecture;
- enforces Board ID matching when a pack declares `board_ids`;
- accepts only root-provisioned manifests that are not group/world writable;
- accepts only private Unix handler sockets owned by the `kingaid` UID;
- has a bounded request size, response size, and execution timeout.

See [`HANDLER-PROTOCOL.md`](HANDLER-PROTOCOL.md) for the board-handler contract.

## Device Pack installation

The image reserves:

```text
/etc/kingai/device-packs/
```

for production Device Pack manifests. The schema is also installed into the image at:

```text
/usr/share/kingai/iot/device-pack.schema.json
```

The handler socket namespace is:

```text
/run/kingai-device/
```

and is created as a root-owned, non-group-writable directory by systemd-tmpfiles.

If a pack contains board IDs, provisioning must set:

```text
KINGAI_DEVICE_BOARD_ID=<exact-board-id>
```

before restarting `kingaid`.

A Device Pack manifest declaring `security.signed_manifest=true` still has to be cryptographically verified by the installer or fleet provisioning layer **before** it is written into the root-owned trusted directory. The runtime ownership checks protect local trust after provisioning; they are not a replacement for release-signature verification.

## Generic image versus board release

A flashable board release is produced by combining the verified common root filesystem with a signed **Device Pack** for a concrete hardware family, for example:

- Raspberry Pi-class ARM64 devices;
- NVIDIA Jetson-class edge AI devices;
- generic UEFI ARM64 servers / industrial PCs;
- x86-64 UEFI gateways / industrial PCs;
- future robotics / industrial reference boards.

Each Device Pack is expected to define:

- CPU architecture and optional exact Board IDs;
- kernel and kernel configuration;
- DTB / ACPI requirements;
- bootloader / UEFI integration;
- redistributable firmware;
- hardware acceleration;
- declared device capabilities and exact resources;
- handler IDs and least-privilege runtime requirements;
- recovery / update behavior;
- artifact hashes and release evidence.

This keeps the KINGAI intelligence/governance core independent from one hardware vendor.

## Security boundary

The base `kingaid.service` hardening remains in force for IoT:

```text
User=_kingai
NoNewPrivileges=yes
PrivateTmp=yes
PrivateDevices=yes
ProtectSystem=strict
ProtectHome=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectKernelLogs=yes
ProtectControlGroups=yes
RestrictAddressFamilies=AF_UNIX
CapabilityBoundingSet=
AmbientCapabilities=
```

The common daemon therefore does not need raw `/dev` access or Linux capabilities just because a board has GPIO, an NPU, or robot motors. Only a board handler receives the minimum hardware permission it actually needs.

## CI evidence

The IoT Edge smoke workflow runs focused tests for Device Pack, Policy, Agent registry, and `kingaid`, then builds both `amd64` and `arm64` generic root filesystems and images. It verifies that:

- `kingaid` is installed and enabled;
- `kingai-update` is present;
- generic `kingai-execd` is absent;
- Installer and Recovery binaries are absent;
- the IoT drop-in disables the generic ExecD requirement;
- the embedded governed Device Broker is enabled;
- the IoT Scheduler budget is 16 steps per run call;
- the Device Pack directory is root-owned;
- the Device Pack schema and handler protocol ship in the image;
- the protected handler tmpfiles rule ships in the image;
- the generic compressed image manifest remains marked `bootable: false` and `device_pack_required: true`.

Real hardware support is still gated on real board validation. A successful generic ARM64 build is not advertised as Raspberry Pi or Jetson certification.

## Next hardware-release layer

The generic runtime is now ready for board work. A production board release still needs, per hardware family:

1. boot/kernel/DTB or ACPI integration;
2. vendor firmware and redistribution review;
3. concrete handlers for GPIO/camera/NPU/GPU/motors/power as required;
4. signed provisioning of the Device Pack manifest;
5. hardware-in-the-loop boot, capability, power-loss, and rollback tests;
6. board-specific OTA/recovery integration.

---

# 中文摘要

KINGAI OS IoT / Edge 现在不再只是“ARM64/x86-64 可以构建的精简 rootfs”，而是已经补上了 **Device Pack → Policy → Approval → Device Broker → 硬件 Handler** 的执行闭环。

通用 IoT 镜像仍然只带：

```text
kingai
kingaid
kingai-update
```

仍然不带：

```text
kingai-execd
kingai-installer
kingai-recovery
```

`kingaid` 继续以 `_kingai` 非 root 身份运行，并保留 `PrivateDevices=yes`、零 Capability Bounding Set、Policy、Approval、Task、Memory、Model、Audit 等安全和智能核心。

设备能力采用明确白名单：例如 `device.gpio.read` 只能访问 Device Pack 声明的 `gpio:17`，不能变成任意 GPIO、任意 `/dev`、任意路径或 root shell。高风险控制仍然按照 L0-L6 风险级别进入中央审批机制。

下一阶段不再需要重做通用 Edge Runtime，重点应转向真实硬件 Device Pack：Raspberry Pi / Jetson / 通用 UEFI ARM64 / x86-64 工业网关 / 机器人参考板，并完成真实启动、硬件能力、OTA、掉电恢复和回滚验证。
