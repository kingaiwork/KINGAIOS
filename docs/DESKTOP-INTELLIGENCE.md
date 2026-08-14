# KINGAI Intelligence

KINGAI Intelligence is the flagship AI-first experience inside **KINGAI OS Desktop**, the single personal-computer / PC edition of KINGAI OS.

It is not a separate operating-system profile. It shares the same Desktop Core and the same governed runtime used by KINGAI Flow and KINGAI Classic.

## Current Phase 3 shell

The current shell establishes eight first-class desktop centers:

1. Home
2. Agents
3. Tasks
4. Approvals
5. Memory
6. Models
7. Automations
8. System Health

The application lives under `desktop/intelligence/` and is installed into the Desktop root filesystem through the shared desktop asset tree.

The application launcher is `kingai-intelligence.desktop`. It registers the local `kingai://` deep-link scheme so system surfaces can open a specific center without creating an alternate privileged API.

```text
kingai://home
kingai://agents
kingai://tasks
kingai://approvals
kingai://memory
kingai://models
kingai://automations
kingai://health
```

Unknown or malformed center values fail back to Home.

## Two local data planes

KINGAI Intelligence now separates public aggregate status from per-user private desktop state.

```text
                        kingaid
                          │
           ┌──────────────┴──────────────┐
           │                             │
           ▼                             ▼
/run/kingai/public-status.json     GET /v1/desktop/private
0644 sanitized aggregate          Unix socket + SO_PEERCRED
           │                             │
           │                             ▼
           │                    kingai-desktop-bridge
           │                       ordinary user
           │                             │
           │                             ▼
           │              $XDG_RUNTIME_DIR/kingai/
           │                  desktop-private.json
           │                         0600
           └──────────────┬──────────────┘
                          ▼
                 KINGAI Intelligence
```

### Public status plane

`/run/kingai/public-status.json` is world-readable because it contains only deliberately sanitized aggregate state.

Current examples include:

- runtime health;
- policy mode;
- registered Agent count;
- active/running/waiting/blocked/paused/planning task counts;
- pending approval count;
- model provider count and routing mode;
- local-first Memory mode;
- whether cloud is required for core survival.

It must not contain:

- prompt text;
- task goals;
- task targets;
- approval targets;
- credentials;
- model API keys;
- Memory Data;
- raw audit events.

### Private per-user plane

Private desktop state is generated from:

```text
GET /v1/desktop/private
```

The endpoint exists only on the local `kingaid` Unix socket. `kingaid` derives the caller UID from Unix peer credentials and then performs server-side filtering and redaction before returning anything to the Desktop Bridge.

The current server path is:

```text
SO_PEERCRED UID
  ↓
Task Store ListForPeer(uid)
  ↓
Memory Store Summarize(uid owner namespace)
  ↓
Desktop Snapshot Builder
  ↓
Redacted JSON
```

The bridge therefore does **not** fetch full `/v1/tasks/list` records and strip them afterward. Raw step fields and Memory payloads are removed before the desktop process receives the response.

## KINGAI Desktop Bridge

`kingai-desktop-bridge` is a per-user process installed only in KINGAI OS Desktop.

It:

- connects only to the local Unix-domain `kingaid` API;
- consumes only `/v1/desktop/private`;
- validates schema, product name and returned UID;
- requires the returned UID to equal its own effective UID;
- writes into the user's runtime directory;
- creates the KINGAI private runtime directory with mode `0700`;
- writes `desktop-private.json` atomically with mode `0600`;
- refuses to run as a long-lived root service by default;
- is managed as a systemd user service;
- declares `RestrictAddressFamilies=AF_UNIX`, `NoNewPrivileges=true` and a restrictive umask.

The default output is:

```text
$XDG_RUNTIME_DIR/kingai/desktop-private.json
```

The Desktop UI detects stale private snapshots instead of silently presenting old data as live state.

## Task Center

Task Center combines two different views:

### Aggregate lifecycle cards

Safe system-level counts from public status:

```text
Active
Running
Waiting
Waiting Approval
Planning
Paused
Blocked
```

### My Tasks

The private snapshot may contain, for tasks owned by the current peer UID:

- Task ID;
- Goal;
- Agent ID;
- task status;
- number of steps;
- completed-step count;
- blocked/failed-step count;
- created/updated timestamps.

