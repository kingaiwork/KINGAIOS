# KINGAI OS — Verified Project Status

**Status date:** 2026-08-14  
**Development line:** D5 Alpha Runtime Foundation / Pre-Alpha  
**Source version:** `0.1.0-dev`

This is the human-readable engineering ledger. Machine-readable release truth lives in [`release/gates.json`](../release/gates.json).

KINGAI OS distinguishes these states:

1. code exists;
2. CI passes;
3. VM/runtime path passes;
4. real hardware passes;
5. an artifact is published;
6. production signing/release governance is ready;
7. Stable is ready.

A later state is never inferred from an earlier one.

## Current software baseline

The D5 source line now contains a connected governed runtime:

```text
Goal / validated plan
        ↓
Task Graph + Scheduler
        ↓
Unix Peer Identity + Agent Registry
        ↓
Capability Policy
        ↓
Approval Broker when required
        ↓
constrained Execution Broker / kingai-execd
        ↓
Result + Audit + Memory
```

### Identity and governance — verified

- local Unix-socket management;
- kernel peer credentials;
- Agent Registry and local role binding;
- fail-closed unknown peers and capabilities;
- risk-ranked Capability Policy;
- Approval lifecycle: pending / approved / denied / consumed / expired;
- approval expiry and one-time consumption;
- binding to Agent + Capability + Target Hash + Peer UID;
- local UID 0 approval decision boundary in the developer implementation;
- clients cannot self-assert Owner/Approved authority.

### Planner / Task Graph / Scheduler — verified foundation

- Planner validation contract;
- persistent tasks and steps;
- dependency validation and cycle rejection;
- peer UID ownership and isolation;
- task/step lifecycle states;
- ready-step selection;
- policy rejection → blocked behavior;
- approval-required → waiting_approval behavior;
- execution failure → failed behavior;
- all-required-steps-success → completed behavior;
- runtime API/CLI task operations.

This is the execution-planning foundation, not a claim that arbitrary model-generated plans are safe or production-ready.

### Constrained Execution Broker — verified foundation

`kingai-execd` exists as a separate privileged broker for Server/Desktop system operations.

Verified properties include:

- only the configured local KINGAI runtime identity can access its Unix socket;
- socket permission checks;
- capability allowlisting;
- unknown capability rejection;
- no generic privileged shell endpoint;
- native privileged surface intentionally limited to `service.restart`;
- strict service-unit validation before `/usr/bin/systemctl` is invoked;
- command-injection regression test;
- default execution deadline;
- bounded Agent, Target and Arguments input sizes;
- unique `execution_id` receipt for attempted registered executions;
- execution duration attached to result/audit evidence;
- systemd hardening and explicit memory/task/file-descriptor bounds.

Additional privileged capabilities remain blocked until each has a dedicated validator, handler, security model and failure test.

### Memory Fabric — verified foundation

The local FileStore and runtime API support layered memory metadata:

```text
M0 Context
M1 Working
M2 Task
M3 Episodic
M4 Semantic
M5 User / Organization
M6 Evolution
```

Implemented foundations include:

- per-peer owner isolation;
- Agent and Namespace metadata;
- Layer / Kind / Sensitivity;
- Source / Confidence / Importance;
- Retention / Jurisdiction;
- Cloud Policy / Model Policy;
- Created / Accessed / Expiry metadata;
- put / get / list / search / delete;
- expiry purge;
- controlled promotion primitive;
- restrictive local file permissions.

M6 does not mean autonomous self-modification authority. Evolution memory remains governed data.

### Model Fabric — verified foundation

Implemented:

- provider-neutral Candidate model;
- capability filtering;
- availability and local/remote classification;
- priority / latency / cost signals;
- Private and Offline local-only enforcement;
- fail-closed behavior when no eligible model exists;
- Provider Registry;
- provider Health aggregation foundation.

Production provider adapters, credentials, quota policy and model-specific operational tuning remain future integration work.

### Runtime Adapter contract — implemented foundation

External systems are adapters rather than the KINGAI trust root.

Current contract direction is implemented around:

```text
Start
Stop
Health
Capabilities
Execute
Cancel
```

OpenClaw, MCP, Codex-compatible runtimes, browsers, containers and future device runtimes remain replaceable integrations. Production adapters are not yet claimed complete.

## Four official platform forms

```text
KINGAI OS Server
KINGAI OS Desktop      # personal computer / PC edition
KINGAI OS IoT / Edge
KINGAI OS Container    # Docker / OCI
```

All four share one governance/runtime core.

### Server

Verified engineering foundations:

- Ubuntu 26.04-based rootfs composition;
- amd64/arm64 rootfs build paths;
- headless KINGAI runtime;
- Hybrid Live ISO engineering;
- BIOS/UEFI VM tracks;
- installer VM track;
- `kingaid` and constrained ExecD packaging;
- A/B update and rollback VM paths;
- Recovery path.

### Desktop

Desktop is the personal-computer / PC edition.

Verified/implemented foundations:

