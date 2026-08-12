# KINGAI OS Device Packs

## English

KINGAI OS separates the common intelligent operating-system root filesystem from hardware-specific boot and acceleration components.

A **Device Pack** is the signed, reviewable hardware contract that transforms a generic KINGAI Edge root filesystem into a device-family release.

### Device Pack responsibilities

A Device Pack may declare:

- target architecture and board identifiers;
- UEFI, U-Boot or vendor boot method;
- kernel and initrd artifacts;
- DTB/ACPI requirements;
- bootloader and firmware artifacts;
- hardware acceleration drivers/configuration;
- artifact hashes and sizes;
- explicit license/source information where needed;
- Secure Boot capability and minimum firmware policy.

### Security and legal gates

Official Device Packs must pass both:

- `signed_manifest = true`
- `redistribution_reviewed = true`

Firmware and bootloader artifacts require an explicit license field. Every artifact requires a SHA-256 hash. The common KINGAI OS rootfs never treats an unsigned or unreviewed Device Pack as trusted.

### Why Device Packs are separate

A universal ARM64 userspace does not imply a universal bootable ARM64 image. Different boards may require different firmware, kernels, DTBs, bootloaders and recovery mechanisms. Separating these concerns keeps KINGAI Intelligence, Agent Runtime, Memory, Policy and Update layers portable without pretending hardware boot chains are identical.

Schema: `iot/device-pack.schema.json`

---

# KINGAI OS Device Pack

## 中文

KINGAI OS 将通用智慧操作系统 RootFS 与具体硬件的启动、固件和加速组件分离。

**Device Pack** 是一个经过签名、可审核的硬件契约，用于把通用 KINGAI Edge RootFS 转换为某个具体设备系列可发行版本。

### Device Pack 负责定义

- CPU 架构与 Board ID；
- UEFI、U-Boot 或厂商启动方式；
- Kernel / initrd；
- DTB / ACPI 要求；
- Bootloader / Firmware；
- AI / GPU / NPU 等硬件加速驱动与配置；
- 每个 Artifact 的哈希和大小；
- 必要的许可证与源码来源；
- Secure Boot 能力与最低 Firmware 策略。

### 安全与法律门禁

官方 Device Pack 必须同时满足：

- `signed_manifest = true`
- `redistribution_reviewed = true`

Firmware 与 Bootloader 必须明确许可证；每个 Artifact 必须有 SHA-256。未签名、未完成再发行审核的 Device Pack 不得被 KINGAI OS 当作可信硬件包。

### 为什么必须独立 Device Pack

通用 ARM64 Userspace 并不意味着存在通用可启动 ARM64 镜像。不同设备的 Firmware、Kernel、DTB、Bootloader 和 Recovery 都可能完全不同。Device Pack 的独立设计可以让 KINGAI Intelligence、Agent Runtime、Memory、Policy 和 Update 保持跨硬件可移植，而不会错误地假设所有 ARM 设备的启动链相同。

Schema：`iot/device-pack.schema.json`
