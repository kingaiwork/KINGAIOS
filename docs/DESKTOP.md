# KINGAI OS Desktop Architecture

## English

KINGAI OS Desktop uses one shared desktop foundation with multiple KINGAI-owned experience profiles.

### Foundation direction

Initial implementation direction:

```text
Wayland
  + KWin
  + Plasma/Qt/QML foundations where useful
  + KINGAI Desktop Core
  + KINGAI-owned shell, panels, settings and AI surfaces
```

This provides mature Linux graphics/window-management infrastructure while allowing KINGAI to own the visible product experience and AI-native workflow.

### Experiences

```text
KINGAI Desktop Core
├── KINGAI Intelligence
├── KINGAI Flow
└── KINGAI Classic
```

All three profiles share:

- compositor/window manager;
- settings framework;
- file integration;
- notification service;
- global search;
- Agent Panel;
- memory/task/project APIs;
- accessibility layer;
- application launcher backend.

Profiles change layout and interaction instead of installing three independent desktops.

### First-run experience

On first graphical login:

```text
KINGAI Welcome
  -> visual animated previews
  -> Intelligence / Flow / Classic
  -> Try / Select
  -> save per-user profile
  -> launch desktop
```

The selection remains changeable later:

```text
Settings -> Personalization -> Desktop Experience
```

### KINGAI Intelligence

The flagship experience exposes AI-native objects as first-class desktop concepts:

```text
Agents
Memory
Tasks
Projects
Knowledge
Files
Apps
Automation
System status
```

The desktop should make it possible to move naturally between a file, project, agent, memory context and active task without requiring a separate chatbot workflow.

### KINGAI Flow

A spatial, dock-oriented layout with a clean top-level interface and workspace-focused navigation. It is a KINGAI-owned product experience and must use original KINGAI assets.

### KINGAI Classic

A traditional PC workflow using taskbar/application-menu concepts for easier migration from conventional desktop environments.

### Performance rules

- do not load inactive experience shells simultaneously;
- share services and libraries across profiles;
- lazy-load AI visualization surfaces;
- keep heavy local models outside the ISO;
- use hardware acceleration where available;
- provide a reduced-effects mode;
- profile cold login, idle RAM, compositor latency and battery impact in CI/hardware validation.

---

## 中文

KINGAI OS Desktop 使用一个统一桌面底座，再提供三种 KINGAI 自有桌面体验，不安装三套独立桌面环境。

初期技术方向：

```text
Wayland
+ KWin
+ Plasma / Qt / QML 基础能力
+ KINGAI Desktop Core
+ KINGAI 自有 Shell / Panel / Settings / AI UI
```

### 三种体验

- **KINGAI Intelligence**：旗舰 AI 原生桌面，把 Agents、Memory、Tasks、Projects、Knowledge、Files、Apps、Automation 直接变成桌面一级对象。
- **KINGAI Flow**：空间化、Dock导向、简洁现代的 KINGAI 自有工作流。
- **KINGAI Classic**：传统任务栏与应用菜单工作流，降低普通 PC 用户迁移成本。

三个桌面共用窗口系统、设置、通知、搜索、文件接口、Agent Panel、Memory/Task API 和应用启动后端。

### 第一次进入系统

第一次图形登录进入 KINGAI Welcome，以动画/实时缩略展示三个桌面，用户可以预览后选择；设置按用户保存。

进入系统后：

```text
Settings -> Personalization -> Desktop Experience
```

仍然可以随时切换。

### 性能原则

未启用的桌面 Shell 不同时加载；共享底层服务；AI 可视化按需加载；大模型不放 ISO；支持硬件加速与低特效模式，并把首次登录速度、Idle RAM、窗口延迟和电池影响纳入测试。
