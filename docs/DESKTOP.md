# KINGAI OS Desktop Architecture

**Role:** Personal computer / PC edition  
**Development line:** D5 Alpha Runtime Foundation

KINGAI OS Desktop is the official personal-computer edition of KINGAI OS. There is no separate PC build profile.

It uses one shared desktop foundation with three KINGAI-owned experience profiles and the same Policy, Approval, Task, Memory, Model and Audit core used by Server, IoT and Container.

## Foundation

```text
Wayland
  + KWin
  + Plasma 6
  + Qt 6 / QML
  + SDDM
  + KINGAI Desktop Core
  + KINGAI Runtime
```

Linux graphics/window-management infrastructure is reused where mature, while KINGAI owns the product experience, intelligent surfaces and governance model.

## One Desktop Core, three experiences

```text
KINGAI Desktop Core
├── KINGAI Intelligence
├── KINGAI Flow
└── KINGAI Classic
```

All three share:

- compositor/window manager;
- notification framework;
- file integration;
- global search;
- settings;
- accessibility;
- application launcher backend;
- KINGAI Agent Center;
- Task Graph API;
- Approval API;
- Memory API;
- Model API;
- system health/status.

They change layout and interaction, not the underlying operating-system runtime.

## KINGAI Intelligence

The flagship personal-computer experience should expose intelligent objects as first-class desktop concepts:

```text
Home
Agents
Tasks
Approvals
Memory
Knowledge
Models
Automations
Devices
Files
Apps
System
```

The goal is not to require the user to open a separate chatbot before every action. Files, projects, active tasks, agent state and memory context should be able to participate in one governed workflow.

Suggested primary layout:

```text
┌──────────────────────────────────────────────────────────┐
│ KINGAI OS                                                │
├──────────────┬──────────────────────────────┬─────────────┤
│ Navigation   │ Workspace / Task / Context   │ Live Status │
│              │                              │              │
│ Home         │ Goal                         │ System       │
│ Agents       │ Plan                         │ Model        │
│ Tasks        │ Steps                        │ Agents       │
│ Approvals    │ Results                      │ Security     │
│ Memory       │ Files / Knowledge            │ Update       │
│ Models       │                              │ Health       │
└──────────────┴──────────────────────────────┴─────────────┘
```

## Approval Center

D5 introduces a real Approval Broker, so the desktop should surface approval as a native OS concept.

Example interaction:

```text
KINGAI requests permission

Agent: system-ops
Action: install package
Target: nginx
Risk: system change
Expires: 5 minutes

[Deny]  [Allow once]
```

The UI must never create a reusable global “always allow everything” shortcut that bypasses policy.

Future UI may support narrowly scoped remembered policy decisions, but those remain separate from one-time approval tokens.

## Task Center

Task Center should reflect the runtime state machine:

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

A task view should show:

- goal;
- owning Agent;
- current state;
- steps;
- dependencies;
- required capabilities;
- approval state;
- execution result;
- audit timeline;
- related memory/context.

## Memory Center

Desktop Memory controls should give the user visibility into local intelligent state instead of hiding it behind opaque model behavior.

Long-term views:

```text
M0 Context
M1 Working
M2 Task
M3 Episodic
M4 Semantic
M5 User / Organization
M6 Evolution
```

User controls should include:

- search;
- sensitivity;
- retention/expiry;
- export;
- delete;
- cloud policy;
- model-access policy;
- source/provenance.

The current backend is simpler and local-first; UI claims must match actual backend capability during Alpha.

## Model Center

Model Center should present local and remote providers through one KINGAI Model Fabric rather than vendor-specific primary navigation.

User-visible controls may include:

- local/cloud/hybrid mode;
- private mode;
- offline mode;
- active providers;
- availability/health;
- latency/cost class;
- capability support;
- local GPU/NPU status.

Private/offline constraints are policy inputs, not decorative toggles.

