# KINGAI OS Roadmap

**Revision:** 2026-08-14  
**Current line:** D5 Alpha Runtime Foundation / Pre-Alpha

KINGAI OS uses one shared intelligence/governance/runtime core across four official distribution forms:

1. **KINGAI OS Server** — servers, VPS, AI nodes and enterprise automation;
2. **KINGAI OS Desktop** — personal computers and workstations;
3. **KINGAI OS IoT / Edge** — embedded, gateways, robotics and edge intelligence;
4. **KINGAI OS Container** — Docker/OCI deployments for cloud, CI, homelab and service composition.

The project is gate-driven: a feature is not considered production-ready merely because source code exists.

## Phase 0 — D4 Developer Foundation — substantially complete

Foundation work established:

- repository architecture and policy baseline;
- Server / Desktop / IoT profiles;
- Ubuntu 26.04-based rootfs/image pipelines;
- `kingai`, `kingaid`, installer, updater and recovery foundations;
- model-neutral routing interfaces;
- local memory store;
- agent identity and capability policy;
- audit logging;
- SPDX SBOM and package inventory;
- release gate ledger;
- installer, recovery, A/B update, TUF and Secure Boot VM test tracks;
- Plasma 6 Desktop Core and three desktop experiences.

## Phase 1 — D5 Runtime Foundation — current

Goal: connect the isolated foundations into one governed local intelligence runtime.

### Completed in the current D5 line

- Approval Broker persistence;
- approval expiry and one-time consumption;
- Agent + Capability + Target Hash + Peer UID binding;
- Task Graph persistence and state machine;
- peer-isolated Task ownership;
- Memory Store wired into `kingaid`;
- Model Router wired into `kingaid`;
- CLI commands for Approval / Memory / Model / Task;
- D5 runtime status reporting;
- Docker/OCI Container profile and image build path;
- Container runtime smoke test in CI;
- Server/Desktop/IoT profiles aligned to the shared D5 runtime.

### Remaining before D5 Runtime Alpha can be called feature-complete

- constrained Execution Broker with no generic unrestricted root shell;
- production-quality sandbox profiles using AppArmor/seccomp/Landlock/cgroup controls;
- Planner and Task Step scheduler;
- capability-scoped execution result model;
- OpenClaw adapter;
- MCP adapter;
- Codex-compatible tool adapter;
- browser execution adapter;
- provider adapters and model health probes;
- richer Memory metadata, retrieval and promotion;
- end-to-end execution fault injection.

## Phase 2 — 0.3 Agent Runtime Alpha

Target: a usable local Agent Runtime where real tasks can move through a controlled lifecycle.

```text
Goal
 -> Plan
 -> Task Graph
 -> Capability Policy
 -> Approval when required
 -> Constrained Executor
 -> Result
 -> Audit
 -> Memory
```

Required gates:

- no arbitrary privileged shell interface;
- capability-specific executor handlers;
- sandbox and resource limits;
- task cancellation and timeout;
- execution provenance;
- adapter health and lifecycle management;
- failure-safe policy defaults;
- runtime integration tests on Server, Desktop and Container.

## Phase 3 — 0.5 Desktop Alpha

Desktop is the **personal-computer edition**; there is no separate PC profile.

Target experience:

- KINGAI Welcome first-run setup;
- KINGAI Intelligence / Flow / Classic switching;
- Agent Center;
- Task Center;
- Approval Center;
- Memory Center;
- Model Center;
- Privacy controls;
- local automation controls;
- device and update status;
- system health intelligence;
- native notifications for approval and task state.

Desktop remains one shared Plasma 6 / Wayland core rather than multiple divergent desktop distributions.

## Phase 4 — 0.6 Server / Container Alpha

### Server

- headless agent orchestration;
- service health and automation;
- rootless execution backends;
- server policy templates;
- SSH/cloud-init deployment flows;
- optional local model service;
- fleet enrollment foundation.

### Container

- amd64/arm64 Docker/OCI images;
- non-root daemon by default;
- persistent state volumes;
- Unix-socket local management;
- Kubernetes-compatible runtime pattern without making Kubernetes mandatory;
- Compose examples;
- GHCR developer publishing workflow;
- SBOM/provenance for container artifacts;
- health/readiness behavior.

Container is an execution form of KINGAI OS Runtime, not a replacement for the bootable Server/Desktop/IoT operating-system editions.

## Phase 5 — 0.7 IoT / Edge Alpha

- generic amd64/arm64 Edge runtime;
- Device Pack contract;
- hardware identity;
- offline/local-first model modes;
- constrained device capabilities;
- OTA update policy;
- recovery behavior;
- Raspberry Pi validated pack;
- Jetson validated pack;
- selected industrial x86/ARM pack;
- telemetry with explicit privacy policy.

A generic image does not become “hardware-supported” until its Device Pack is boot-validated on that hardware family.

