# KINGAI OS

> **Sovereign Distributed Intelligence Operating System**
>
> AI-native · Agent-native · Local-first · Secure-by-default · Model-neutral · Cloud-neutral

**Status:** D4 Developer Foundation / Pre-Alpha  
**Official OS site:** `https://os.kingai.work`  
**Repository:** `https://github.com/kingaiwork/KINGAIOS`

## Official Sources

🌐 **KINGAI OS Website:** https://os.kingai.work  
🌐 **KINGAI Main Website:** https://www.kingai.work  
📚 **Full Documentation:** [docs/INDEX.md](docs/INDEX.md)  
🧠 **Machine-Readable Facts:** [llms.txt](llms.txt)  
🔎 **GEO Knowledge Graph:** [seo/GEO-KNOWLEDGE-GRAPH.md](seo/GEO-KNOWLEDGE-GRAPH.md)  
🧬 **Architecture:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)  
🖥️ **Desktop Architecture:** [docs/DESKTOP.md](docs/DESKTOP.md)  
🛡️ **Security:** [SECURITY.md](SECURITY.md)  
✉️ **Business & Partnership:** vip@kingai.work

---

# English

## The AI-native operating system for the next generation of intelligent computing

KINGAI OS is a long-term operating-system project from KINGAI, designed to deeply integrate intelligence, memory, agents, policy, security, execution, model routing, updates and device management into one coherent Linux platform.

It integrates advanced technologies from the global Linux, AI, cloud-native, cybersecurity and software-supply-chain ecosystems while preserving a sovereign KINGAI architecture that is not locked to any single model provider, cloud vendor, agent framework or execution engine.

> **Intelligence may be autonomous. Authority remains controlled, auditable and revocable.**

## Editions

### KINGAI OS Server

Headless edition for VPS, servers, AI nodes, enterprise automation and distributed agent infrastructure.

### KINGAI OS Desktop

One optimized Desktop ISO with a first-run visual experience selector:

- **KINGAI Intelligence** — KINGAI's AI-first desktop centered on agents, memory, tasks, projects, knowledge and automation.
- **KINGAI Flow** — a clean modern spatial workflow with dock-oriented interaction.
- **KINGAI Classic** — a familiar taskbar and application-menu workflow for traditional PC users.

Desktop experiences share one common Desktop Core and can be changed later without reinstalling the operating system.

### KINGAI OS IoT / Edge

Minimal edition for ARM64/x86-64 edge systems, robotics, gateways, embedded AI and intelligent devices.

## D4 Architecture

```text
User / Organization
        │
        ▼
KINGAI Intelligence
Brain · Planner · Memory · Knowledge · Evolution
        │
        ▼
KINGAI Governance
Policy · Capability · Risk · Identity · Audit
        │
        ▼
Execution Fabric
Native · OpenClaw · MCP · Codex · Browser · Containers · VM
        │
        ▼
KINGAI Secure Core
systemd · cgroup v2 · AppArmor · seccomp · Landlock
        │
        ▼
Linux Kernel
```

## Core principles

- **Local-first** — core intelligence can continue operating without cloud connectivity.
- **Agent-native** — agents are first-class system actors governed by explicit capabilities and policy.
- **Model-neutral** — local and cloud models are interchangeable through KINGAI Model Fabric.
- **Cloud-neutral** — cloud infrastructure enhances the system but is not required for local survival.
- **Least privilege** — AI agents never receive unrestricted authority by default.
- **Controlled evolution** — autonomous improvements pass sandboxing, tests, policy and deployment gates.
- **Secure updates** — official releases are designed for signatures, SBOM, provenance, integrity verification and rollback.
- **Privacy-first memory** — sensitive memory can remain local with explicit synchronization policy.
- **Replaceable execution engines** — OpenClaw, MCP, Codex, browsers and other runtimes remain adapters rather than the identity of KINGAI OS.

## KINGAI Intelligence Layer

```text
KINGAI Brain
├── Planner
├── Memory Fabric
├── Agent Manager
├── Model Fabric
├── Knowledge
├── Task Graph
└── Controlled Evolution
```

The goal is not to place an AI chatbot on top of Linux. The goal is to make intelligence a governed operating-system capability.

## Agent security

Agents operate through capabilities instead of unrestricted root shells.

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

High-impact actions are evaluated through KINGAI Policy and Capability Broker before privileged execution.

## Model Fabric

KINGAI OS is designed to support local, cloud, hybrid and offline-fallback models without binding the system to one vendor.

Routing can consider capability, privacy, latency, cost, availability, context length, region, trust and license status.

Large third-party model weights are not embedded in the default ISO unless redistribution rights are explicitly verified.

## Distribution targets

```text
KINGAI-OS-Server-<version>-amd64.iso
KINGAI-OS-Server-<version>-arm64.iso
KINGAI-OS-Desktop-<version>-amd64.iso
KINGAI-OS-IoT-<version>-arm64.img.xz
```

Engineering size targets:

- Server: **~1.2–1.6 GB**
- Desktop: **~2.8–3.5 GB**
- IoT / Edge compressed image: **~0.4–0.7 GB**

Actual release sizes are measured by the build pipeline.

## Downloads and large artifacts

GitHub is the source-code and release-metadata home.

```text
<= 2 GiB  → GitHub Releases when appropriate
>  2 GiB  → KINGAI object storage / Cloudflare R2
```

Public distribution is unified through:

```text
https://os.kingai.work/download
https://os.kingai.work/updates
https://os.kingai.work/repo
```

Official images are designed to ship with checksums, signatures, SBOM and build provenance.

## Release channels

```text
nightly → dev → beta → rc → stable
```

Stable releases must pass build, security, install, upgrade, rollback and recovery gates before signing.

