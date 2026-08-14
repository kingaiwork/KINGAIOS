# KINGAI OS D5 Architecture

KINGAI OS D5 is a sovereign distributed-intelligence operating-system architecture built around one shared intelligence/governance/runtime core and four official distribution forms: Server, Desktop, IoT/Edge and Container.

## 1. Architectural intent

The system is designed around a separation between intelligence and authority.

```text
Intelligence may propose and plan.
Policy decides what class of action is allowed.
Approval supplies explicit owner authority when needed.
Execution is constrained and audited.
Memory records outcomes without silently expanding privilege.
```

The target is not an unrestricted autonomous root agent. The target is a governed operating system in which intelligent components can become more capable without erasing user control.

## 2. Layer model

```text
Optional KINGAI Cloud / Control Plane
                │
                ▼
┌────────────────────────────────────────────┐
│ KINGAI Intelligence                        │
│ Brain · Planner · Task Graph · Knowledge   │
│ Memory · Model Fabric · Evolution          │
└────────────────────┬───────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────┐
│ KINGAI Governance                          │
│ Identity · Capability · Policy · Approval  │
│ Risk · Audit · Privacy                     │
└────────────────────┬───────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────┐
│ Execution Fabric                           │
│ Native · OpenClaw · MCP · Codex-compatible │
│ Browser · Rootless Container · VM/MicroVM  │
└────────────────────┬───────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────┐
│ KINGAI Secure Core                         │
│ systemd · cgroup v2 · AppArmor · seccomp   │
│ Landlock · Unix peer identity              │
└────────────────────┬───────────────────────┘
                     │
                     ▼
                 Linux Kernel
```

Container deployments reuse the KINGAI Intelligence/Governance/Runtime layers without requiring systemd inside the OCI image.

## 3. Current D5 runtime path

The current implemented path is:

```text
Local client
    │
    ▼
Unix socket `/run/kingai/kingaid.sock`
    │
    ▼
Peer credential identity
    │
    ├── Agent Registry
    ├── Capability Policy
    ├── Approval Broker
    ├── Task Graph
    ├── Memory Store
    ├── Model Router
    └── Audit
```

`kingaid` currently combines several logical services in one process. This is deliberate for the Alpha foundation: service boundaries are architectural boundaries first and process boundaries second.

## 4. Logical services

Long-term logical service model:

- `kingaid` — orchestration and local intelligence hub;
- `kingai-policyd` — capability/risk/authorization policy;
- `kingai-approvald` — approval lifecycle and owner decisions;
- `kingai-execd` — constrained privileged execution broker;
- `kingai-taskd` — task graph and scheduling;
- `kingai-modeld` — provider-neutral model routing and health;
- `kingai-memoryd` — local-first memory and retrieval;
- `kingai-updated` — signed update, channel and rollback;
- `kingai-auditd` — append-oriented runtime/security audit;
- `kingai-deviced` — hardware identity and edge integration.

The current source may keep several of these inside `kingaid` until a real security, lifecycle or scaling reason justifies process separation.

## 5. Identity model

Local management is intentionally not exposed as a generic TCP management service.

The default trust path is:

```text
Unix connection
  ↓
SO_PEERCRED
  ↓
Peer UID
  ↓
Local username where available
  ↓
Agent role binding
```

Current trusted agent identities include:

```text
main
system-ops
sec-ops
```

A client JSON payload cannot self-assert `Owner` or `Approved` authorization.

## 6. Capability and risk model

Agents receive capabilities, not blanket root authority.

Current capability examples:

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

Unknown capabilities fail closed. L6 authority is never self-grantable by an agent.

## 7. Approval Broker

D5 introduces a real approval lifecycle rather than allowing a client to set an approval flag.

An approval record is bound to:

```text
Approval ID
Agent
Capability
Target Hash
Peer UID
Created At
Expires At
Status
Decision identity
```

States:

```text
pending
approved
denied
consumed
expired
```

Security properties:

- expiration;
- single-use consumption;
- target binding;
- peer binding;
- agent/capability binding;
- owner decision requirement;
- reuse rejection;
- mismatch rejection.

The current developer implementation restricts approval decisions to local UID 0.

## 8. Task Graph

Task Graph is the persistent contract between future planners and execution adapters.

```text
Task
├── ID
├── Goal
├── Agent
├── Peer UID
├── Status
├── Steps
│   ├── Step ID
│   ├── Title
│   ├── Capability
│   ├── Dependencies
│   ├── Approval ID
│   └── Status
├── Result
├── Error
├── Created At
└── Updated At
```

Task states:

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

Invalid transitions fail closed. Non-root local peers cannot transition or list another peer's tasks.

## 9. Memory architecture

Current storage is a safe local persistence foundation. Long-term architecture remains:

```text
M0 Context
M1 Working
M2 Task
M3 Episodic
M4 Semantic
M5 User / Organization
M6 Evolution
```

