# KINGAI OS

> **Sovereign Distributed Intelligence Operating System**  
> **主权式分布智能操作系统**

AI-native · Agent-native · Local-first · Secure-by-default · Model-neutral · Cloud-neutral

**Development line:** D5 Alpha Runtime Foundation / Pre-Alpha  
**Source version:** `0.1.0-dev`  
**Official site:** `https://os.kingai.work`  
**KINGAI:** `https://www.kingai.work`

KINGAI OS is a long-term AI-native operating-system project that moves **agents, memory, tasks, model routing, policy, approvals, audit, recovery and controlled execution** into the operating-system architecture itself.

KINGAI OS 不是在 Linux 上简单放一个聊天机器人，而是把 **智能体、记忆、任务、模型路由、权限治理、审批、审计、恢复与受控执行** 逐步建设成系统原生能力。

> **Intelligence may be autonomous. Authority remains controlled, auditable and revocable.**  
> **智能可以自主，权限必须始终可控、可审计、可撤销。**

## Official documentation / 官方资料

- [Verified Status / 已验证状态](docs/STATUS.md)
- [Architecture / 系统架构](docs/ARCHITECTURE.md)
- [Platform Editions / 四个平台](docs/PLATFORMS.md)
- [Roadmap / 路线图](docs/ROADMAP.md)
- [Building / 构建说明](docs/BUILDING.md)
- [Desktop Architecture](docs/DESKTOP.md)
- [Device Packs](docs/DEVICE-PACKS.md)
- [Release Policy](docs/RELEASE-POLICY.md)
- [Security](SECURITY.md)
- [Machine-readable facts](llms.txt)
- [GEO Knowledge Graph](seo/GEO-KNOWLEDGE-GRAPH.md)

Business & partnership: `vip@kingai.work`

---

# English

## 1. What KINGAI OS is

KINGAI OS is a Linux-based intelligent operating-system architecture developed by KINGAI. Its goal is to make intelligence a governed system capability rather than an unrestricted application sitting above the operating system.

A KINGAI task is intended to move through a controlled lifecycle:

```text
Goal
  ↓
Planner
  ↓
Task Graph
  ↓
Capability Policy
  ↓
Approval when required
  ↓
Constrained Execution
  ↓
Result / Health Check
  ↓
Audit
  ↓
Memory
```

The current D5 line has connected the Policy, Approval, Task, Memory, Model and Audit foundations. A privileged constrained Execution Broker is the next protected runtime milestone and is not claimed production-complete yet.

## 2. One core, four official platform editions

KINGAI OS does not maintain four unrelated products. All editions share the same KINGAI governance and intelligence core.

```text
                        KINGAI Core
              Intelligence · Governance · Runtime
      Policy · Approval · Task · Memory · Model · Audit
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
       Server              Desktop             IoT/Edge
          │                   │                   │
          └───────────────────┴──────────────┬────┘
                                             │
                                          Container
                                        Docker / OCI
```

### KINGAI OS Server

For VPS, physical servers, AI nodes, enterprise automation and distributed agent infrastructure.

Profile: `profiles/server.yaml`

Current direction:

- headless by default;
- amd64 and arm64 rootfs paths;
- SSH/cloud-init friendly;
- optional local AI runtime;
- rootless container support direction;
- shared Policy / Approval / Task / Memory / Model / Audit runtime;
- installer, update, rollback and recovery engineering tracks.

The already-published Server Developer Preview is amd64. Additional architecture publication is gated separately from rootfs build support.

### KINGAI OS Desktop

**Desktop is the personal-computer / PC edition. There is no separate PC profile.**

Profile: `profiles/desktop.yaml`

For home PCs, office PCs, developer machines, creator workstations and local-AI workstations.

Current Desktop Core:

- Plasma 6;
- KWin Wayland;
- SDDM;
- Qt 6;
- KINGAI Welcome;
- KINGAI Agent Center.

Three switchable experiences share one Desktop Core:

1. **KINGAI Intelligence** — AI-first workspace centered on tasks, agents, memory, models, knowledge and automation;
2. **KINGAI Flow** — modern spatial workflow and dock-oriented interaction;
3. **KINGAI Classic** — familiar taskbar/application-menu workflow.

