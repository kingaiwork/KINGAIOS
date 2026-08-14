# KINGAI OS Desktop Experience Contract

## Product rule

`KINGAI OS Desktop` is the single personal-computer edition of KINGAI OS.

PC, personal computer, home computer, office computer, developer workstation, creator workstation and local-AI workstation are deployment/use-case descriptions of the Desktop edition. They are not separate build profiles.

The repository must not introduce a `profiles/pc.yaml` profile.

## Platform profile vs Desktop experience

KINGAI OS has platform profiles such as:

- `profiles/server.yaml`
- `profiles/desktop.yaml`
- `profiles/iot.yaml`
- `profiles/container.yaml`

Desktop then provides three switchable user experiences inside the one Desktop platform:

```text
KINGAI OS Desktop
└── KINGAI Desktop Core
    ├── KINGAI Intelligence
    ├── KINGAI Flow
    └── KINGAI Classic
```

The three experiences are not separate operating systems and are not separate platform profiles. They share the same kernel, Plasma/Wayland foundation, KINGAI Runtime, Policy, Approval, Task, Memory, Model and Audit services.

## Experience manifests

Machine-readable Desktop experience metadata lives under:

```text
desktop/experiences/
├── intelligence.json
├── flow.json
└── classic.json
```

Each manifest declares:

- schema version;
- stable experience ID;
- product name and description;
- Plasma look-and-feel package;
- Plasma layout script;
- interaction mode;
- whether it is the default recommendation;
- primary desktop surfaces.

`KINGAI Intelligence` is the single default recommendation for first run. The user can choose Flow or Classic instead and can switch later.

## Experience definitions

### KINGAI Intelligence

AI-first desktop experience for agents, tasks, approvals, memory, knowledge, models and automation. This is the flagship and default experience.

### KINGAI Flow

Modern workspace-first experience with centered dock interaction and a lighter spatial workflow.

### KINGAI Classic

Familiar PC experience using an application menu, taskbar and system tray while preserving the same KINGAI governance and runtime underneath.

## Runtime switching

The KINGAI CLI remains the system contract for selecting and applying a Desktop experience:

```text
kingai desktop list
kingai desktop show
kingai desktop set kingai-intelligence
kingai desktop set kingai-flow
kingai desktop set kingai-classic
kingai desktop apply
```

Unknown experience IDs fail closed. Before a live experience is applied, the Desktop manager verifies that the trusted manifest, theme package and layout script agree with the built-in experience contract.

Per-user selection is stored in `kingai-desktop.ini` with private file permissions.

## Repository guardrail

Run:

```text
make desktop-validate
```

or:

```text
bash scripts/validate-desktop.sh
```

The validator fails when:

- a separate PC profile is added;
- legacy `desktop/profiles/` terminology reappears;
- one of the three official experiences is missing or an unexpected fourth manifest is added;
- more than one default exists;
- Intelligence is no longer the default;
- manifest IDs/themes/layouts do not match the official contract;
- a referenced Plasma theme or layout file is missing.

GitHub Actions runs the same contract automatically for Desktop-related changes.

---

# 中文说明

`KINGAI OS Desktop` 就是唯一的个人电脑 / PC 版本，不再另外维护 `PC Profile`。

家庭电脑、办公电脑、开发者电脑、创作者工作站、本地 AI 工作站都属于 Desktop 的使用场景，而不是新的平台版本。

平台级 Profile 只保留 Server、Desktop、IoT/Edge、Container 等正式平台。Desktop 内部的三套界面统一称为 **Desktop Experiences / 桌面体验**：

- `KINGAI Intelligence`：AI 原生、默认推荐；
- `KINGAI Flow`：现代 Dock / 工作区体验；
- `KINGAI Classic`：传统任务栏 / 应用菜单体验。

三套体验共享同一套 KINGAI Desktop Core 和 Policy / Approval / Task / Memory / Model / Audit 安全与智能底座，只改变交互方式，不改变权限模型。
