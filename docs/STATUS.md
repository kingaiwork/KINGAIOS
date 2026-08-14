# KINGAI OS — Verified Project Status

**Status date:** 2026-08-14  
**Development line:** D5 Alpha Runtime Foundation / Pre-Alpha  
**Source version:** `0.1.0-dev`

This file is the human-readable status ledger. Machine-readable release truth is in [`release/gates.json`](../release/gates.json).

KINGAI OS intentionally distinguishes five states:

1. source code exists;
2. source code passes CI;
3. a VM/hardware/runtime path is verified;
4. an artifact is published;
5. a capability is production-ready.

A later state is never inferred from an earlier one.

## Published developer artifact

### KINGAI OS Server Developer Preview

- Release: `v0.1.0-dev-server-dev.2`
- Artifact: `KINGAI-OS-Server-0.1.0-dev-amd64.iso`
- Size: `1,092,444,160 bytes` (about 1.02 GiB)
- Assets: ISO, SHA-256 checksum and machine-readable manifest
- Install-to-disk in this published preview: **disabled**
- Production Secure Boot signing: **not enabled**

The published image predates part of the newer D5 runtime work. A published older Developer Preview must not be described as containing every capability now present on the development branch.

## Verified D5 runtime foundation

Foundation CI has verified the connected D5 local runtime path.

### Core daemon / trust boundary

Verified:

- `kingaid` local Unix-socket API;
- Unix peer credential identity;
- Agent Registry and local role binding;
- capability policy evaluation;
- fail-closed unknown peer identity;
- sanitized public status;
- append-oriented audit logging;
- CLI integration.

### Approval Broker

Implemented and verified by unit/integration tests:

- persistent approval requests;
- pending / approved / denied / consumed / expired states;
- approval expiry;
- one-time consumption;
- binding to Agent + Capability + Target Hash + Peer UID;
- binding-mismatch rejection;
- owner decision endpoint restricted to local UID 0;
- client JSON still cannot self-assert `Owner` or `Approved`;
- consumed approval reuse is rejected.

### Task Graph

Implemented and verified:

- persistent tasks;
- peer UID ownership;
- task steps;
- dependency validation;
- capability/approval reference fields;
- lifecycle states: `created`, `planning`, `waiting`, `waiting_approval`, `running`, `paused`, `blocked`, `failed`, `completed`, `cancelled`;
- invalid state-transition rejection;
- peer-isolated list/transition behavior.

### Memory service

Existing local-first FileStore is connected to `kingaid` and verified for:

- put;
- list;
- delete implementation path;
- per-peer owner namespace;
- valid JSON payload requirement;
- restrictive state-directory permissions.

The complete M0-M6 retrieval/promotion/evolution design remains future work; current storage is a persistent local foundation.

### Model service

Existing provider-neutral selection is connected to `kingaid`.

Verified routing behavior includes:

- capability filter;
- local/remote classification;
- availability;
- priority;
- latency;
- cost class;
- private-mode local enforcement;
- offline-mode local enforcement;
- fail-closed response when no eligible model exists.

Production provider adapters and provider-health telemetry are not yet claimed complete.

## Four official platform forms

KINGAI OS now maintains one shared core with four official distribution forms:

```text
Server
Desktop      # personal computer / PC edition
IoT / Edge
Container    # Docker / OCI
```

Desktop is the PC edition; there is no separate `pc` profile.

### Server

Verified engineering foundations include:

- Ubuntu 26.04-based Server rootfs;
- KINGAI system identity;
- kernel/initramfs composition;
- `kingaid` systemd enablement;
- Hybrid Live ISO;
- BIOS/UEFI VM validation tracks;
- ownership/UID leakage checks;
- installer VM validation;
- update/rollback VM validation.

Generic amd64/arm64 rootfs paths exist, but published boot artifact support is architecture-specific and must be verified separately.

### Desktop / personal computer

Verified engineering foundations include:

- Ubuntu 26.04 Desktop rootfs;
- Plasma 6 / KWin Wayland / SDDM / Qt 6;
- KINGAI Welcome;
- KINGAI Intelligence / Flow / Classic layouts;
- KINGAI Agent Center;
- Desktop Live VM validation;
- automated desktop-frame capture tooling.

Desktop remains actively developed even though its current VM gate is verified.

### IoT / Edge

Verified engineering foundations include:

- generic amd64 Edge rootfs/image pipeline;
- generic arm64 Edge rootfs/image pipeline;
- compressed `.img.xz` checksum/manifest validation;
- compressed-size budget gate.

Generic Edge images remain separate from concrete hardware support. Raspberry Pi, Jetson and industrial hardware families require validated Device Packs before they are listed as supported.

### Container / Docker / OCI

Verified in Foundation CI:

- Docker image builds successfully for the CI amd64 path;
- container starts `kingaid` successfully;
- `kingai status --json` succeeds inside the running image;
- Memory operation succeeds inside the container;
- Task creation succeeds inside the container;
- daemon runs as non-root `_kingai` by default;
- no management TCP port is exposed by default.