Current Desktop architecture target is amd64. ARM64 Desktop remains a future hardware-validation track.

### KINGAI OS IoT / Edge

For gateways, edge AI, embedded devices, robotics and intelligent hardware.

Profile: `profiles/iot.yaml`

Generic architecture build paths:

- arm64;
- amd64.

The Edge edition keeps the same governance concepts while reducing the OS footprint. Hardware support is deliberately separated from generic architecture support: Raspberry Pi, Jetson and industrial boards require validated Device Packs before they are listed as officially supported hardware.

### KINGAI OS Container

For Docker/OCI, CI, cloud services, homelab environments and service composition.

Profile: `profiles/container.yaml`  
Dockerfile: `container/Dockerfile`

Current Container design:

- linux/amd64 and linux/arm64 Buildx targets;
- no systemd requirement in the image;
- `kingaid` runs directly as container entrypoint;
- non-root `_kingai` daemon by default;
- Unix-socket management API;
- no management TCP port exposed by default;
- persistent `/var/lib/kingai` and `/var/log/kingai` volume targets;
- Policy, Approval, Task Graph, Memory, Model Router and Audit remain available.

Container is a deployment form of KINGAI Runtime. It does not replace the bootable Server/Desktop/IoT operating-system editions.

## 3. D5 runtime architecture

```text
User / Organization / Device
            │
            ▼
┌──────────────────────────────────────────┐
│ KINGAI Intelligence                      │
│ Brain · Planner · Tasks · Knowledge      │
│ Memory · Models · Controlled Evolution   │
└───────────────────┬──────────────────────┘
                    │
                    ▼
┌──────────────────────────────────────────┐
│ KINGAI Governance                        │
│ Identity · Capability · Policy           │
│ Approval · Risk · Audit                  │
└───────────────────┬──────────────────────┘
                    │
                    ▼
┌──────────────────────────────────────────┐
│ Execution Fabric                         │
│ Native · OpenClaw · MCP · Codex          │
│ Browser · Rootless Container · VM        │
└───────────────────┬──────────────────────┘
                    │
                    ▼
┌──────────────────────────────────────────┐
│ KINGAI Secure Core                       │
│ systemd · cgroup v2 · AppArmor           │
│ seccomp · Landlock · Unix peer identity  │
└───────────────────┬──────────────────────┘
                    │
                    ▼
                Linux Kernel
```

Cloud services may accelerate the system, but they are not intended to become a survival dependency for local Policy, Memory, Task or offline-capable intelligence.

## 4. Current D5 runtime capabilities

### `kingaid`

The core daemon uses a local Unix socket instead of a TCP management listener. Peer identity is derived from Unix peer credentials.

Current services exposed through the daemon include:

- health;
- sanitized public status;
- Agent Registry;
- capability-policy evaluation;
- Approval Broker;
- local Memory service;
- Model Router;
- Task Graph;
- Audit.

### Agent identity

The runtime currently separates:

```text
main
system-ops
sec-ops
```

Privileged roles are bound to trusted local identities or root. A JSON request cannot simply claim a privileged role and receive authority.

### Capability Policy

Current capability classes include:

```text
filesystem.read
network.read
audit.read
filesystem.write
process.execute
service.restart
package.install
network.modify
security.modify
boot.modify
disk.raw
trust.modify
```

Unknown capabilities fail closed. Higher-risk capabilities require approval. Trust-root modification remains owner-only.

### Approval Broker

Approval records are bound to:

- Agent;
- Capability;
- Target Hash;
- Peer UID;
- expiration time.

Approvals can be pending, approved, denied, consumed or expired. Approved records are one-time tokens. Reuse and binding mismatch fail closed.

This closes a critical security gap: clients still cannot self-assert `Owner` or `Approved`, but there is now a controlled path for a real owner decision to become a single-use policy authorization.

### Task Graph

Persistent task lifecycle states:

```text
created
planning
waiting
waiting_approval
running
paused
blocked
failed
completed
cancelled
```

Tasks are bound to the local peer that created them. Steps may include dependencies, capabilities and approval references.

