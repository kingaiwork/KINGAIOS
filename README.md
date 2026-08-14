# KINGAI OS

> **Sovereign Distributed Intelligence Operating System**  
> **主权式分布智能操作系统**

AI-native · Agent-native · Local-first · Secure-by-default · Model-neutral · Cloud-neutral

**Development line:** D5 Alpha Runtime Foundation / Pre-Alpha  
**Current source version:** `0.1.0-dev`  
**Official OS site:** `https://os.kingai.work`  
**Repository:** `https://github.com/kingaiwork/KINGAIOS`

KINGAI OS is not a Linux desktop with a chatbot attached. It is a long-term operating-system project that moves **intelligence, agents, memory, task planning, model routing, policy, approvals, audit, recovery and controlled execution** into the operating-system architecture itself.

KINGAI OS 不是给 Linux 加一个聊天窗口，而是把 **智能、智能体、记忆、任务、模型路由、权限治理、审批、审计、恢复和受控执行** 变成操作系统原生能力。

> **Intelligence may be autonomous. Authority remains controlled, auditable and revocable.**  
> **智能可以自主，权限必须始终可控、可审计、可撤销。**

## Official sources / 官方入口

- KINGAI OS: `https://os.kingai.work`
- KINGAI: `https://www.kingai.work`
- Documentation: [docs/INDEX.md](docs/INDEX.md)
- Architecture: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Verified status: [docs/STATUS.md](docs/STATUS.md)
- Roadmap: [docs/ROADMAP.md](docs/ROADMAP.md)
- Desktop: [docs/DESKTOP.md](docs/DESKTOP.md)
- Device Packs: [docs/DEVICE-PACKS.md](docs/DEVICE-PACKS.md)
- Security: [SECURITY.md](SECURITY.md)
- Release policy: [docs/RELEASE-POLICY.md](docs/RELEASE-POLICY.md)
- Machine-readable facts: [llms.txt](llms.txt)
- GEO knowledge graph: [seo/GEO-KNOWLEDGE-GRAPH.md](seo/GEO-KNOWLEDGE-GRAPH.md)
- Business & partnership: `vip@kingai.work`

---

# English

## What KINGAI OS is

KINGAI OS is a Linux-based AI-native operating-system architecture developed by KINGAI. It combines a hardened Linux foundation with a governed intelligence layer so that AI agents can plan, remember, choose models, request permissions, create tasks and eventually perform constrained system actions without receiving unrestricted root authority.

The project deliberately remains independent from any single model provider, cloud vendor or agent framework. Local models, remote models, OpenClaw, MCP, Codex-compatible tools, browser runtimes and future execution engines are treated as replaceable adapters around a stable KINGAI governance and runtime core.

The operating principle is simple:

```text
understand -> plan -> evaluate policy -> request approval -> execute safely -> audit -> remember
```

## Architecture

```text
User / Organization / Device
            │
            ▼
┌──────────────────────────────────────┐
│ KINGAI Intelligence                  │
│ Brain · Planner · Tasks · Knowledge  │
│ Memory · Models · Evolution          │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ KINGAI Governance                    │
│ Identity · Capability · Policy       │
│ Approval · Risk · Audit              │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ Execution Fabric                     │
│ Native · OpenClaw · MCP · Codex      │
│ Browser · Containers · VM            │
└──────────────────┬───────────────────┘
                   │
                   ▼
┌──────────────────────────────────────┐
│ KINGAI Secure Core                   │
│ systemd · cgroup v2 · AppArmor       │
│ seccomp · Landlock · local IPC       │
└──────────────────┬───────────────────┘
                   │
                   ▼
               Linux Kernel
```

The current D5 development line is connecting previously separate foundations into one runtime path. The privileged Execution Broker remains a protected next-stage component and is not represented as production-complete.

## What is implemented now

### Core daemon and local trust boundary

`kingaid` is the local orchestration daemon. Management traffic stays on a Unix domain socket instead of exposing a TCP management API. The daemon derives peer identity from Unix peer credentials and uses that identity when evaluating privileged agent roles.

