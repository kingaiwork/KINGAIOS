# KINGAI OS

> **Sovereign Distributed Intelligence Operating System**
>
> AI-native. Agent-native. Local-first. Secure by default. Model-neutral. Cloud-neutral. Built for long-term global deployment.

**Project status:** Developer Foundation / Pre-Alpha  
**Primary site:** `https://os.kingai.work`  
**Source:** `https://github.com/kingaiwork/KINGAIOS`  
**Base generation:** Ubuntu 26.04 LTS source lineage + KINGAI-owned build, policy, intelligence, security, update and distribution layers.

---

## English

### Vision

KINGAI OS is a long-term operating-system project from KINGAI. It is designed as an AI-native Linux platform where intelligence, memory, agents, policy, security, execution, model routing, updates and device management are first-class operating-system capabilities rather than loosely connected applications.

The project integrates advanced open technologies from the global Linux, cloud-native, AI, security and software-supply-chain ecosystems while keeping the KINGAI architecture independent from any single model provider, cloud vendor, agent framework or execution engine.

KINGAI OS is designed around a simple rule:

> **Intelligence may be autonomous, but authority must remain controlled, auditable and revocable.**

### Product editions

KINGAI OS uses one shared core and three official distribution profiles:

- **KINGAI OS Server** — headless server, VPS, AI node and enterprise agent runtime.
- **KINGAI OS Desktop** — desktop edition with a first-run visual choice between KINGAI Intelligence, KINGAI Flow and KINGAI Classic experiences. The desktop experience can be changed later without reinstalling the OS.
- **KINGAI OS IoT / Edge** — minimal image for ARM64/x86-64 edge devices, robotics, gateways and embedded AI systems.

### Desktop experiences

The Desktop edition ships one shared desktop core with multiple visual/interaction profiles:

- **KINGAI Intelligence** — the native AI-first workspace centered on agents, memory, tasks, knowledge, projects and automation.
- **KINGAI Flow** — a clean, dock-oriented workflow designed for users who prefer a modern spatial desktop model.
- **KINGAI Classic** — a taskbar/application-menu workflow designed for users who prefer a traditional PC desktop model.

These are KINGAI product experiences. Third-party trademarks, icons and proprietary visual assets are not used as KINGAI branding.

### Core architecture

```text
User / Organization
        |
        v
KINGAI Intelligence
        |
        +-- Brain / Planner
        +-- Memory Fabric
        +-- Agent Manager
        +-- Model Fabric
        +-- Knowledge
        +-- Controlled Evolution
        |
        v
KINGAI Governance
        |
        +-- Policy Engine
        +-- Capability Broker
        +-- Risk Engine
        +-- Identity / Secrets
        +-- Audit / Compliance
        |
        v
Execution Fabric
        |
        +-- Native Runtime
        +-- OpenClaw Adapter
        +-- MCP Adapter
        +-- Codex Adapter
        +-- Browser Adapter
        +-- Rootless Containers
        +-- Isolated VM / MicroVM backends
        |
        v
KINGAI Secure System Core
        |
        +-- systemd / cgroup v2
        +-- AppArmor / seccomp / Landlock
        +-- Secure Boot / TPM integration
        +-- signed updates / rollback
        |
        v
Linux Kernel
```

### Architectural principles

1. **Local-first:** core functions continue to operate without the cloud.
2. **Model-neutral:** local and remote models are connected through a provider-neutral model fabric.
3. **Cloud-neutral:** Cloudflare may be used as the first control-plane implementation, but the OS must not depend on it for survival.
4. **Agent-native:** agents are managed through policy and capabilities, not unrestricted root shells.
5. **Least privilege:** an agent cannot grant itself higher authority.
6. **Immutable-core direction:** production images are designed for verifiable, rollback-friendly system updates.
7. **Privacy by default:** sensitive memory can remain local and cloud synchronization is explicit and policy-controlled.
8. **Supply-chain security:** releases are designed to carry checksums, signatures, SBOM and build provenance.
9. **Controlled evolution:** autonomous improvement must pass sandboxing, testing, policy evaluation and deployment gates.
10. **Replaceable dependencies:** OpenClaw, model vendors, cloud providers and execution engines are adapters, not the identity of KINGAI OS.

### AI model strategy

Large third-party model weights are intentionally not embedded in the default ISO unless redistribution rights are explicitly verified.

The OS ships the model-management and routing layer. Models may be configured after installation as:

- Local
- Cloud
- Hybrid
- Offline fallback

Routing policies can consider privacy, cost, latency, capability, availability, context length, jurisdiction and license status.

### Security model

