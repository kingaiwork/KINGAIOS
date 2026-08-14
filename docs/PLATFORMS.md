# KINGAI OS Platform Editions

KINGAI OS uses one shared intelligence, governance and runtime core across four official distribution forms. Product naming describes the deployment form; it does not create four unrelated codebases.

## Shared KINGAI Core

Every edition is designed around the same logical core:

```text
KINGAI Intelligence
├── Agent Runtime
├── Task Graph
├── Memory Fabric
├── Model Fabric
├── Policy Engine
├── Approval Broker
├── Audit
├── Health Intelligence
├── Update / Recovery contracts
└── Security boundaries
```

The editions differ in boot model, device integration, user interface, package composition and update delivery.

## 1. KINGAI OS Server

**Purpose:** VPS, physical servers, AI nodes, enterprise automation and distributed agent infrastructure.

Current profile: `profiles/server.yaml`

Primary characteristics:

- headless by default;
- amd64 and arm64 rootfs targets;
- SSH and cloud-init friendly;
- rootless-container support direction;
- optional local AI runtime;
- KINGAI daemon/CLI/policy/approval/task/memory/model/audit core;
- installer, recovery and A/B update engineering tracks.

Current bootable Server Developer Preview is amd64. ARM64 publication is gated separately from generic rootfs support.

## 2. KINGAI OS Desktop

**Purpose:** personal computers, workstations, developer machines, creator workstations and local-AI PCs.

Current profile: `profiles/desktop.yaml`

**Desktop is the PC edition. There is no separate PC profile.**

Current architecture target:

- amd64.

Desktop Core:

- Plasma 6;
- KWin Wayland;
- SDDM;
- Qt 6;
- KINGAI Welcome;
- KINGAI Agent Center.

User-selectable experiences:

1. KINGAI Intelligence;
2. KINGAI Flow;
3. KINGAI Classic.

Long-term Desktop centers on tasks, approvals, agents, models, memory, knowledge, privacy, health and automation rather than presenting AI as a separate chatbot application.

ARM64 Desktop is a future hardware-validation track, not a current release claim.

## 3. KINGAI OS IoT / Edge

**Purpose:** gateways, embedded systems, edge AI, robotics and intelligent devices.

Current profile: `profiles/iot.yaml`

Generic architecture targets:

- arm64;
- amd64.

Primary characteristics:

- headless/minimal composition;
- local-first and offline-capable runtime direction;
- Device Pack abstraction;
- constrained device capabilities;
- OTA/update and recovery contracts;
- optional local AI runtime;
- shared KINGAI governance and audit core.

A generic Edge image is not the same as hardware support. Raspberry Pi, Jetson and industrial hardware families require separately validated Device Packs before they are listed as supported devices.

## 4. KINGAI OS Container

**Purpose:** Docker/OCI, cloud services, CI, homelab, development environments and service composition.

Current profile: `profiles/container.yaml`

Current architecture build targets:

- linux/amd64;
- linux/arm64.

Container characteristics:

- no systemd requirement inside the image;
- `kingaid` runs directly as the container entrypoint;
- non-root daemon by default;
- Unix-socket management API remains the default local control path;
- `/var/lib/kingai` and `/var/log/kingai` are persistent-volume targets;
- Policy, Approval, Task Graph, Memory, Model Router and Audit remain present;
- no management TCP port is exposed by default;
- Dockerfile: `container/Dockerfile`;
- Buildx helper: `scripts/build-container.sh`.

Container is a deployment form of the KINGAI runtime. It does not replace the bootable Server/Desktop/IoT operating-system editions.

## Support matrix

| Capability | Server | Desktop | IoT / Edge | Container |
|---|---:|---:|---:|---:|
| `kingai` CLI | yes | yes | yes | yes |
| `kingaid` | yes | yes | yes | yes |
| Agent Registry | yes | yes | yes | yes |
| Capability Policy | yes | yes | yes | yes |
| Approval Broker | D5 | D5 | D5 | D5 |
| Task Graph | D5 | D5 | D5 | D5 |
| Local Memory | yes | yes | yes | yes |
| Model Router | yes | yes | yes | yes |
| Audit | yes | yes | yes | yes |
| Plasma Desktop | no | yes | no | no |
| Installer | OS track | OS track | device-specific | no |
| Recovery environment | OS track | OS track | device-specific | container restart/volume recovery |
| A/B OS update | OS track | OS track | target design | image replacement model |
| Docker/OCI artifact | optional runtime | optional runtime | optional runtime | native |

`D5` means the capability is part of the current Alpha Runtime development line and remains pre-Stable.

## Artifact families

```text
Server:
  KINGAI-OS-Server-<version>-amd64.iso

Desktop / personal computer:
  KINGAI-OS-Desktop-<version>-amd64.iso

IoT / Edge:
  KINGAI-OS-IoT-<version>-<arch>.img.xz

Container:
  ghcr.io/kingaiwork/kingai-os:<version>
```

Additional architecture/artifact combinations are published only after dedicated verification.

---

# 中文

KINGAI OS 正式维护四种发行形态：

- **Server**：服务器 / VPS / AI 节点 / 企业自动化；
- **Desktop**：个人电脑 / 工作站，也就是 PC 版本；
- **IoT / Edge**：边缘设备 / 网关 / 机器人 / 嵌入式智能；
- **Container**：Docker / OCI / 云端容器 / CI / Homelab。

四个平台共享同一个 KINGAI Core，不会分别维护四套互不兼容的智能体、记忆、模型、权限和审计系统。

## Desktop 就是 PC 版

个人电脑正式使用名称：

**KINGAI OS Desktop**

不再额外创建 `PC Edition` Profile，以免出现两套桌面构建链。

## Container 的定位

Container 不是“把整个 Linux ISO 塞进 Docker”，而是把 KINGAI Runtime 以容器原生方式运行：

- 非 root daemon；
- 不要求 systemd；
- Unix Socket 管理；
- 持久化 Memory / Task / Approval / Audit；
- amd64 / arm64 Buildx；
- 默认不开放管理 TCP 端口。

## IoT 的硬件支持原则

通用 ARM64 或 amd64 镜像只能说明“架构构建链存在”，不能自动等于“支持所有硬件”。必须通过对应 Device Pack 和真实启动验证，才列为正式支持设备。

## 统一目标

无论运行在服务器、个人电脑、IoT 还是 Docker，用户看到的核心逻辑保持一致：

```text
目标
 ↓
任务
 ↓
Policy
 ↓
Approval
 ↓
受控执行
 ↓
Audit
 ↓
Memory
```

平台可以不同，但 KINGAI 的安全边界和智能治理模型必须一致。
