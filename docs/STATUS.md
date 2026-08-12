# KINGAI OS — Verified Project Status

**Status date:** 2026-08-12  
**Development line:** D4 Developer Foundation / Pre-Alpha

## English

This document separates what KINGAI OS has **actually verified** from what remains a roadmap or protected release gate.

### Published

#### KINGAI OS Server Developer Preview

- Release: `v0.1.0-dev-server-dev.2`
- Artifact: `KINGAI-OS-Server-0.1.0-dev-amd64.iso`
- Size: `1,092,444,160 bytes` (approximately 1.02 GiB)
- Distribution: GitHub Pre-release because the artifact is below the 2 GiB large-artifact routing threshold.
- Assets: ISO, SHA-256 checksum and machine-readable manifest.
- Install-to-disk: **disabled** in this Developer Foundation image.
- Production Secure Boot signing: **not enabled yet**.

The release was rebuilt from repository source in GitHub Actions and passed core tests, rootfs construction, ownership validation, ISO checksum/manifest validation and QEMU BIOS boot verification before publication.

### Verified engineering foundations

#### Core / Security

- `kingai`, `kingaid`, `kingai-update`, `kingai-installer` build successfully.
- Go unit tests and `go vet` are CI gates.
- `kingaid` uses local Unix-socket IPC rather than opening a TCP management port.
- Capability policy defaults to deny unknown capabilities.
- High-risk capabilities require explicit approval; trust-root modification is owner-only.
- Client JSON cannot self-assert `Owner` or `Approved` authorization.
- Local peer identity is derived from Unix peer credentials.
- `main`, `system-ops` and `sec-ops` are separated by the agent registry and trusted local identities.
- Audit events record policy decisions without storing raw target values.
- Desktop-readable public status is sanitized and excludes prompts, memory content, API keys, passwords, tokens and secrets.

#### Server

- Ubuntu 26.04 based KINGAI Server rootfs: verified.
- KINGAI identity (`os-release`, MOTD, issue): verified.
- Linux kernel/initramfs inclusion: verified.
- `kingaid` systemd enablement: verified.
- Hybrid Live ISO generation: verified.
- Casper filesystem/checksum metadata: verified.
- QEMU BIOS boot to `KINGAI OS 0.1 Developer Foundation`: verified.
- Build-host UID/GID leakage checks: verified.

#### Desktop

- Ubuntu 26.04 Desktop rootfs: verified.
- Plasma 6 / KWin Wayland / SDDM / Qt 6 desktop foundation: verified.
- KINGAI Welcome first-run selector: included.
- KINGAI Intelligence / Flow / Classic Plasma 6 package manifests: verified.
- Switchable desktop layout scripts: included.
- KINGAI Agent Center package and sanitized local status integration: included and under CI validation.
- Full Desktop Live ISO boot/size validation: **in progress**.

#### IoT / Edge

- amd64 generic KINGAI Edge rootfs/image pipeline: verified.
- arm64 generic KINGAI Edge rootfs/image pipeline: verified.
- `.img.xz` checksum and manifest validation: verified.
- Hard compressed-image budget gate: enabled.
- Generic Edge artifact intentionally reports `bootable=false` and `device_pack_required=true`.
- Board-specific bootable Device Packs (for concrete Raspberry Pi/Jetson/industrial hardware families): not yet released.

### Supply-chain / compliance foundation

- Full Apache-2.0 text for KINGAI-authored repository code.
- NOTICE and third-party policy included.
- Installed package inventory is embedded in rootfs builds.
- Deterministic SPDX 2.3 package SBOM generation has been added to the current build line.
- Future Live ISO builds export the SPDX SBOM both inside the ISO and as a release-side artifact.
- Unknown package-license conclusions are intentionally represented as `NOASSERTION` instead of being guessed.
- Release channels are gated as `dev → beta → rc → stable`.

### Protected / incomplete gates

The following are deliberately **not claimed as complete**:

- production Secure Boot signing and offline release-key custody;
- destructive install-to-disk execution;
- A/B slot activation and automatic rollback in production;
- full recovery-environment validation;
- final TUF root/threshold key operations;
- production vulnerability/CVE automation and final legal review;
- R2 large-artifact publishing credentials in GitHub Secrets;
- board-specific IoT/Edge boot packs;
- Stable release lifecycle commitment.

RC and Stable publication remain blocked until their release gates are implemented and verified.

---

# KINGAI OS — 已验证项目状态

**状态日期：** 2026-08-12  
**开发阶段：** D4 Developer Foundation / Pre-Alpha

## 中文