Future record metadata may include:

```text
owner
agent
namespace
layer
kind
sensitivity
source
confidence
importance
retention
jurisdiction
cloud policy
model policy
embedding reference
created / accessed / expiry timestamps
```

Current `kingaid` service maps memory ownership to local peer UID and exposes put/list/delete operations.

## 10. Model Fabric

KINGAI OS is provider-neutral.

Current candidate signals:

```text
provider
local
available
capabilities
priority
latency
cost class
```

Current request constraints:

```text
capability
private
offline
```

Private/offline requests exclude non-local candidates. If no eligible model exists, routing fails rather than silently violating the request.

Long-term routing adds provider health, region, context length, trust, license, hardware acceleration and policy constraints.

## 11. Execution architecture

The protected next-stage execution path is:

```text
Task Step
   ↓
Declared Capability
   ↓
Policy Result
   ↓
Approval token if required
   ↓
Execution Broker
   ↓
Capability-specific handler
   ↓
Sandbox / timeout / resource limit
   ↓
Result + audit
```

The Execution Broker must not expose an unrestricted privileged shell API.

Examples of capability-specific handlers:

```text
service.restart -> validated system service operation
package.install -> controlled package operation
filesystem.write -> scoped path operation
network.modify -> validated network policy operation
```

Generic arbitrary command execution is treated as a higher-risk separate capability and must not become a shortcut around the policy model.

## 12. Adapter architecture

External runtimes remain adapters rather than defining KINGAI OS identity.

Planned/staged adapters include:

- OpenClaw;
- MCP;
- Codex-compatible tool runtimes;
- browser automation;
- rootless containers;
- VM/MicroVM isolation;
- future device runtimes.

Adapter contract direction:

```text
Start
Stop
Health
Capabilities
Execute
Cancel
Metrics
```

## 13. Four platform editions

### Server

Headless OS profile for servers/VPS/AI nodes. Uses bootable OS engineering, installer, recovery and A/B update tracks.

### Desktop

Personal-computer/PC edition. Uses one Plasma 6 Desktop Core with KINGAI Intelligence, Flow and Classic experiences.

### IoT / Edge

Minimal OS profile for edge devices. Generic architecture images require board-specific Device Packs for concrete hardware support.

### Container

Docker/OCI runtime form. It does not require systemd inside the image, runs `kingaid` directly, remains non-root by default and uses persistent state/log volume paths.

Profiles:

```text
profiles/server.yaml
profiles/desktop.yaml
profiles/iot.yaml
profiles/container.yaml
```

## 14. Update and recovery architecture

Bootable OS editions follow the production direction:

```text
signed metadata
    ↓
staged image
    ↓
inactive slot
    ↓
verify
    ↓
boot
    ↓
health gate
 ┌──┴─────────────┐
 │                │
success         failure
 │                │
commit          rollback
```

Container uses an OCI replacement model rather than pretending A/B disk slots apply inside a container image. Persistent KINGAI state must remain separate from immutable image replacement.

## 15. Cloud boundary

Cloud is an accelerator, not a survival dependency.

Local operation should retain:

- agent identity;
- policy;
- approval;
- task state;
- local memory;
- audit;
- offline-capable model routing.

Cloudflare is an initial control-plane/storage implementation candidate, but interfaces remain replaceable.

## 16. Build and release principles

- pinned/reviewable dependencies;
- reproducible-oriented builds;
- deterministic package composition where practical;
- SBOM and provenance on official artifacts;
- no unverified model redistribution;
- release signing separated from ordinary web infrastructure;
- fail-closed release gates;
- staged channels: `nightly -> dev -> beta -> rc -> stable`;
- hardware support only after hardware-specific validation;
- public claims must distinguish code, CI, VM/hardware verification, published artifact and production readiness.

---

# 中文架构摘要

KINGAI OS D5 采用 **一套核心、四种发行形态**：

```text
KINGAI Intelligence
        ↓
KINGAI Governance
        ↓
Execution Fabric
        ↓
KINGAI Secure Core
        ↓
Linux Kernel

发行形态：
Server / Desktop / IoT-Edge / Container
```

Desktop 就是个人电脑 / PC 版本，不另外建立第二套 PC Profile。

当前 D5 已经真实接通：

```text
Unix Peer Identity
 ↓
Agent Registry
 ↓
Policy
 ↓
Approval Broker
 ↓
Task Graph
 ↓
Memory / Model
 ↓
Audit
```

Approval 不再是客户端可以伪造的布尔值，而是具备过期、一次性消费、Peer UID、Agent、Capability 和 Target Hash 强绑定的授权记录。

下一阶段 Execution Broker 必须坚持 Capability 专用执行器，不允许演变成“AI 无限 root shell”。

四个平台共享同一治理模型：服务器、个人电脑、IoT 和 Docker 可以拥有不同的启动、硬件与发布方式，但不能拥有不同的权限安全标准。