Current daemon services include:

- health and sanitized public status;
- agent registry and local identity binding;
- capability policy evaluation;
- persistent approval requests and decisions;
- one-time approval consumption;
- local-first memory persistence;
- provider-neutral model selection;
- persistent task graph creation and lifecycle transitions;
- append-oriented audit events.

### Approval Broker

Higher-risk capabilities do not become executable simply because an agent asks for them. Approval records are bound to:

- agent identity;
- capability;
- hashed target;
- local peer UID;
- expiration time.

Approved records are single-use. Reuse, expiry or binding mismatch fails closed. Owner-level approval decisions are restricted to local UID 0 in the current developer implementation.

### Task Graph

KINGAI OS now has a persistent task primitive with explicit lifecycle states:

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

Tasks are bound to the creating local peer and may include ordered steps, dependencies, capabilities and approval references. This is the foundation for future planners and execution runtimes.

### Memory

The local memory store currently provides persistent records with:

- owner namespace;
- kind;
- sensitivity;
- creation time;
- optional expiry;
- JSON payload;
- list and delete operations.

`kingaid` exposes memory through a peer-isolated local service. The long-term architecture expands this into M0-M6 memory layers covering context, working, task, episodic, semantic, user/organization and controlled evolution memory.

### Model Fabric

The model router is provider-neutral and already evaluates:

- capability support;
- local vs remote placement;
- availability;
- priority;
- latency;
- cost class;
- private mode;
- offline mode.

Private or offline requests automatically reject non-local candidates. Provider adapters and richer health/region/license signals are staged for later D5 work.

### Policy and capability security

Agents use declared capabilities rather than unrestricted shell authority.

Current policy classes include:

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

Unknown capabilities are denied by default. High-impact capabilities require explicit approval, and trust-root modification remains owner-only.

### Audit

Security-relevant policy and runtime decisions are appended to the KINGAI audit log. Raw target values are not placed into the policy audit record; target hashes are used where appropriate to reduce unnecessary sensitive-data exposure.

## Editions

### KINGAI OS Server

Headless profile for VPS, servers, AI nodes, enterprise automation and distributed agent infrastructure.

Verified foundations include Ubuntu 26.04-based rootfs generation, KINGAI system identity, kernel/initramfs composition, `kingaid` service enablement, Hybrid Live ISO generation and QEMU boot validation.

### KINGAI OS Desktop

One Desktop Core with three switchable experiences:

- **KINGAI Intelligence** — AI-first workspace centered on agents, tasks, memory, models, knowledge and automation.
- **KINGAI Flow** — modern spatial workflow with dock-oriented interaction.
- **KINGAI Classic** — familiar taskbar/application-menu experience.

The desktop foundation uses Plasma 6, KWin Wayland, SDDM and Qt 6. KINGAI Welcome and KINGAI Agent Center are part of the current desktop development line.

### KINGAI OS IoT / Edge

Minimal profile for ARM64/x86-64 edge systems, gateways, robotics and embedded intelligence. Generic amd64 and arm64 image pipelines exist; board-specific bootable Device Packs remain separate validation work.

## Installer, recovery and update foundation

The repository includes real engineering work for:

- installer planning and execution components;
- BIOS/UEFI installation validation;
- recovery environment and recovery drills;
- signed update metadata foundations;
- TUF client and pinned-root protection;
- A/B slot state and rollback tests;
- Secure Boot VM validation;
- health-gated update confirmation.

Production release signing, final TUF repository key operations and stable lifecycle commitments remain protected release gates.

## Supply-chain and release engineering

Official build design includes:

- Go unit tests and `go vet`;
- CodeQL;
- rootfs/ISO smoke tests;
- installer VM tests;
- recovery VM tests;
- A/B update VM tests;
- Secure Boot VM tests;
- deterministic package inventory;
- SPDX 2.3 SBOM generation;
- checksums and machine-readable manifests;
- dev -> beta -> rc -> stable release gates.

