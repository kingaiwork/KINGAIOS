# KINGAI OS IoT / Edge

KINGAI OS IoT / Edge separates the **common governed intelligent userspace** from device-specific boot firmware, kernels and privileged hardware operations.

The generic artifact is an ext4 root filesystem image compressed as `.img.xz`. It deliberately does **not** claim to be a universal bootable ARM image.

## Generic Edge Runtime

The common IoT image now carries only the KINGAI binaries required for a local governed runtime:

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

The generic `kingaid` process runs as the non-root `_kingai` identity with the same local Peer Identity, Policy, Approval, Task, Memory, Model and Audit foundations used by the other KINGAI OS forms.

IoT applies a platform-specific systemd drop-in:

```text
KINGAI_REQUIRE_EXECD=false
KINGAI_TASK_RUN_BUDGET=16
```

This keeps Edge execution smaller and prevents a generic privileged system broker from being exposed on devices where the correct privileged operations are hardware-specific.

## Device capabilities, not generic root

Future device-level actions such as GPIO, camera control, accelerator reset, power management, motor/robotics control or firmware operations must be exposed through explicit **Device Pack capability handlers** with their own validation and policy.

A board-specific privileged action must not be implemented by adding a generic root shell to the IoT image.

The intended path is:

```text
Agent / Task
    ↓
KINGAI Policy + Approval
    ↓
Declared Device Capability
    ↓
Device Pack handler
    ↓
validated hardware operation
    ↓
Audit / Result
```

## Generic image versus board release

A flashable board release is produced by combining the verified common root filesystem with a signed **Device Pack** for a concrete hardware family, for example:

- Raspberry Pi-class ARM64 devices;
- NVIDIA Jetson-class edge AI devices;
- generic UEFI ARM64 servers / industrial PCs;
- future robotics / industrial reference boards.

Each Device Pack is expected to define:

- kernel and kernel configuration;
- DTB / ACPI requirements;
- bootloader / UEFI integration;
- redistributable firmware;
- hardware acceleration;
- declared device capabilities;
- recovery behavior;
- artifact hashes and release evidence.

This keeps the KINGAI intelligent/governance core independent from one hardware vendor.

## CI evidence

The IoT Edge smoke workflow builds both `amd64` and `arm64` generic root filesystems and images. It verifies that:

- `kingaid` is installed and enabled;
- `kingai-update` is present;
- generic `kingai-execd` is absent;
- Installer and Recovery binaries are absent;
- the IoT drop-in explicitly disables the ExecD requirement;
- the IoT Scheduler budget is 16 steps per run call;
- the generic compressed image manifest remains marked `bootable: false` and `device_pack_required: true`.

Real hardware support is still gated on real board validation. A successful generic ARM64 build is not advertised as Raspberry Pi or Jetson certification.

---

# 中文摘要

KINGAI OS IoT / Edge 现在采用更精简的运行方式：

```text
只带：
kingai
kingaid
kingai-update

不带：
kingai-execd
kingai-installer
kingai-recovery
```

IoT 的 `kingaid` 仍然是非 root 的 KINGAI Runtime，继续保留 Identity、Policy、Approval、Task、Memory、Model、Audit 等核心能力，但**不提供通用高权限 ExecD**。

需要 GPIO、摄像头、NPU/GPU、机器人、电源或固件等设备级高权限动作时，应由具体 Device Pack 提供明确 Capability Handler，再经过 KINGAI Policy / Approval；不能为了方便给 IoT Agent 一个无限 root shell。

IoT 单次 Task Scheduler 默认预算降低为 16 Step，更适合资源受限设备。
