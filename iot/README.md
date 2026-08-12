# KINGAI OS IoT / Edge

KINGAI OS IoT / Edge separates the **common intelligent operating-system root filesystem** from device-specific boot firmware and kernels.

The generic artifact is an ext4 root filesystem image compressed as `.img.xz`. It contains KINGAI Core, policy, agents, model routing, memory/update foundations and Linux userland, but deliberately does **not** claim to be a universal bootable ARM image.

A flashable board release is produced by combining the verified common root filesystem with a signed **Device Pack** for a concrete hardware family, for example:

- Raspberry Pi-class ARM64 devices
- NVIDIA Jetson-class edge AI devices
- generic UEFI ARM64 servers/industrial PCs
- future robotics/industrial reference boards

Each Device Pack is expected to define kernel, DTB/ACPI requirements, bootloader/UEFI integration, firmware, hardware acceleration and recovery behavior. This avoids coupling the KINGAI intelligent core to one hardware vendor.
