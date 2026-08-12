# KINGAI OS Roadmap

## English

This roadmap prioritizes a reliable release system before feature volume.

### Phase 0 — D4 Developer Foundation

- repository architecture and policy baseline;
- Server/Desktop/IoT profiles;
- first image-build pipeline;
- model-neutral interfaces;
- agent capability model;
- legal/license inventory;
- SBOM/provenance skeleton;
- R2 large-artifact publishing path;
- security disclosure process;
- desktop experience specification.

### Phase 1 — 0.1 Developer Image

- bootable Server developer image;
- `kingai` CLI and `kingaid` skeleton;
- system health reporting;
- local policy file and capability evaluator;
- provider adapter interface;
- local memory store;
- CI checks for package licenses and image size;
- signed checksum generation.

### Phase 2 — 0.3 Agent Runtime

- isolated execution broker;
- AppArmor/seccomp/Landlock profiles;
- rootless container backend;
- OpenClaw/MCP/Codex adapter contracts;
- audit log and task lifecycle;
- first local/cloud model routing policy.

### Phase 3 — 0.5 Desktop Alpha

- KINGAI Welcome first-run setup;
- live preview desktop selector;
- KINGAI Intelligence / Flow / Classic profiles;
- KINGAI Agent panel;
- model manager;
- memory/privacy controls;
- settings integration.

### Phase 4 — 0.7 Update & Trust

- signed update metadata;
- TUF repository layout;
- A/B or equivalent atomic rollback prototype;
- Secure Boot/TPM integration track;
- recovery environment;
- update failure injection tests.

### Phase 5 — 0.9 Release Candidate

- clean-install validation;
- upgrade/rollback validation;
- SBOM and provenance on every image;
- corresponding-source publication workflow;
- CVE/security advisory process;
- release signing ceremony/documentation;
- privacy/export/delete controls;
- public documentation and hardware matrix.

### Phase 6 — 1.0 Stable

Stable is released only after security, legal, licensing, upgrade, rollback, recovery and reproducibility gates are satisfied.

### After 1.0

- ARM64 Desktop when hardware validation is sufficient;
- IoT hardware packs;
- distributed fleet management;
- organization memory and policy federation;
- optional remote attestation;
- agent marketplace with signed manifests;
- advanced scheduling and hardware acceleration;
- enterprise support lifecycle.

---

## 中文

本路线图首先保证“能长期安全发行”，再追求功能数量。

### 阶段 0 — D4 Developer Foundation

完成仓库架构、三个发行 Profile、镜像构建骨架、模型抽象、Agent 权限、许可证/SBOM、R2 大文件发布、安全流程与桌面规范。

### 阶段 1 — 0.1 Developer Image

完成可启动 Server 开发镜像、`kingai` CLI、`kingaid`、系统健康、基础 Policy、模型 Provider 接口、本地 Memory、镜像体积与许可证 CI、签名校验文件。

### 阶段 2 — 0.3 Agent Runtime

完成执行 Broker、AppArmor/seccomp/Landlock、Rootless Container、OpenClaw/MCP/Codex 适配接口、审计和基础模型路由。

### 阶段 3 — 0.5 Desktop Alpha

完成 KINGAI Welcome、三个桌面实时预览与切换、Agent Panel、模型管理、Memory/Privacy 设置。

### 阶段 4 — 0.7 Update & Trust

完成签名更新元数据、TUF布局、A/B回滚原型、Secure Boot/TPM、恢复环境和故障注入测试。

### 阶段 5 — 0.9 RC

完成安装/升级/回滚验证、每个镜像的 SBOM/provenance、对应源码流程、安全公告、发布签名流程、隐私控制和公开硬件兼容文档。

### 阶段 6 — 1.0 Stable

只有通过安全、法律、许可证、升级、回滚、恢复和构建验证后才发布 Stable。

1.0 之后再扩展 ARM64 Desktop、IoT 硬件包、全球 Fleet、组织级记忆/策略、远程证明、签名 Agent Marketplace 和企业长期支持。