## Agent Center

Agent Center should make Agent identity and permissions understandable:

```text
Agent
Role
Capabilities
Current tasks
Recent decisions
Runtime adapter
Health
```

Privileged roles such as `system-ops` and `sec-ops` must remain visually distinguishable from ordinary Agent identities.

## KINGAI Flow

A spatial, dock-oriented layout with a clean top-level interface and workspace-focused navigation. It shares all governance and runtime services with Intelligence and Classic.

## KINGAI Classic

A traditional PC workflow using taskbar/application-menu concepts for easier migration from conventional operating systems. Classic changes interaction style, not the security model.

## First-run experience

```text
KINGAI Welcome
  ↓
Preview Intelligence / Flow / Classic
  ↓
Select experience
  ↓
Privacy / local-vs-cloud defaults
  ↓
Optional model setup
  ↓
Enter Desktop
```

The selected experience remains changeable later:

```text
Settings -> Personalization -> Desktop Experience
```

## Personal-computer target users

Desktop is intended for:

- everyday home PCs;
- business/office PCs;
- software-development machines;
- AI development workstations;
- creator workstations;
- local-model PCs;
- high-performance GPU/NPU workstations.

## Performance principles

- inactive experience shells are not loaded simultaneously;
- shared services/libraries are reused across profiles;
- AI visualization surfaces are lazy-loaded;
- large local model weights are not bundled into the default ISO without explicit reason/license review;
- hardware acceleration is used where supported;
- reduced-effects mode is available;
- cold login, idle RAM, compositor latency and battery impact belong in hardware validation;
- background Agent work must be resource-controlled and visible to the user.

## Security principles

Desktop UI cannot bypass the core runtime simply because it is a trusted-looking KINGAI application.

The same path applies:

```text
Desktop action
  ↓
Agent Identity
  ↓
Capability Policy
  ↓
Approval if required
  ↓
Constrained Executor
  ↓
Audit
```

The desktop should explain permissions better than a terminal does, not weaken them.

---

# 中文

KINGAI OS Desktop 就是正式的 **个人电脑 / PC 版本**，不会另外维护第二个 PC Profile。

它面向家庭电脑、办公电脑、开发者电脑、创作者工作站和本地 AI 工作站。

## 一个底座，三套体验

```text
KINGAI Desktop Core
├── KINGAI Intelligence
├── KINGAI Flow
└── KINGAI Classic
```

三个体验共享同一套：

- Wayland / KWin / Plasma 6；
- 通知、文件、搜索、设置；
- Agent Center；
- Task Graph；
- Approval；
- Memory；
- Model；
- Health；
- Audit。

## D5 桌面重点

桌面不再只是“好看的 Linux UI”，而是把系统智能能力做成可理解、可操作的原生界面：

```text
Agent Center
Task Center
Approval Center
Memory Center
Model Center
Automation
System Health
```

例如 AI 请求安装软件、修改网络、重启服务时，应出现明确的系统审批卡片，而不是后台偷偷执行。

## KINGAI Intelligence

旗舰体验把 Agents、Tasks、Approvals、Memory、Knowledge、Models、Files、Apps、Automation 作为桌面一级对象。

目标是让用户从文件、项目、任务、智能体、知识和记忆之间自然切换，而不是每次都先打开聊天机器人。

## KINGAI Flow

现代、空间化、Dock 导向的工作流，但与其他体验共享同一个安全与 Runtime Core。

## KINGAI Classic

保留传统任务栏、应用菜单和熟悉的 PC 交互，降低普通用户迁移门槛；Classic 不会获得更宽松的安全规则。

## 核心原则

Desktop UI 只能把权限解释得更清楚，不能绕过 Policy。

最终路径仍然是：

```text
用户动作
 ↓
Agent Identity
 ↓
Capability Policy
 ↓
需要时 Approval
 ↓
受限 Executor
 ↓
Audit
```