KINGAI OS uses capability-oriented execution. Instead of giving an AI agent unrestricted `sudo`, privileged actions pass through policy-controlled capabilities such as:

```text
filesystem.read
filesystem.write
process.execute
service.restart
package.install
network.modify
security.modify
boot.modify
disk.raw
```

Risk levels are designed from read-only operations through owner-only trust-root operations. High-impact authority remains explicit and auditable.

### Distribution and large files

Source code, build definitions, documentation and release metadata live in GitHub.

GitHub Release assets are limited per file, so large operating-system images are published through KINGAI-controlled object storage. The initial large-object backend is Cloudflare R2 using its S3-compatible multipart API.

Planned public paths:

```text
https://os.kingai.work/download
https://os.kingai.work/updates
https://os.kingai.work/repo
https://os.kingai.work/security
https://os.kingai.work/docs
```

Artifact policy:

- `<= 2 GiB`: eligible for GitHub Releases when appropriate.
- `> 2 GiB`: publish to the KINGAI R2 release bucket and expose through `os.kingai.work`.
- checksums, signatures, SBOM and provenance accompany official images.

### Target image families

```text
KINGAI-OS-Server-<version>-amd64.iso
KINGAI-OS-Server-<version>-arm64.iso
KINGAI-OS-Desktop-<version>-amd64.iso
KINGAI-OS-IoT-<version>-arm64.img.xz
```

Initial engineering size goals:

- Server: approximately 1.2–1.6 GB
- Desktop: approximately 2.8–3.5 GB
- IoT/Edge compressed image: approximately 0.4–0.7 GB

These are engineering targets, not release guarantees. CI will record actual image sizes.

### Repository layout

```text
base/           upstream source manifests and package policy
boot/           verified boot, Secure Boot, TPM and integrity work
core/           KINGAI system daemons and shared core
intelligence/   brain, planner, memory, knowledge and evolution
agents/         agent runtime, permissions and registry
models/         provider-neutral model fabric
runtime/        process, container and isolated execution backends
adapters/       OpenClaw, MCP, Codex, browser and provider adapters
security/       policy, sandbox, audit and hardening
profiles/       Server, Desktop and IoT/Edge build profiles
desktop/        shared desktop core and experience profiles
distro/         rootfs, installer and image generation
update/         signing, TUF metadata and rollback framework
cloud/          optional control-plane interfaces
scripts/        build, validation and publishing helpers
docs/           architecture, security, legal and roadmap documents
legal/          license manifests, notices and compliance metadata
.github/        CI/CD and release automation
```

### Release channels

```text
nightly -> dev -> beta -> rc -> stable
```

A stable release must pass build, security, license, upgrade, rollback, clean-install and recovery gates before signing.

### Legal and compliance direction

KINGAI OS is a KINGAI distribution and technology platform, but upstream software remains under its respective copyright and license terms.

The project is designed to maintain:

- upstream source and patch traceability;
- license and copyright inventories;
- corresponding-source workflows where required;
- third-party model redistribution checks;
- software bill of materials (SBOM);
- vulnerability and security-advisory processes;
- privacy/export/delete controls;
- AI transparency and audit metadata;
- release signing and provenance.

Formal global commercial release will require jurisdiction-specific legal review. Engineering controls in this repository are intended to make that review repeatable and auditable.

### Current milestone

The repository is currently establishing the **D4 Developer Foundation**:

- repository and documentation baseline;
- distribution profiles;
- reproducible image-build skeleton;
- R2-aware artifact publishing;
- security and policy architecture;
- desktop-experience framework;
- legal/compliance gates;
- CI release pipeline.

The presence of build scaffolding does **not** mean a production-ready KINGAI OS ISO has already been released.

---

# 中文

## 项目愿景

KINGAI OS 是 KINGAI 面向未来长期发展的 AI 原生操作系统项目。它不是“Ubuntu 安装几个 AI 软件”，而是把智能、记忆、智能体、权限治理、模型路由、安全、执行、更新和设备管理设计为操作系统的一等能力。

项目将持续整合全球 Linux、AI、云原生、安全、软件供应链等领域成熟且先进的开放技术，同时保证 KINGAI 的核心架构不被任何单一模型厂商、云服务商、智能体框架或执行引擎绑定。

KINGAI OS 的核心原则：

> **智能可以自主，但权限必须始终可控、可审计、可撤销。**

## 三个正式版本

KINGAI OS 使用同一套核心代码，建立三个官方发行 Profile：

