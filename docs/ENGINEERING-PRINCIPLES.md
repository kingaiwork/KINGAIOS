# KINGAI OS Engineering Principles

KINGAI OS is an independently engineered AI-native operating-system project. We learn from public research, open standards, security practice and the behavior of mature operating systems, but KINGAI-specific architecture, runtime code, product UI and governance logic are implemented for KINGAI OS.

## 1. Product priorities

Every major design choice is evaluated in this order:

1. **Simple** — fewer moving parts, fewer hidden states, clear failure modes.
2. **Reliable** — deterministic behavior, bounded inputs, explicit timeouts, recoverable state.
3. **Secure** — least privilege, fail closed, separate intelligence from authority.
4. **Efficient** — low idle overhead, bounded background work, local-first execution where practical.
5. **Stable** — compatibility and operational predictability matter more than novelty.

A new technology is not adopted because it is fashionable. It must measurably improve one or more priorities without making the others materially worse.

## 2. Independent implementation

KINGAI OS follows a clean engineering boundary:

- do not copy proprietary product source code, UI assets, private implementation details or protected brand material;
- use public specifications and documented interfaces as interoperability references;
- use open-source dependencies only under compatible licenses and record them in the legal/SBOM process;
- prefer small KINGAI-owned interfaces around external components so they remain replaceable;
- do not make one model vendor, cloud vendor, agent framework or desktop vendor the identity of KINGAI OS.

## 3. Minimal core

The default runtime should stay small.

```text
kingaid       non-root local intelligence/governance hub
kingai-execd  narrowly scoped privileged execution broker where required
```

Logical service boundaries do not automatically become separate daemons. A new process requires a concrete security, lifecycle, isolation or scaling reason.

## 4. Intelligence is not authority

The invariant is:

```text
Goal / model output
      ↓
validated plan
      ↓
Agent identity
      ↓
Capability Policy
      ↓
Approval when required
      ↓
constrained execution
      ↓
result + audit + memory
```

Models and agents never gain authority simply because they produced a convincing answer.

## 5. Bounded execution

Execution paths must have explicit bounds:

- bounded request sizes;
- bounded target sizes;
- deadlines/timeouts;
- process/file-descriptor/memory limits where appropriate;
- capability-specific handlers rather than generic privileged shell APIs;
- unique execution receipts that connect result and audit evidence;
- cancellation and failure propagation.

## 6. Local-first and offline-capable

Core safety functions must not require cloud availability:

- local identity;
- policy;
- approvals;
- task state;
- local memory;
- audit;
- health and recovery;
- local/offline model routing where a compatible model is installed.

Cloud services may accelerate or coordinate the system, but should not become an implicit trust root or survival dependency.

## 7. Replaceable integrations

External runtimes and services are adapters.

Examples include:

- MCP-compatible tools;
- OpenClaw-compatible runtimes;
- Codex-compatible runtimes;
- browsers;
- model providers;
- containers;
- VMs / microVMs;
- future device runtimes.

The KINGAI governance boundary remains above those integrations.

## 8. Four forms, one core

Official platform forms:

```text
Server
Desktop        # personal computer / PC edition
IoT / Edge
Container      # Docker / OCI
```

They share identity, policy, approval, task, memory, model and audit semantics. Platform-specific boot, hardware, UI and delivery behavior must not create weaker security semantics.

## 9. Evidence before claims

KINGAI OS distinguishes:

```text
code exists
CI passes
VM test passes
real hardware test passes
artifact is published
production signing is ready
Stable release is ready
```

One state never implies the next.

## 10. Dependency policy

Prefer the Go standard library and mature OS primitives. Add a dependency only when implementing the same behavior ourselves would increase risk or maintenance cost.

Dependencies must be:

- pinned/reviewable;
- license-compatible;
- represented in SBOM/release review;
- replaceable where practical;
- kept out of privileged code paths unless justified.

## 11. New technology adoption rule

Before adding a new subsystem, answer:

1. What user/system problem does it solve?
2. Can an existing KINGAI primitive solve it more simply?
3. Does it add a daemon, network listener, privileged surface or persistent state?
4. What is the failure mode?
5. How is it tested?
6. How is it removed or replaced later?

If those questions do not have clear answers, the feature stays outside the trusted core.

---

# 中文原则

KINGAI OS 的长期目标不是堆砌技术，而是打造一套 **简单、高效、稳定、可靠、安全** 的 AI 原生操作系统。

核心要求：

- 学习全球公开技术、研究与开放标准，但不复制别人的专有代码、网页代码、品牌素材或私有实现；
- KINGAI Runtime、治理逻辑、UI 与产品架构保持独立实现；
- 能一个进程解决的问题，不为了“微服务化”拆成十个常驻进程；
- 能使用标准库和 Linux 成熟能力解决的问题，不轻易增加第三方依赖；
- Agent 和模型拥有智能，不自动拥有系统权限；
- 高权限能力必须最小化、可审批、可追踪、可限制、可恢复；
- Server、Desktop、IoT/Edge、Container 共用一套安全语义；
- 代码存在、CI 通过、VM 通过、真实硬件通过、公开发布、生产就绪必须分别记录，不能互相代替。