Release truth is tracked in [release/gates.json](release/gates.json). A capability is not called production-ready merely because source code exists.

## CLI

The `kingai` command is the local operator interface.

```text
kingai status
kingai doctor
kingai policy check filesystem.read

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
kingai desktop set kingai-intelligence
kingai desktop apply
```

Approval decisions are intentionally privileged in the current developer design.

## Distribution targets

```text
KINGAI-OS-Server-<version>-amd64.iso
KINGAI-OS-Desktop-<version>-amd64.iso
KINGAI-OS-IoT-<version>-arm64.img.xz
```

GitHub is the source-code and release-metadata home. Smaller release assets may use GitHub Releases; large artifacts are designed to route through KINGAI-controlled object storage such as Cloudflare R2 once release credentials and policy gates are in place.

## Release channels

```text
nightly -> dev -> beta -> rc -> stable
```

Stable is blocked until the security, install, update, rollback, recovery, signing, supply-chain and legal gates required by the release policy are satisfied.

---

# 中文

## KINGAI OS 是什么

KINGAI OS 是 KINGAI 长期建设的 AI 原生操作系统。它以 Linux 为可靠底座，但系统设计的重点不再只是“运行应用”，而是让操作系统能够在受控边界内理解目标、组织任务、调用智能体、选择模型、管理长期记忆、请求权限、执行操作并留下完整审计轨迹。

它不是把某个大模型绑定到系统里，也不是把某个 Agent 框架当成操作系统本身。KINGAI OS 将模型、云服务和执行引擎都设计成可替换组件，真正长期稳定的是 KINGAI 自己的 **Intelligence + Governance + Runtime + Secure Core**。

核心闭环是：

```text
理解目标
  ↓
规划任务
  ↓
权限策略判断
  ↓
需要时请求用户审批
  ↓
受限执行
  ↓
健康检查与审计
  ↓
沉淀记忆
  ↓
继续学习和改进
```

## 当前 D5 Alpha Runtime Foundation

当前开发已经从早期 D4“工程基础”进入 D5“运行时闭环”阶段。

现在已经真实接通：

- `kingaid` 本机 Unix Socket 核心服务；
- Agent Registry；
- 本机 UID 身份绑定；
- Capability Policy；
- Approval Broker；
- 一次性审批凭证；
- 本地 Memory Service；
- Model Router；
- Task Graph；
- Audit；
- Health / Public Status。

这意味着 KINGAI OS 正从“有各个 AI OS 模块”进入“这些模块真正开始互相协作”的阶段。

## 权限与审批

KINGAI OS 的设计原则不是给 AI 一个无限制 root shell。

