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

The flagship AI-first experience. Phase 2 exposes eight first-class centers:

```text
Home
Agents
Tasks
Approvals
Memory
Models
Automations
System Health
```

Its primary managed Plasma panel is vertical on the left and gives the KINGAI Agent Center priority placement. The full Intelligence application is documented in `docs/DESKTOP-INTELLIGENCE.md`.

### KINGAI Flow

Modern workspace-first experience with centered dock interaction and a lighter spatial workflow. Its managed panel is content-sized, centered, dodge-windows enabled and includes a Workspace Pager when that Plasma widget is available.

### KINGAI Classic

Familiar PC experience using a full-width bottom taskbar, application menu and system tray. A Show Desktop control is included when the corresponding Plasma widget is available.

## Managed layout convergence

Desktop switching must be repeatable. Switching Intelligence → Flow → Classic → Intelligence must not keep appending old widgets.

Each official layout script therefore:

1. searches for a panel marked `kingaiManaged=true`;
2. adopts the first existing Plasma panel only when no KINGAI-managed panel exists yet;
3. clears widgets from that managed panel;
4. applies the selected experience geometry and behavior;
5. records `kingaiExperience=<experience-id>` on the panel;
6. adds only widgets belonging to the selected official experience.

User-created secondary panels are not selected for rebuilding once a KINGAI-managed panel exists.

This gives KINGAI OS a convergent system-owned shell layout without treating every user-created Plasma panel as disposable.

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

### Graphical switching

KINGAI OS Desktop also installs:

```text
KINGAI Desktop Experience
```

This graphical settings application reuses the first-run selector in `--settings` mode. It shows the current experience and lets the user choose Intelligence, Flow or Classic.

The UI only stages the selection. The launcher commits it through:

```text
kingai desktop set <experience>
```

The command applies trusted assets first and persists the new selection only after successful theme/layout application. If switching fails, the previous persisted selection remains authoritative.

Selecting the already-active experience is a no-op rather than an unnecessary desktop reload.

## First-run transaction

First-run selection uses the same safety contract:

```text
KINGAI Welcome
  ↓
Temporary cache selection
  ↓
Validate exact official experience ID
  ↓
kingai desktop set
  ↓
Apply trusted theme/layout
  ↓
Persist only on success
```

A failed application never leaves a false “configured” state that suppresses Welcome on the next login.

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
- a referenced Plasma theme or layout file is missing;
- the Intelligence shell or its eight centers disappear;
- the public desktop shell directly reaches private Task, Approval or Memory list APIs;
- the `kingai://` scheme or launcher contract is broken.

GitHub Actions runs the same contract automatically for Desktop-related changes. Desktop RootFS Smoke additionally verifies that the shell, launchers, Plasma assets and Qt QML runtime dependencies are actually present in the generated Ubuntu 26.04 Desktop root filesystem.

---

# 中文说明

`KINGAI OS Desktop` 就是唯一的个人电脑 / PC 版本，不再另外维护 `PC Profile`。

家庭电脑、办公电脑、开发者电脑、创作者工作站、本地 AI 工作站都属于 Desktop 的使用场景，而不是新的平台版本。

平台级 Profile 只保留 Server、Desktop、IoT/Edge、Container 等正式平台。Desktop 内部的三套界面统一称为 **Desktop Experiences / 桌面体验**：

- `KINGAI Intelligence`：AI 原生、默认推荐，现阶段包含 Home / Agents / Tasks / Approvals / Memory / Models / Automations / System Health 八个中心；
- `KINGAI Flow`：现代 Dock / 工作区体验；
- `KINGAI Classic`：传统任务栏 / 应用菜单体验。

三套体验共享同一套 KINGAI Desktop Core 和 Policy / Approval / Task / Memory / Model / Audit 安全与智能底座，只改变交互方式，不改变权限模型。

系统现在还提供图形化 `KINGAI Desktop Experience` 设置入口。用户切换体验时，系统会先应用可信主题和布局，成功后才保存选择；失败不会破坏之前已经生效的桌面配置。

三套布局都采用 KINGAI managed panel 机制。切换体验会重建受 KINGAI 管理的主面板，而不会不断累积旧组件；用户后来新增的其他独立 Plasma 面板不会因为切换体验而被当作 KINGAI 主面板反复重建。