Implemented but not yet marked verified:

- dedicated Buildx multi-architecture workflow for linux/amd64 + linux/arm64;
- developer GHCR publish option;
- OCI SBOM/provenance flags on developer publishing workflow.

Until the dedicated multi-architecture workflow is actually run successfully, `container_multiarch_ci` remains `false`.

## Current release-gate ledger

| Gate | Current state |
|---|---|
| D5 Runtime CI | verified |
| Container image CI (amd64 CI path) | verified |
| Container multiarch CI/publish | **not yet verified** |
| Installer VM | verified |
| Desktop Live VM | verified |
| Server BIOS/UEFI VM | verified |
| A/B Update VM | verified |
| Recovery VM | verified |
| TUF Client | verified |
| Secure Boot VM | verified |
| Rollback drill | verified |
| Recovery drill | verified |
| Production TUF repository | **not ready** |
| Production signing | **not ready** |
| Protected branch governance | **not enabled** |
| R2 release delivery | **not ready** |

## Update, recovery and trust

Current repository verification includes:

- TUF client behavior;
- pinned-root path protection;
- permission self-healing limited to allowlisted trust files;
- A/B update VM path;
- health confirmation;
- intentional boot-failure rollback drill;
- recovery VM and recovery drill;
- Microsoft-enrolled OVMF Secure Boot VM validation.

These tests do **not** equal production signing readiness. Final production TUF repository operations, release-key ceremony/custody and production Secure Boot signing remain blocked.

## Supply chain and compliance

Implemented foundations include:

- Apache-2.0 project license for KINGAI-authored repository code;
- NOTICE and third-party policy;
- package inventory embedded into OS builds;
- deterministic SPDX 2.3 package SBOM generation;
- checksum and manifest generation;
- CodeQL workflow;
- Docker/OCI developer build workflow with provenance/SBOM options;
- staged release channels: `dev -> beta -> rc -> stable`;
- fail-closed release gates.

Unknown third-party license conclusions remain `NOASSERTION` instead of being guessed.

## Still protected / incomplete

The following are explicitly **not production-complete**:

- constrained privileged Execution Broker (`kingai-execd`) and final sandbox profiles;
- production AppArmor/seccomp/Landlock agent policy set;
- final Planner / automatic task-step scheduler;
- production OpenClaw / MCP / Codex / browser adapters;
- full M0-M6 memory retrieval, promotion and evolution pipeline;
- production model-provider adapters and model health service;
- production TUF repository and threshold-key operations;
- offline release-key custody and production Secure Boot signing;
- branch-protection/release-governance gate;
- production R2 release-delivery credentials and path;
- verified multi-architecture container publication;
- board-specific bootable Edge Device Packs;
- final CVE automation and final legal review;
- Stable support lifecycle commitment.

RC and Stable remain blocked until their required security, update, signing, recovery, supply-chain, hardware and governance gates are verified.

---

# 中文状态摘要

KINGAI OS 已从 D4 Developer Foundation 推进到 **D5 Alpha Runtime Foundation**。

当前已真实通过 Foundation CI 的 D5 闭环基础：

```text
Unix Peer Identity
 ↓
Agent Registry
 ↓
Capability Policy
 ↓
Approval Broker
 ↓
Task Graph
 ↓
Memory / Model
 ↓
Audit
```

已经验证：

- Approval 持久化、过期、批准、拒绝、一次性消费；
- Agent + Capability + Target Hash + Peer UID 强绑定；
- Approval 重复使用拒绝；
- Task Graph 持久化、依赖校验、状态机、UID 隔离；
- Memory 接入 `kingaid` 并实际读写；
- Model Router 接入 `kingaid`，没有合适模型时安全失败；
- CLI 集成；
- Docker amd64 CI 镜像真实构建并启动；
- Docker 内部 Memory / Task / Status 操作通过。

四个平台正式统一为：

```text
KINGAI OS Server
KINGAI OS Desktop（个人电脑 / PC）
KINGAI OS IoT / Edge
KINGAI OS Container（Docker / OCI）
```

Desktop 就是 PC 版本，不再创建第二套 PC Profile。

Container 已有 amd64 CI 验证；linux/amd64 + linux/arm64 Buildx 工作流已经写入，但在专用多架构 workflow 真正成功运行前，不把它标成已验证。

当前仍不能宣称生产完成的核心部分包括：Execution Broker、最终 Agent Sandbox、Planner、正式 Adapter、生产 TUF/签名、R2 正式发布链、硬件 Device Pack、Stable 生命周期等。

因此当前正确定位是：

> **已经具备真实 OS 构建/安装/恢复/更新验证基础，并形成 Policy → Approval → Task → Memory/Model → Audit 运行闭环，同时覆盖 Server、Desktop、IoT/Edge、Docker/OCI 四种发行形态的 AI 原生操作系统 Pre-Alpha。**