The Task Graph is the base contract for future planners and execution adapters.

### Memory

The current local-first Memory store supports:

- owner namespace;
- kind;
- sensitivity;
- creation timestamp;
- optional expiry;
- JSON payload;
- list and delete operations.

`kingaid` maps memory ownership to the Unix peer UID so ordinary local clients do not share one global memory namespace by default.

Long-term Memory Fabric remains designed around:

```text
M0 Context
M1 Working
M2 Task
M3 Episodic
M4 Semantic
M5 User / Organization
M6 Evolution
```

### Model Fabric

The current model router evaluates:

- requested capability;
- local vs remote;
- availability;
- priority;
- latency;
- cost class;
- private mode;
- offline mode.

Private and offline requests reject non-local candidates. If no candidate is valid, the service fails closed rather than silently choosing an incompatible model.

Provider adapters, provider health, region, license and richer hardware signals are later D5 work.

### Audit

Security-relevant runtime decisions are appended to the audit log. Policy audit records use target hashes rather than storing raw target values where the raw target is not necessary.

### CLI

```text
kingai version
kingai status [--json]
kingai doctor [--json] [--repair-safe]
kingai policy check <capability> [target]

kingai approval request <agent> <capability> [target]
kingai approval list
kingai approval approve <id>
kingai approval deny <id>

kingai memory put <kind> <json>
kingai memory list
kingai memory delete <id>

kingai model select <capability> [--private] [--offline]

kingai task create <agent> <goal...>
kingai task list
kingai task transition <id> <status>

kingai desktop list
kingai desktop show
kingai desktop set <kingai-intelligence|kingai-flow|kingai-classic>
kingai desktop apply
```

Approval decisions are intentionally privileged in the current developer implementation.

## 5. OS engineering foundation

The repository contains engineering tracks for:

- Ubuntu 26.04-based rootfs composition;
- Hybrid Live ISO;
- BIOS/UEFI boot validation;
- installer planning and execution components;
- Recovery environment;
- TUF client;
- pinned trust-root handling;
- A/B slot update state;
- health-gated update confirmation;
- rollback drills;
- Secure Boot VM verification;
- package inventory;
- SPDX 2.3 SBOM;
- release manifests and checksums.

Current machine-readable release gate truth lives in `release/gates.json`.

A VM-verified capability is not automatically a production release claim.

## 6. Security model

KINGAI OS is intentionally designed around the statement:

> powerful intelligence does not require unlimited authority.

The long-term execution path is:

```text
Agent Identity
  ↓
Declared Capability
  ↓
Central Policy
  ↓
Owner Approval when required
  ↓
Constrained Executor
  ↓
Sandbox / Resource Limits
  ↓
Audit / Health / Rollback
```

The future Execution Broker must not become a generic “AI root shell.” Privileged actions are expected to be capability-specific and constrained.

## 7. Build and test

Core:

```bash
make check
make build
```

Server:

```bash
sudo bash scripts/build-rootfs.sh server amd64 dist
sudo bash scripts/build-live-iso.sh server dist
```

Desktop / personal computer:

```bash
sudo bash scripts/build-rootfs.sh desktop amd64 dist
sudo bash scripts/build-live-iso.sh desktop dist
```

IoT / Edge:

```bash
sudo bash scripts/build-iot-image.sh arm64 dist
sudo bash scripts/build-iot-image.sh amd64 dist
```

Docker / OCI:

```bash
docker build -f container/Dockerfile -t kingai-os:dev .
```

or:

```bash
KINGAI_CONTAINER_PLATFORMS=linux/amd64,linux/arm64 \
  bash scripts/build-container.sh
```

Foundation CI currently validates Go build/test/vet, configuration, release controls, local IPC runtime behavior and the Container image startup path.

## 8. Release channels

```text
nightly -> dev -> beta -> rc -> stable
```

Stable remains blocked until the required security, installation, update, rollback, recovery, signing, supply-chain, hardware and governance gates have passed.

## 9. Artifact families

```text
KINGAI-OS-Server-<version>-amd64.iso
KINGAI-OS-Desktop-<version>-amd64.iso
KINGAI-OS-IoT-<version>-<arch>.img.xz
ghcr.io/kingaiwork/kingai-os:<version>
```