## Phase 6 — 0.8 Trust, Update and Fleet Beta

- production TUF repository layout;
- threshold-key operations;
- offline release-key custody procedure;
- production Secure Boot signing track;
- TPM-backed identity where available;
- signed update metadata;
- A/B or equivalent atomic update semantics;
- automatic rollback health gates;
- recovery environment;
- controlled fleet update rollout;
- Cloudflare/R2 or replaceable artifact delivery backend;
- update failure injection at scale.

Cloud remains an accelerator and management option, not a survival dependency for local KINGAI runtimes.

## Phase 7 — 0.9 Release Candidate

Before RC:

- clean-install validation;
- upgrade and rollback validation;
- recovery validation;
- SBOM and provenance for every official artifact type;
- corresponding-source publication workflow;
- vulnerability/CVE automation;
- security advisory process;
- release signing ceremony and documentation;
- privacy export/delete controls;
- public hardware compatibility matrix;
- protected branch/release governance;
- release artifact delivery validation;
- legal and license review.

## Phase 8 — 1.0 Stable

Stable is released only when required security, licensing, update, rollback, recovery, signing, reproducibility and governance gates are satisfied.

Planned stable artifact families:

```text
KINGAI-OS-Server-<version>-amd64.iso
KINGAI-OS-Desktop-<version>-amd64.iso
KINGAI-OS-IoT-<version>-<arch>.img.xz
OCI: ghcr.io/kingaiwork/kingai-os:<version>
```

Additional architectures are published only after their own validation gates pass.

## After 1.0

- ARM64 Desktop when real hardware validation is sufficient;
- more IoT/Edge Device Packs;
- organization memory and policy federation;
- distributed fleet management;
- optional remote attestation;
- signed Agent Marketplace manifests;
- advanced scheduling;
- GPU/NPU acceleration policy;
- enterprise support lifecycle;
- multi-node cooperative intelligence;
- controlled evolution pipelines with staged verification and rollback.

---

# 中文路线图

KINGAI OS 正式统一为四种发行形态，共享同一套核心：

```text
KINGAI Core
├── Intelligence
├── Agent Runtime
├── Policy / Approval
├── Task Graph
├── Memory Fabric
├── Model Fabric
├── Audit
├── Update / Recovery
└── Security
        │
        ├── Server
        ├── Desktop（个人电脑）
        ├── IoT / Edge
        └── Container（Docker / OCI）
```

## 当前阶段：D5 Alpha Runtime Foundation

当前重点已经从“把系统做出来”转向“把智能运行闭环接起来”。

已完成的 D5 基础：

- Approval Broker；
- 一次性审批凭证；
- Task Graph；
- Memory 接入核心守护进程；
- Model Router 接入核心守护进程；
- CLI 管理入口；
- Server/Desktop/IoT Profile 对齐；
- Docker/OCI Container Profile；
- Container Dockerfile 与 Buildx 构建脚本；
- Container CI 启动与运行时 Smoke Test。

## Desktop 就是个人电脑版本

不会另外维护 PC Profile。

正式名称仍然是：

**KINGAI OS Desktop**

定位覆盖：

- 家庭电脑；
- 办公电脑；
- AI 工作站；
- 开发者电脑；
- 创作者电脑；
- 本地模型电脑。

## 四个平台的目标

### Server

服务器、VPS、企业自动化、AI 节点、分布式 Agent、Fleet。

### Desktop

个人电脑与工作站，提供 KINGAI Intelligence / Flow / Classic 三套体验，共享一个 Desktop Core。

### IoT / Edge

ARM64/x86-64 网关、机器人、边缘设备与嵌入式智能。具体硬件必须通过 Device Pack 验证后才算正式支持。

### Container

Docker/OCI 云端和本地容器运行环境，支持 amd64/arm64 构建，默认非 root daemon，保留 KINGAI Policy、Approval、Task、Memory、Model 和 Audit 核心。

## 下一个关键里程碑：0.3 Agent Runtime Alpha

必须真正完成：

```text
目标
 ↓
Planner
 ↓
Task Graph
 ↓
Policy
 ↓
Approval
 ↓
受限 Execution Broker
 ↓
Sandbox
 ↓
结果
 ↓
Audit
 ↓
Memory
```

Execution Broker 不允许设计成“AI 获得无限制 root shell”。所有高权限操作必须映射成明确 Capability，并具备超时、资源限制、审计和可撤销控制。

## 1.0 前必须完成

- 生产 TUF；
- 生产签名；
- Secure Boot 正式密钥流程；
- A/B 更新与回滚；
- Recovery；
- 全 Artifact SBOM/Provenance；
- CVE 自动化；
- 分支与 Release Governance；
- R2/对象存储正式交付；
- 硬件兼容矩阵；
- 法律与许可证审核。

只有全部必需门禁通过后，KINGAI OS 才进入 Stable。
