# KINGAI OS — Verified Project Status

**Status date:** 2026-08-14  
**Development line:** D5 Alpha Runtime Foundation / Pre-Alpha  
**Source version:** `0.1.0-dev`

This file is the human-readable status ledger. Machine-readable release truth is in [`release/gates.json`](../release/gates.json).

The project intentionally separates five states that must never be confused:

1. source code exists;
2. source code passes CI;
3. a VM or hardware path is verified;
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

The published image was rebuilt from repository source in GitHub Actions and passed its release workflow gates before publication.

## D5 runtime foundation

The D5 development line connects the previous D4 modules into one local runtime path.

### Core daemon

Verified by Foundation CI:

- `kingaid` local Unix-socket API;
- Unix peer credential identity;
- agent registry and role binding;
- capability policy evaluation;
- fail-closed behavior for unknown peer identity;
- sanitized public status;
- append-oriented audit logging.

### Approval Broker

Implemented and unit-tested:

- persistent approval requests;
- pending / approved / denied / consumed / expired states;
- approval expiry;
- one-time consumption;
- binding to Agent + Capability + Target Hash + Peer UID;
- binding-mismatch rejection;
- owner decision endpoint restricted to local UID 0;
- client requests still cannot self-assert `Owner` or `Approved`.

### Task Graph

Implemented and unit-tested:

- persistent tasks;
- peer UID ownership;
- task steps;
- dependency validation;
- optional capability and approval references;
- lifecycle states: `created`, `planning`, `waiting`, `waiting_approval`, `running`, `paused`, `blocked`, `failed`, `completed`, `cancelled`;
- invalid state-transition rejection;
- peer isolation for task listing and transitions.

### Memory service

Existing local-first FileStore is now connected to `kingaid`:

- put;
- list;
- delete;
- per-peer owner namespace;
- JSON data validation;
- restrictive filesystem permissions inherited from the store and systemd state directory.

The full M0-M6 semantic memory architecture is still a roadmap item; current storage is a safe persistent foundation rather than the final retrieval/evolution layer.

### Model service

Existing provider-neutral model selection is now connected to `kingaid`:

- capability filter;
- local/remote classification;
- availability;
- priority;
- latency;
- cost class;
- private-mode local enforcement;
- offline-mode local enforcement.

Concrete production provider adapters and model-health telemetry are not yet claimed complete.

### CLI

The operator CLI now covers the connected runtime foundations:

```text
kingai status
kingai doctor
kingai policy ...
kingai approval ...
kingai memory ...
kingai model ...
kingai task ...
kingai desktop ...
```

## OS engineering verification

According to the current release-gate ledger:

| Gate | Current state |
|---|---|
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

### Server

Verified engineering foundations include:

- Ubuntu 26.04-based Server rootfs;
- KINGAI `os-release`, MOTD and issue identity;
- kernel/initramfs composition;
- `kingaid` systemd enablement;
- Hybrid Live ISO construction;
- Casper filesystem/checksum metadata;
- BIOS/UEFI VM validation paths;
- ownership/UID leakage checks;
- installer VM validation path;
- update and rollback VM validation path.

The already-published `server-dev.2` image predates some later installer/update verification and therefore must not be described as containing every current main-branch capability.

### Desktop

Verified engineering foundations include:

- Ubuntu 26.04 Desktop rootfs;
- Plasma 6 / KWin Wayland / SDDM / Qt 6;
- KINGAI Welcome;
- KINGAI Intelligence / Flow / Classic manifests and layouts;
- KINGAI Agent Center;
- Desktop Live VM validation;
- automated desktop-frame capture tooling.

The visual experience remains under active Alpha refinement even though the current VM gate is marked verified.

### IoT / Edge

Verified engineering foundations include:

- generic amd64 Edge rootfs/image pipeline;
- generic arm64 Edge rootfs/image pipeline;
- compressed `.img.xz` checksum/manifest validation;
- compressed-size budget gate.

Generic Edge artifacts intentionally remain `bootable=false` / `device_pack_required=true` until a board-specific pack is validated. Raspberry Pi, Jetson and industrial hardware packs are not declared released.

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

These tests do **not** equal production signing readiness. The final production TUF repository, release-key ceremony/custody and production Secure Boot signing remain blocked.

## Supply chain and compliance

Implemented foundations include:

- Apache-2.0 project license for KINGAI-authored repository code;
- NOTICE and third-party policy;
- package inventory embedded into builds;
- deterministic SPDX 2.3 package SBOM generation;
- checksum and manifest generation;
- CodeQL workflow;
- staged release channels: `dev -> beta -> rc -> stable`;
- fail-closed release gates.

Unknown third-party license conclusions remain `NOASSERTION` rather than being guessed.

## Still protected / incomplete

The following are explicitly **not production-complete**:

- constrained privileged Execution Broker (`kingai-execd`) and its final sandbox profiles;
- production AppArmor/seccomp/Landlock agent policy set;
- final Planner / automatic task-step scheduler;
- production OpenClaw / MCP / Codex adapters;
- full M0-M6 memory retrieval, promotion and evolution pipeline;
- production model-provider adapters and model health service;
- production TUF repository and threshold-key operations;
- offline release-key custody and production Secure Boot signing;
- branch-protection/release-governance gate;
- R2 production release-delivery credentials and path;
- board-specific bootable Edge Device Packs;
- final CVE automation and final legal review;
- Stable support lifecycle commitment.

RC and Stable remain blocked until their required security, update, signing, recovery, supply-chain and governance gates are verified.

---

# 中文状态摘要

KINGAI OS 已从 **D4 Developer Foundation** 推进到 **D5 Alpha Runtime Foundation**。

当前不是简单增加功能，而是把原本分离的模块真正连接起来：

```text
Agent Identity
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

当前已经完成并进入 CI 验证的 D5 基础包括：

- Approval Broker 持久化；
- 审批过期、拒绝、批准、一次性消费；
- Agent + Capability + Target Hash + Peer UID 强绑定；
- Task Graph 持久化；
- 任务状态机和依赖校验；
- Task 按 Peer UID 隔离；
- Memory 接入 `kingaid` 并按 UID 隔离；
- Model Router 接入 `kingaid`；
- `kingai` CLI 增加 Approval / Memory / Model / Task 操作入口。

现有操作系统工程门禁已经比旧版 STATUS 更进一步：Installer VM、Desktop Live VM、Server BIOS/UEFI VM、A/B Update VM、Recovery VM、TUF Client、Secure Boot VM、Rollback Drill 和 Recovery Drill 当前都已在 `release/gates.json` 标记为已验证。

仍然不能宣称生产完成的部分包括：

- 真正的受限高权限 Execution Broker；
- 最终 Agent Sandbox；
- 自动 Planner 调度；
- OpenClaw/MCP/Codex 正式 Adapter；
- M0-M6 完整记忆检索与进化；
- 正式模型 Provider Adapter；
- Production TUF Repository；
- 生产签名密钥与 Secure Boot 签名；
- R2 正式发布链；
- 分支保护和正式 Release Governance；
- 硬件级 Edge Device Packs；
- Stable 生命周期承诺。

所以当前正确定位是：

> **一个已经具备真实 OS 构建、安装/恢复/更新验证基础，并开始形成 Agent → Policy → Approval → Task → Memory/Model → Audit 闭环的 AI 原生操作系统 Pre-Alpha。**
