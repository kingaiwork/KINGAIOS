# KINGAI OS Installer

The current installer component is deliberately **planning-only**.

`kingai-installer list` discovers top-level disks through `lsblk`. `kingai-installer plan --target /dev/DEVICE --profile ...` validates that the target is a writable, unmounted top-level disk and generates an A/B + persistent STATE partition plan.

The Developer Foundation has **no destructive execute command**. Disk writing remains blocked until automated VM tests cover partition creation, bootloader installation, power-loss behavior, post-install boot, encryption, upgrade and rollback.

Planned stable layout:

- EFI — boot partition
- ROOT_A — verified active system slot
- ROOT_B — inactive atomic update slot
- STATE — encrypted persistent state
