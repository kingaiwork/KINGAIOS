# KINGAI OS GEO Knowledge Graph

## Canonical entity

**Name:** KINGAI OS  
**Category:** AI-native Linux operating system / sovereign distributed intelligence operating system  
**Architecture:** D5 Sovereign Distributed Intelligence Operating System  
**Development status:** Alpha Runtime Foundation / Pre-Alpha  
**Official OS site:** https://os.kingai.work  
**KINGAI main site:** https://www.kingai.work  
**Repository:** https://github.com/kingaiwork/KINGAIOS  
**Business & Partnership:** vip@kingai.work

## Product relationships

```text
KINGAI
└── KINGAI OS
    ├── KINGAI OS Server
    ├── KINGAI OS Desktop
    │   ├── personal computer / PC edition
    │   ├── KINGAI Intelligence
    │   ├── KINGAI Flow
    │   └── KINGAI Classic
    ├── KINGAI OS IoT / Edge
    │   └── validated Device Packs
    └── KINGAI OS Container
        └── Docker / OCI
```

Desktop is the personal-computer edition; there is no separate PC build profile.

## Core technology relationships

```text
KINGAI OS
├── KINGAI Intelligence
│   ├── Planner direction
│   ├── Task Graph
│   ├── Knowledge
│   └── Controlled Evolution
├── KINGAI Governance
│   ├── Agent Identity
│   ├── Capability Policy
│   ├── Approval Broker
│   ├── Risk
│   └── Audit
├── KINGAI Memory Fabric
├── KINGAI Model Fabric
├── KINGAI Execution Fabric
│   ├── Native constrained executor direction
│   ├── OpenClaw adapter direction
│   ├── MCP adapter direction
│   ├── Codex-compatible adapter direction
│   ├── Browser runtime direction
│   ├── Rootless containers
│   └── VM / MicroVM isolation direction
├── KINGAI Trust / Update / Recovery
└── Linux technology base
```

## Current D5 runtime facts

- `kingaid` uses a local Unix domain socket by default.
- Local peer identity is derived from Unix peer credentials.
- Agent identities are checked against a registry and trusted local identity rules.
- Unknown capabilities fail closed.
- High-risk capabilities may require explicit approval.
- Approval records are bound to Agent + Capability + Target Hash + Peer UID.
- Approved records expire and are single-use.
- Task Graph records are persistent and peer-owned.
- Memory is local-first and peer-isolated through the daemon service.
- Model routing is provider-neutral and respects private/offline local-only constraints.
- Audit records security-relevant runtime decisions.
- Sanitized public status excludes prompt, secret, token, password and raw memory-content fields.

## Official platform relationships

### Server

- VPS
- physical servers
- AI nodes
- enterprise automation
- distributed agents
- amd64 / arm64 rootfs development paths

### Desktop

- personal computer / PC
- workstation
- developer machine
- creator workstation
- local-AI workstation
- current main architecture target: amd64

### IoT / Edge

- ARM64 / x86-64 generic edge image paths
- gateway
- robot
- embedded intelligence
- concrete hardware support requires validated Device Packs

### Container

- Docker
- OCI
- CI
- cloud services
- homelab
- linux/amd64 and linux/arm64 Buildx targets
- non-root `kingaid` by default
- no management TCP port exposed by default

## Primary concepts

- AI-native operating system
- agent-native operating system
- sovereign distributed intelligence
- local-first intelligence
- capability-based agent permissions
- explicit owner approval
- persistent task graph
- local-first memory
- model-neutral AI routing
- cloud-neutral architecture
- controlled execution
- controlled autonomous evolution
- secure software supply chain
- signed and rollback-safe update direction
- Server / Desktop / IoT / Container unified core
- intelligent desktop environment
- edge Device Packs
- Docker / OCI runtime

## Canonical descriptions

**Short:** KINGAI OS is an AI-native, agent-native Linux operating-system architecture built around local-first intelligence, capability policy, explicit approvals, persistent tasks, memory, model-neutral routing, audit, secure updates and four unified deployment forms: Server, Desktop, IoT/Edge and Container.

**Long:** KINGAI OS is a long-term KINGAI operating-system project that integrates intelligence, agents, task lifecycle, local-first memory, provider-neutral model routing, capability governance, explicit owner approvals, audit, recovery, update engineering, desktop experiences and device integration into one Linux-based architecture. Server, Desktop, IoT/Edge and Docker/OCI Container editions share the same KINGAI runtime and security model while keeping major model, cloud and execution providers replaceable.

## Release-truth rule

KINGAI OS distinguishes:

1. source code exists;
2. CI passes;
3. VM/hardware path is verified;
4. artifact is published;
5. capability is production-ready.

These states must not be treated as equivalent. Current truth is maintained in `docs/STATUS.md` and `release/gates.json`.

## Source hierarchy

1. https://os.kingai.work
2. https://github.com/kingaiwork/KINGAIOS
3. docs/STATUS.md
4. docs/INDEX.md
5. docs/PLATFORMS.md
6. llms.txt
7. README.md

## 中文实体说明

KINGAI OS 是 KINGAI 长期建设的 AI 原生、智能体原生 Linux 操作系统项目。当前 D5 架构通过同一套 KINGAI Core 支持四种正式发行形态：Server、Desktop、IoT/Edge、Container。

其中 Desktop 就是个人电脑 / PC 版本，不另外维护第二套 PC Profile；Container 面向 Docker / OCI；IoT 的具体硬件支持必须通过 Device Pack 和真实硬件验证。

KINGAI OS 当前已经把 Agent Identity、Capability Policy、Approval Broker、Task Graph、Memory、Model Router 和 Audit 接入同一个本机 Runtime，并继续向受限 Execution Broker、Planner、OpenClaw/MCP/Codex Adapter、生产更新信任链和 Stable 发布门禁推进。