智能体必须先声明能力：

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
trust.modify
```

未知能力默认拒绝。高风险能力需要显式审批；信任根修改属于最高级别权限，只允许 Owner 控制。

新的 Approval Broker 会把审批与以下信息绑定：

- Agent；
- Capability；
- Target Hash；
- 本机 Peer UID；
- 过期时间。

审批只能消费一次，不能让 Agent 自己伪造 `Approved=true`，也不能把一个审批拿去批准另一个目标。

## Task Graph

KINGAI OS 不希望未来的智能体“想到什么就执行什么”，而是把复杂目标拆成可追踪任务。

任务状态包括：

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

任务可以包含多个步骤、依赖关系、Capability 和 Approval ID，并按本机用户隔离。这为后续 Planner、OpenClaw、MCP、Codex 和浏览器执行器提供统一任务生命周期。

## Memory Fabric

当前已经有真实的本地持久化 Memory Store，并通过 `kingaid` 暴露成受 UID 隔离的本地服务。

长期设计继续扩展为：

```text
M0 Context       当前上下文
M1 Working       工作记忆
M2 Task          任务记忆
M3 Episodic      情景记忆
M4 Semantic      语义记忆
M5 User / Org    用户与组织记忆
M6 Evolution     受控进化记忆
```

敏感记忆优先保留在本机，云端只能成为增强能力，不是 KINGAI OS 生存所必需的依赖。

## Model Fabric

KINGAI OS 不绑定 OpenAI、DeepSeek、Cloudflare、OpenRouter、Ollama 或任何其他单一提供商。

模型路由层会逐步综合：

```text
能力
隐私
本地/云端
可用性
延迟
成本
上下文
区域
信任
许可证
GPU/NPU
```

当前实现已经支持 Capability、Local、Availability、Priority、Latency、Cost、Private 和 Offline 约束。Private / Offline 模式会自动排除不符合要求的远程模型。

## 三个发行版本

### KINGAI OS Server

面向 VPS、服务器、AI 节点、企业自动化和分布式智能体基础设施。

当前已经验证 Ubuntu 26.04 RootFS、KINGAI 系统身份、Kernel/initramfs、`kingaid` systemd 服务、Hybrid Live ISO 和 QEMU 启动链。

### KINGAI OS Desktop

只维护一个 Desktop Core，提供三个可切换体验：

- **KINGAI Intelligence** — 原生 AI 工作空间；
- **KINGAI Flow** — 现代化空间工作流；
- **KINGAI Classic** — 熟悉的传统桌面工作流。

当前桌面底座采用 Plasma 6 / KWin Wayland / SDDM / Qt 6，并已经包含 KINGAI Welcome 和 KINGAI Agent Center。

### KINGAI OS IoT / Edge

面向 ARM64/x86-64 边缘节点、机器人、网关和嵌入式智能设备。通用 amd64/arm64 镜像链已经存在，真正针对 Raspberry Pi、Jetson 和工业硬件的可启动 Device Pack 仍按硬件单独验证。

## 安装、恢复、更新与信任

仓库已经包含并验证了相当多的操作系统工程能力，包括：

- Installer 规划与执行组件；
- BIOS/UEFI 安装测试；
- Recovery Environment；
- Recovery Drill；
- TUF Client；
- Pinned Root 防护；
- A/B Slot；
- 自动回滚测试；
- Secure Boot VM 验证；
- Update Health Gate。

但 KINGAI OS 不会把“测试环境已验证”写成“生产发行已经完成”。生产签名密钥、最终 TUF 仓库、长期 Stable 生命周期和正式发布治理仍然必须通过独立门禁。

## 为什么 KINGAI OS 不只是另一个 Linux 发行版

普通 Linux 发行版重点解决应用、驱动、桌面、服务器和软件包管理。

KINGAI OS 在这些能力之上增加一套操作系统级智能治理结构：

```text
传统 OS
应用 + 文件 + 进程 + 网络 + 用户 + 权限

KINGAI OS
传统 OS
+ Agent Identity
+ Capability Policy
+ Approval
+ Task Graph
+ Memory
+ Model Fabric
+ Audit
+ Controlled Execution
+ Recovery / Rollback
```

目标不是让 AI 权限更大，而是让 AI **能力更强、边界更清楚、行为更可控**。

## 项目状态

KINGAI OS 当前仍属于 **Pre-Alpha**，不是 Stable 产品。代码仓库坚持区分：

- 已写代码；
- 已通过 CI；
- 已通过 VM 验证；
- 已发布 Developer Preview；
- 已达到生产标准。

这五件事不是同一件事。

真实状态请始终以 [docs/STATUS.md](docs/STATUS.md) 和 [release/gates.json](release/gates.json) 为准。

## 下一阶段

D5 后续重点是把当前 Runtime Foundation 继续推进到完整 Alpha：

1. constrained Execution Broker；
2. AppArmor / seccomp / Landlock Agent 沙箱；
3. Planner 与 Task Step 调度；
4. OpenClaw Adapter；
5. MCP Adapter；
6. 模型 Provider Adapter 与健康检查；
7. Memory M0-M6 元数据与检索；
8. Desktop Approval / Task / Memory / Model Center；
9. 完整端到端故障注入；
10. 再进入 Beta 的生产信任与发布门禁。

KINGAI OS 的长期目标是一台 **会理解、会记忆、会规划、会执行，但不会越权的智能计算系统**。