## Current milestone

**KINGAI OS 0.1 Developer Foundation** is establishing:

- shared Server / Desktop / IoT build profiles;
- KINGAI CLI and core runtime foundation;
- agent capability and policy architecture;
- model-neutral and local-first interfaces;
- secure release pipeline;
- optimized image generation;
- large-artifact R2 publishing;
- KINGAI Desktop Experience framework.

This repository is under active development and does not yet represent a production-ready Stable ISO.

---

# 中文

## 官方入口

🌐 **KINGAI OS 官网：** https://os.kingai.work  
🌐 **KINGAI 主站：** https://www.kingai.work  
📚 **完整文档：** [docs/INDEX.md](docs/INDEX.md)  
🧠 **机器可读事实：** [llms.txt](llms.txt)  
🔎 **GEO 知识图谱：** [seo/GEO-KNOWLEDGE-GRAPH.md](seo/GEO-KNOWLEDGE-GRAPH.md)  
🧬 **系统架构：** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)  
🖥️ **桌面架构：** [docs/DESKTOP.md](docs/DESKTOP.md)  
🛡️ **安全：** [SECURITY.md](SECURITY.md)  
✉️ **商务与合作：** vip@kingai.work

## 面向下一代智能计算的 AI 原生操作系统

KINGAI OS 是 KINGAI 面向未来长期发展的 AI 原生操作系统项目，将智能、记忆、智能体、权限治理、安全、执行、模型路由、系统更新与设备管理深度整合为统一 Linux 平台。

项目持续整合全球 Linux、人工智能、云原生、网络安全和软件供应链领域成熟先进的开放技术，同时保持 KINGAI 自主架构，不被任何单一模型厂商、云平台、智能体框架或执行引擎绑定。

> **智能可以自主，权限必须始终可控、可审计、可撤销。**

## 三个正式版本

### KINGAI OS Server

无桌面版本，面向 VPS、服务器、AI 节点、企业自动化和分布式智能体基础设施。

### KINGAI OS Desktop

只维护一个优化后的 Desktop ISO。第一次进入系统时提供可视化桌面体验展示，由用户选择：

- **KINGAI Intelligence** — KINGAI 原生 AI 桌面，以智能体、记忆、任务、项目、知识和自动化为核心。
- **KINGAI Flow** — 现代化空间工作流和 Dock 风格交互。
- **KINGAI Classic** — 面向传统 PC 用户的任务栏与应用菜单工作流。

三个体验共享同一套 KINGAI Desktop Core，进入系统后仍可自由切换，无需重新安装系统。

### KINGAI OS IoT / Edge

面向 ARM64/x86-64 IoT、机器人、边缘网关、嵌入式 AI 和智能终端的精简版本。

## D4 核心架构

```text
用户 / 企业
    │
    ▼
KINGAI Intelligence
Brain · Planner · Memory · Knowledge · Evolution
    │
    ▼
KINGAI Governance
Policy · Capability · Risk · Identity · Audit
    │
    ▼
Execution Fabric
Native · OpenClaw · MCP · Codex · Browser · Containers · VM
    │
    ▼
KINGAI Secure Core
systemd · cgroup v2 · AppArmor · seccomp · Landlock
    │
    ▼
Linux Kernel
```

## 核心原则

- **Local-first**：即使云端不可用，核心智能仍可在本机运行。
- **Agent-native**：智能体成为操作系统原生能力，并受 Policy 与 Capability 管理。
- **Model-neutral**：本地模型与云模型通过 KINGAI Model Fabric 自由替换。
- **Cloud-neutral**：云端用于增强，而不是系统生存依赖。
- **Least privilege**：AI 默认不拥有无限 root 权限。
- **Controlled evolution**：自主升级必须经过沙箱、测试、策略和发布门禁。
- **Secure updates**：正式发行支持签名、SBOM、Provenance、完整性检查与回滚。
- **Privacy-first memory**：敏感记忆默认可以只保存在本地。
- **执行引擎可替换**：OpenClaw、MCP、Codex、Browser 等只是 Adapter。

## 镜像目标

```text
KINGAI-OS-Server-<version>-amd64.iso
KINGAI-OS-Server-<version>-arm64.iso
KINGAI-OS-Desktop-<version>-amd64.iso
KINGAI-OS-IoT-<version>-arm64.img.xz
```

工程体积目标：

- Server：**约 1.2–1.6 GB**
- Desktop：**约 2.8–3.5 GB**
- IoT / Edge 压缩镜像：**约 0.4–0.7 GB**

最终以自动构建后的实际体积为准。

## ISO 与大型文件

GitHub 主要保存源码、构建系统、文档和版本元数据。

```text
<= 2 GiB  → 按需要使用 GitHub Releases
>  2 GiB  → KINGAI 对象存储 / Cloudflare R2
```

用户统一通过：

```text
https://os.kingai.work/download
https://os.kingai.work/updates
https://os.kingai.work/repo
```

获取正式镜像和更新。

## 当前阶段

**KINGAI OS 0.1 Developer Foundation** 正在建立统一构建 Profile、KINGAI CLI、Agent Capability/Policy、模型中立接口、安全发布体系、精简镜像、R2 大文件发布以及 KINGAI Desktop Experience 框架。

当前仍属于开发基础阶段，尚未宣称存在生产级 Stable ISO。

---

**KINGAI OS — Intelligence that can act, under authority that remains controlled.**

> **Base notice / 底层说明：** KINGAI OS is independently developed on the Ubuntu 26.04 LTS technology base and is distributed in accordance with applicable open-source licenses and trademark rules. / KINGAI OS 基于 Ubuntu 26.04 LTS 技术底座进行独立二次开发与发行，并遵守适用的开源许可证与商标规则。