The private Desktop Task schema deliberately excludes:

- peer UID from each task record;
- raw step title;
- capability;
- target/path;
- approval ID/token;
- execution result;
- raw error text.

Task goal text is bounded before it enters the Desktop Snapshot so an unusually large task cannot cause unbounded UI snapshot growth.

## Memory Center

Memory Center receives **metadata counts only** from the private snapshot.

The server-side Memory Summary contains:

- total non-expired records;
- counts by M0–M6 layer;
- counts by sensitivity class;
- count of records with an expiry.

The seven displayed layers are:

```text
M0 Context
M1 Working
M2 Task
M3 Episodic
M4 Semantic
M5 User / Organization
M6 Evolution
```

Memory Summary does not model the record `data` field at all. Expired records are excluded from the current summary.

## Approval boundary

The public desktop may show that approvals are pending, but KINGAI Intelligence does not weaken the Approval Broker merely to make a convenient UI.

The existing `/v1/approval/list` endpoint remains owner/root-authorized. The Desktop private snapshot does not include approval targets or reusable authorization material.

A future graphical Approval Center must preserve:

```text
Local identity
  ↓
Owner authorization
  ↓
Scoped approval record
  ↓
Agent + Capability + Target Hash + Peer UID
  ↓
Expiration
  ↓
Single-use consumption
  ↓
Audit
```

## Model, Agent, Automation and Health centers

The current Phase 3 shell already gives these centers distinct aggregate views rather than repeating one generic dashboard:

- **Agents**: registered identities, active work, approval pressure and policy posture;
- **Models**: provider count, routing mode, provider-neutral strategy and cloud dependency posture;
- **Automations**: active/planning/paused/blocked task-backed work;
- **System Health**: runtime, policy, blocked workload and local/cloud dependency posture.

Private provider credentials and raw execution state remain outside the public shell.

## Agent Center integration

The Plasma Agent Center remains a compact always-available status surface. It reads only the sanitized public status file and deep-links into:

- KINGAI Intelligence Home;
- Approval Center;
- Task Center.

This keeps the panel widget lightweight while the richer private state lives in the dedicated Intelligence application.

## Experience relationship

```text
KINGAI OS Desktop
│
├── KINGAI Intelligence   ← flagship AI-first shell
├── KINGAI Flow           ← dock / workspace interaction
└── KINGAI Classic        ← familiar PC interaction
        │
        └──── all share KINGAI Core
              Policy · Approval · Task · Memory · Model · Audit
```

Changing experience changes presentation and workflow, not authority.

## Validation

Run:

```bash
make desktop-validate
```

Desktop validation now covers both the product contract and the private-data contract.

It validates, among other things:

- Desktop remains the only PC platform profile;
- exactly three official Desktop Experiences exist;
- KINGAI Intelligence remains the single first-run default;
- trusted theme/layout manifests agree;
- managed Plasma layouts converge rather than accumulating widgets;
- all eight Intelligence centers remain present;
- public UI reads the sanitized public status plane;
- Task and Memory centers use only the per-user runtime snapshot;
- UI code does not bypass the bridge with raw Task/Memory list APIs;
- the bridge consumes only the server-redacted Desktop endpoint;
- the returned private snapshot UID must match the bridge process UID;
- task step targets/capabilities/results and Memory Data are absent from Desktop snapshot schemas;
- launcher deep links are constrained to known centers;
- malformed deep links fall back to Home.

GitHub Actions run the same contract in `.github/workflows/desktop-contract.yml`. Desktop RootFS Smoke additionally builds the Ubuntu 26.04 Desktop root filesystem and verifies that the shell, private centers, Bridge binary, systemd user unit, autostart and QML runtime assets are actually installed.

## Next implementation layer

The next Desktop layer can now build interactive features on top of the established identity boundary:

- Task creation and task detail actions through the peer-scoped runtime API;
- graphical Approval review without relaxing owner authorization;
- Memory search/detail through an explicit private read contract with sensitivity/provenance controls;
- model provider health plus local/cloud/private/offline controls;
- Automation schedules backed by governed Task Graph execution;
- richer Health diagnostics and explicitly approved repair actions;
- native Qt application packaging after the QML product contracts stabilize.

The design rule remains unchanged:

> Intelligence may be convenient and autonomous; authority remains identity-bound, policy-controlled, auditable and revocable.