本文档严格区分 KINGAI OS **已经真实验证的能力** 与仍属于路线图或受保护发行门禁的能力。

### 已发布

#### KINGAI OS Server Developer Preview

- Release：`v0.1.0-dev-server-dev.2`
- 镜像：`KINGAI-OS-Server-0.1.0-dev-amd64.iso`
- 大小：`1,092,444,160 bytes`（约 1.02 GiB）
- 发布位置：GitHub Pre-release，因为该文件低于 2 GiB 大文件分流阈值。
- 发布资产：ISO、SHA-256 校验文件、机器可读 manifest。
- 写盘安装：当前 Developer Foundation **未开放**。
- 生产 Secure Boot 签名：**尚未启用**。

该版本由 GitHub Actions 从仓库源码重新构建，并在正式上传前通过核心测试、RootFS 构建、ownership 检查、ISO 校验/manifest 检查以及 QEMU BIOS 真实启动验证。

### 已验证工程基础

#### 核心 / 安全

- `kingai`、`kingaid`、`kingai-update`、`kingai-installer` 可以成功构建。
- Go 单元测试与 `go vet` 已成为 CI 门禁。
- `kingaid` 使用本地 Unix Socket IPC，不对外开放 TCP 管理端口。
- Capability Policy 对未知能力默认拒绝。
- 高风险能力需要显式批准；Trust Root 修改仅允许 Owner。
- 客户端 JSON 不能自行伪造 `Owner` 或 `Approved`。
- 本机调用身份通过 Unix peer credentials 获得。
- `main`、`system-ops`、`sec-ops` 通过 Agent Registry 和受信本机身份隔离。
- Audit 保存策略决策，但不直接保存原始目标值。
- 桌面可读 Public Status 已做去敏感化，不包含 Prompt、记忆正文、API Key、密码、Token 或 Secret。

#### Server

- 基于 Ubuntu 26.04 的 KINGAI Server RootFS：已验证。
- KINGAI `os-release` / MOTD / issue 品牌身份：已验证。
- Linux Kernel / initramfs：已验证。
- `kingaid` systemd 启用：已验证。
- Hybrid Live ISO：已验证。
- Casper filesystem/checksum 元数据：已验证。
- QEMU BIOS 启动至 `KINGAI OS 0.1 Developer Foundation`：已验证。
- 构建机 UID/GID 污染检查：已验证。

#### Desktop

- Ubuntu 26.04 Desktop RootFS：已验证。
- Plasma 6 / KWin Wayland / SDDM / Qt 6 基础：已验证。
- KINGAI Welcome 首次进入选择器：已加入。
- KINGAI Intelligence / Flow / Classic Plasma 6 包 manifest：已验证。
- 三套可切换桌面布局脚本：已加入。
- KINGAI Agent Center 与本机去敏感状态：已加入并持续进行 CI 验证。
- 完整 Desktop Live ISO 启动与体积验证：**进行中**。

#### IoT / Edge

- amd64 通用 KINGAI Edge RootFS / 镜像链：已验证。
- arm64 通用 KINGAI Edge RootFS / 镜像链：已验证。
- `.img.xz` SHA / manifest：已验证。
- 压缩镜像硬体积门禁：已启用。
- 通用 Edge 镜像明确标记 `bootable=false`、`device_pack_required=true`。
- Raspberry Pi / Jetson / 工业设备等具体硬件 Device Pack：尚未正式发布。

### 供应链 / 合规基础

- KINGAI 自有仓库代码提供完整 Apache-2.0 文本。
- 已加入 NOTICE 与第三方组件规则。
- RootFS 自动嵌入实际安装包清单。
- 当前构建链已加入确定性 SPDX 2.3 Package SBOM。
- 后续 Live ISO 会同时在 ISO 内部和 Release 外部输出 SPDX SBOM。
- 未机器确认的许可证结论使用 `NOASSERTION`，不会猜测授权。
- Release Channel 按 `dev → beta → rc → stable` 受控推进。

### 仍受保护 / 尚未完成的门禁

当前明确**不宣称已经完成**：

- 生产 Secure Boot 签名与离线 Release Key 托管；
- 破坏性写盘安装执行；
- 生产环境 A/B Slot 激活与自动回滚；
- 完整 Recovery Environment 验证；
- 最终 TUF Root / Threshold Key 操作；
- 完整 CVE 自动化与最终全球法律审核；
- GitHub Secrets 中的 R2 大文件发布凭据；
- 具体硬件的 IoT / Edge Device Pack；
- Stable 长期支持生命周期承诺。

在这些门禁完成并验证前，RC 与 Stable 发布保持阻断。