- Plasma 6 / KWin Wayland / SDDM / Qt 6;
- one Desktop Core;
- KINGAI Intelligence / Flow / Classic experiences;
- Welcome application;
- Agent Center foundation;
- Desktop Live VM path;
- installer path;
- desktop-frame capture tooling;
- shared D5 Policy/Approval/Task/Memory/Model runtime.

Task Center, Approval Center, Memory Center and Model Center remain active desktop product work rather than completed production UI.

### IoT / Edge

Verified engineering foundations:

- generic amd64 image path;
- generic arm64 image path;
- compressed image checksum/manifest checks;
- size budget;
- Device Pack contract/schema.

Raspberry Pi, Jetson and industrial devices are not listed as supported until their board-specific Device Pack boots on real hardware.

### Container

Verified engineering foundations:

- Docker/OCI profile;
- non-root `_kingai` runtime;
- Unix-socket management model;
- persistent state pattern;
- amd64 image build/runtime smoke;
- Linux amd64 + arm64 Buildx verification;
- no host Docker socket/root filesystem exposure by default;
- developer SBOM/provenance-capable build path.

Official production registry publication/signing remains gated.

## System lifecycle and trust

Verified engineering paths include:

- installer VM;
- Desktop Live VM;
- Server BIOS/UEFI VM;
- A/B Update VM;
- rollback drill;
- Recovery VM;
- recovery drill;
- TUF client behavior;
- pinned-root tamper/path protections;
- Secure Boot VM validation with enrolled OVMF test environment.

These do **not** imply production key custody or production signing readiness.

## Release-gate summary

| Gate | State |
|---|---|
| D5 Runtime CI | verified |
| Constrained ExecD CI | verified |
| Container image CI | verified |
| Container amd64 + arm64 Buildx CI | verified |
| Installer VM | verified |
| Desktop Live VM | verified |
| Server BIOS/UEFI VM | verified |
| A/B Update VM | verified |
| Recovery VM | verified |
| Rollback drill | verified |
| Recovery drill | verified |
| TUF client | verified |
| Secure Boot VM | verified |
| Production TUF repository | **not ready** |
| Production signing | **not ready** |
| Protected branch governance | **not enabled** |
| Production R2/artifact delivery | **not ready** |
| Real board Device Packs | **not ready** |
| Stable lifecycle | **not ready** |

## Engineering policy

See [`ENGINEERING-PRINCIPLES.md`](ENGINEERING-PRINCIPLES.md).

The short version:

> KINGAI OS learns from public research, open standards and mature operating-system practice, but KINGAI-specific code, runtime governance and product UI are independently engineered. New technology must make the system simpler, safer, more reliable, more stable or measurably more efficient.

The trusted core favors:

- Go standard library where practical;
- mature Linux primitives;
- minimal daemon count;
- minimal privileged surface;
- bounded requests and timeouts;
- explicit failure modes;
- local-first operation;
- replaceable providers/adapters;
- evidence before claims.

## Published developer artifact

The currently public artifact remains the earlier Server Developer Preview:

- Release: `v0.1.0-dev-server-dev.2`
- Artifact: `KINGAI-OS-Server-0.1.0-dev-amd64.iso`
- Size: `1,092,444,160 bytes`
- Developer Preview, not Stable
- Published image predates part of the current D5 source line

Do not describe that older ISO as containing every newer source capability until a new image is built, tested and published.

## Still protected / incomplete

The following remain intentionally incomplete or gated:

- additional privileged capability handlers beyond the narrow current ExecD surface;
- production AppArmor/seccomp/Landlock profiles for broader agent execution;
- production OpenClaw/MCP/Codex/browser adapters;
- production model-provider adapters and credential/quota operations;
- semantic/vector retrieval backend for large-scale memory;
- Desktop Task/Approval/Memory/Model Center production UI;
- production TUF repository and threshold-key ceremony;
- offline release-key custody;
- production Secure Boot signing;
- protected-branch/release governance;
- production artifact delivery and signing;
- real-hardware board Device Packs;
- final CVE automation/legal release review;
- Stable support lifecycle commitments.

---

# 中文摘要

当前 KINGAI OS 已形成一套可验证的 D5 Alpha Runtime 基础：

```text
计划
 ↓
Task Graph / Scheduler
 ↓
身份
 ↓
Policy
 ↓
Approval
 ↓
受限 ExecD
 ↓
Result / Audit / M0-M6 Memory
```

四个平台统一为：Server、Desktop（个人电脑）、IoT/Edge、Container。

当前代码已经具备受限高权限 Execution Broker、步骤级 Scheduler、M0–M6 Memory 元数据/检索基础、Provider Registry/Health 基础、多架构 Container CI，以及完整的安装/更新/恢复/VM 信任工程路径。

但生产 TUF 密钥、正式 Secure Boot 签名、真实硬件 Device Pack、正式外部 Adapter、生产模型 Provider、最终 Desktop 系统中心与 Stable 生命周期仍然受到发布门禁保护。

因此目前正确定位仍然是：

> **KINGAI OS D5 Alpha Runtime Foundation / Pre-Alpha — 软件核心已经进入真实可运行闭环，但不提前冒充生产 Stable。**
