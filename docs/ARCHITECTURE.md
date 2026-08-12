# KINGAI OS D4 Architecture

## English

KINGAI OS D4 is a sovereign distributed-intelligence operating-system architecture built around one shared core and three distribution profiles: Server, Desktop, and IoT/Edge.

### Layer model

```text
KINGAI Cloud / Optional Control Plane
            |
            v
KINGAI Intelligence
  Brain / Planner / Memory / Knowledge / Evolution
            |
            v
KINGAI Governance
  Policy / Capability / Risk / Identity / Audit
            |
            v
Execution Fabric
  Native / OpenClaw / MCP / Codex / Browser / Containers / VM
            |
            v
KINGAI Secure Core
  systemd / cgroup v2 / AppArmor / seccomp / Landlock
            |
            v
Linux Kernel
```

### Core services

Planned stable services:

- `kingaid` — orchestration and system intelligence daemon.
- `kingai-policyd` — capability and authorization policy service.
- `kingai-execd` — privileged execution broker.
- `kingai-modeld` — provider-neutral model routing and health.
- `kingai-memoryd` — local-first memory and retrieval service.
- `kingai-updated` — signed update, channel and rollback service.
- `kingai-auditd` — append-oriented security and agent audit service.
- `kingai-deviced` — hardware/device identity and edge integration service.

The initial implementation may combine multiple logical services in fewer binaries to reduce operational complexity. Service boundaries are architectural boundaries first, process boundaries second.

### Capability security

Agents never receive blanket root authority. Actions are described as capabilities, evaluated by policy, then executed by a constrained broker.

Examples:

```text
filesystem.read
filesystem.write:/workspace
process.execute
service.restart:nginx
package.install
network.modify
security.modify
boot.modify
disk.raw
```

Risk classes:

```text
L0 read-only
L1 user-safe
L2 application change
L3 system change
L4 security/network
L5 destructive/critical
L6 owner trust root
```

L6 is never self-grantable by an agent.

### Memory architecture

```text
M0 context
M1 working
M2 task
M3 episodic
M4 semantic
M5 user/organization
M6 evolution
```

Each memory record can carry owner, source, confidence, sensitivity, retention, jurisdiction, cloud policy and model policy metadata.

### Model fabric

Provider-neutral routing supports local and remote providers through adapters. The route decision may consider capability, privacy, cost, latency, context length, health, license, region and trust.

### Desktop architecture

Desktop ships one shared Desktop Core with three experience profiles:

- KINGAI Intelligence
- KINGAI Flow
- KINGAI Classic

The first-run experience shows visual previews and lets the user choose. The choice is per-user and changeable later.

### Distribution profiles

`profiles/server.yaml`, `profiles/desktop.yaml`, and `profiles/iot.yaml` define image composition. Shared packages remain in the base manifest. Profiles should contain deltas, not duplicated package lists.

### Update architecture

Production direction:

```text
signed metadata -> staged image -> inactive slot -> verify -> boot -> health gate
                                                    |            |
                                                    |            +-> success: commit
                                                    +--------------> failure: rollback
```

Stable releases are expected to support A/B or equivalent atomic rollback semantics.

### Cloud boundary

Cloud is an accelerator, not a survival dependency. Local agents, policy, memory and offline model support remain available without cloud connectivity.

Cloudflare is the initial control-plane/storage implementation, but interfaces must remain replaceable.

### Build principles

- pinned source manifests;
- reproducible-oriented builds;
- deterministic package composition where practical;
- SBOM and provenance on official artifacts;
- no unverified model redistribution;
- release signing outside ordinary web infrastructure;
- release channels: nightly -> dev -> beta -> rc -> stable.

---

## 中文

KINGAI OS D4 是一套“主权式分布智能操作系统”架构。Server、Desktop、IoT/Edge 三个版本共享同一套核心，只通过发行 Profile 控制软件包、驱动、界面和运行策略。

### 核心分层

```text
可选 KINGAI Cloud
      ↓
KINGAI Intelligence
      ↓
KINGAI Governance
      ↓
Execution Fabric
      ↓
KINGAI Secure Core
      ↓
Linux Kernel
```

### 核心服务

规划中的逻辑服务包括：`kingaid`、`kingai-policyd`、`kingai-execd`、`kingai-modeld`、`kingai-memoryd`、`kingai-updated`、`kingai-auditd`、`kingai-deviced`。

第一阶段可以为了简单可靠，把多个逻辑服务合并到较少进程中；先保持清晰的模块边界，再根据负载和安全要求拆分。

### 权限模型

智能体不能获得无限制 root。所有高权限动作必须通过 Capability + Policy + Executor 三层执行，并按照 L0-L6 风险等级管理。L6 信任根只能由所有者或受控硬件授权，智能体不得自行提升。

### 记忆模型

记忆分为 Context、Working、Task、Episodic、Semantic、User/Organization、Evolution 七层，并附带敏感等级、来源、置信度、保存期限、地区、云同步策略和模型访问策略。

### 桌面模型

Desktop 只有一个 Desktop Core，提供 KINGAI Intelligence、Flow、Classic 三种桌面体验。首次启动可视化展示后由用户选择，以后可随时切换。

### 更新模型

Stable 方向采用签名元数据、非活动槽写入、完整性验证、启动健康检查和失败自动回滚。云端不可成为系统基本生存依赖。

### 构建原则

固定源版本、减少不可追溯依赖、自动生成 SBOM/provenance、模型许可证门禁、密钥与普通网站隔离、Nightly 到 Stable 的分级发布门禁。