- **KINGAI OS Server**：无桌面服务器、VPS、AI 节点、企业智能体运行环境。
- **KINGAI OS Desktop**：桌面版本。首次进入系统通过可视化展示选择 KINGAI Intelligence、KINGAI Flow 或 KINGAI Classic，进入系统后仍可随时切换，不需要重装。
- **KINGAI OS IoT / Edge**：面向 ARM64/x86-64 IoT、机器人、边缘网关和嵌入式 AI 设备的精简版本。

## KINGAI 原生桌面

Desktop 只维护一套 Desktop Core，并提供三种 KINGAI 自有桌面体验：

- **KINGAI Intelligence**：以 AI、智能体、记忆、任务、知识、项目和自动化为核心的 KINGAI 原生 AI 桌面。
- **KINGAI Flow**：现代化 Dock/空间工作流体验。
- **KINGAI Classic**：传统任务栏、应用菜单和 PC 工作流体验。

三种模式共享底层组件，不安装三套完整桌面环境，从而降低 ISO 大小、内存占用和长期维护成本。

## 智能体与执行层

OpenClaw、Codex、MCP、浏览器控制和其他执行引擎均作为 Adapter 接入 KINGAI，而不是 KINGAI OS 的根依赖。

真正长期保持不变的是：

```text
KINGAI Intelligence
KINGAI Memory
KINGAI Governance
KINGAI Agent Runtime
KINGAI Security
KINGAI Trust / Update Chain
KINGAI Desktop Experience
```

因此未来即使替换模型、OpenClaw、Cloudflare 或其他供应商，KINGAI OS 仍然可以继续演进。

## 安全设计

KINGAI OS 不把 AI 直接等同于 root。智能体通过 Capability Broker 申请受策略控制的能力，关键操作经过风险分级、审计和必要的人工批准。

正式稳定版方向包括：

- systemd + cgroup v2
- AppArmor
- seccomp
- Landlock
- rootless containers
- Secure Boot / TPM
- 可验证系统镜像
- 签名更新
- A/B 更新与自动回滚
- 软件供应链 SBOM / provenance

## 模型策略

默认 ISO 不直接打包没有明确再发行授权的大型第三方模型权重。

系统内置 Model Manager、Model Router 和 Provider Adapter；安装后用户可以选择：

- 云模型
- 本地模型
- 混合模式
- 离线备用

路由策略可根据隐私、成本、速度、能力、上下文、可用性、地区和许可证条件动态选择模型。

## ISO 与大文件发布

GitHub 保存：

- 源码
- 构建脚本
- 文档
- Release Notes
- 小型发布资产
- 校验与元数据

超过 GitHub 单个 Release Asset 限制的大型 ISO/IMG 使用 KINGAI 管理的对象存储发布。第一阶段采用 Cloudflare R2 S3 兼容 Multipart Upload。

规则：

```text
<= 2 GiB  -> 可按需要发布到 GitHub Releases
>  2 GiB  -> 发布到 KINGAI R2，并通过 os.kingai.work 提供下载
```

用户统一访问：

```text
os.kingai.work/download
os.kingai.work/updates
os.kingai.work/repo
```

## 初始体积目标

- Server：约 1.2–1.6 GB
- Desktop：约 2.8–3.5 GB
- IoT/Edge 压缩镜像：约 0.4–0.7 GB

这些是工程目标，不是未经构建验证的宣传数字。CI 会记录每次实际构建体积。

## 法律与全球发行

KINGAI OS 是 KINGAI 自己的发行版和技术平台，但 Linux 内核以及第三方软件仍然分别遵守其原始版权和许可证。

项目将把法律与合规直接纳入构建和发布流程，包括：

- 上游源码与补丁可追溯；
- 自动许可证清单；
- 必要的对应源码提供机制；
- 第三方模型再发行权限检查；
- SBOM；
- 安全漏洞与公告机制；
- 隐私数据导出/删除能力；
- AI 透明度与审计元数据；
- Release 签名和构建 provenance。

全球正式商业发行前仍需按照实际发行国家、功能和商业模式完成专业法律审查。

## 当前阶段

当前仓库正在建立 **KINGAI OS D4 Developer Foundation**，重点是把长期架构、构建、发行、安全、法律和智能体边界一次设计正确。

当前阶段不对外宣称已经存在生产级稳定 ISO；完成可重复构建、测试、签名、升级和回滚验证后，再进入 Alpha / Beta / RC / Stable。

---

## Roadmap

See `docs/ROADMAP.md`.

## Architecture

See `docs/ARCHITECTURE.md`.

## Security

See `SECURITY.md`.

## Legal / Compliance

See `docs/LEGAL.md`.

---

**KINGAI OS** — Intelligence that can act, under authority that remains controlled.