Additional architecture/artifact combinations are published only after dedicated validation.

## 10. What is not production-complete yet

The project deliberately does **not** claim the following as complete:

- final constrained privileged Execution Broker;
- production AppArmor/seccomp/Landlock agent profiles;
- automatic Planner/task-step scheduler;
- production OpenClaw/MCP/Codex/browser adapters;
- full M0-M6 memory retrieval/promotion/evolution;
- production model-provider adapters and model health service;
- production TUF repository and threshold-key operations;
- production Secure Boot signing and offline key custody;
- production R2 release delivery;
- branch/release governance;
- board-specific Edge Device Packs;
- final CVE/legal review;
- Stable lifecycle commitment.

See [docs/STATUS.md](docs/STATUS.md) for current verified truth.

---

# 中文

## KINGAI OS 的核心定位

KINGAI OS 的目标不是“做一个带 AI 的 Ubuntu”，而是形成一套真正的 **AI 原生操作系统治理与执行架构**。

传统系统主要管理：

```text
用户
文件
进程
网络
应用
硬件
权限
```

KINGAI OS 在这些能力之上继续增加：

```text
Agent Identity
Capability Policy
Approval
Task Graph
Memory Fabric
Model Fabric
Audit
Controlled Execution
Recovery / Rollback
Controlled Evolution
```

最终希望用户拥有的是一台：

> **会理解、会记忆、会规划、会执行，但不会越权的智能计算系统。**

## 四个正式版本

### KINGAI OS Server

面向服务器、VPS、企业自动化、AI 节点与分布式智能体。

### KINGAI OS Desktop

**Desktop 就是个人电脑 / PC 版本。**

不另外维护 `PC Profile`。

覆盖家庭电脑、办公电脑、开发者电脑、创作者工作站和本地 AI 工作站。

三套桌面体验：

- KINGAI Intelligence；
- KINGAI Flow；
- KINGAI Classic。

### KINGAI OS IoT / Edge

面向 ARM64/x86-64 边缘设备、网关、机器人和嵌入式智能。

通用架构镜像不代表自动支持所有硬件，具体 Raspberry Pi、Jetson、工业设备必须有真实 Device Pack 验证。

### KINGAI OS Container

面向 Docker / OCI / 云端容器 / CI / Homelab。

特点：

- amd64 / arm64 Buildx；
- 默认非 root 运行 `kingaid`；
- 不要求 systemd；
- 默认不开放管理 TCP；
- 保留 Policy / Approval / Task / Memory / Model / Audit；
- 数据通过 `/var/lib/kingai` 和 `/var/log/kingai` 持久化。

## 当前 D5 已经接通什么

```text
Agent
 ↓
Policy
 ↓
Approval
 ↓
Task Graph
 ↓
Memory / Model
 ↓
Audit
```

Approval 已经具备：

- 过期；
- 拒绝；
- 批准；
- 一次性消费；
- Agent + Capability + Target Hash + Peer UID 强绑定。

Task 已经具备：

- 持久化；
- UID 隔离；
- Step；
- Dependency；
- 状态机；
- Approval 关联基础。

Memory 已经通过 `kingaid` 成为本机服务，并按调用者 UID 隔离。

Model Router 已经成为 `kingaid` 的正式服务入口，Private/Offline 请求会排除非本地模型。

CLI 已经可以直接管理 Approval、Memory、Model、Task。

## 下一核心目标

下一步不是继续堆概念，而是完成真正的：

```text
Planner
 ↓
Task Graph
 ↓
Policy
 ↓
Approval
 ↓
Execution Broker
 ↓
Sandbox
 ↓
Result
 ↓
Audit
 ↓
Memory
```

其中 Execution Broker 必须是受限 Capability Broker，不能变成 AI 的无限制 root shell。

## 项目状态原则

KINGAI OS 坚持区分：

1. 代码存在；
2. CI 通过；
3. VM/硬件验证；
4. Artifact 已发布；
5. Production Ready。

这五个状态不是同一件事。

真实状态以 `docs/STATUS.md` 与 `release/gates.json` 为准。
