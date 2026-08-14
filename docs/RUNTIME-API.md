# KINGAI OS D5 Local Runtime API

**Status:** Developer / Pre-Alpha  
**Transport:** local Unix domain socket  
**Default socket:** `/run/kingai/kingaid.sock`

This API is a local operating-system interface. It is not designed as a public unauthenticated TCP API.

Peer identity is derived from Unix peer credentials and is part of authorization decisions.

## Health

### `GET /healthz`

Returns daemon health and source version.

## Status

### `GET /v1/status`

Returns sanitized runtime status suitable for local UI surfaces.

Current fields include:

- product name;
- version;
- architecture line;
- policy status;
- approval/task/memory/model service state;
- registered agent count;
- model strategy/mode;
- audit state.

Sensitive prompt, token, password, secret and memory payload data must not be published through public status.

## Policy

### `POST /v1/policy/evaluate`

Request:

```json
{
  "agent": "system-ops",
  "capability": "package.install",
  "target": "nginx",
  "approval_id": "optional-one-time-approval-id"
}
```

The daemon determines Owner state from peer UID. A client cannot submit `owner` or `approved` fields.

If an approval is supplied, it must match Agent + Capability + Target Hash + Peer UID and must still be approved/unconsumed/unexpired.

## Approval Broker

### `POST /v1/approval/request`

```json
{
  "agent": "system-ops",
  "capability": "package.install",
  "target": "nginx",
  "ttl_seconds": 300
}
```

An approval request is created only when the agent/capability identity is valid and central policy says explicit approval is required.

### `GET /v1/approval/list`

Current developer implementation requires local UID 0.

### `POST /v1/approval/decision`

```json
{
  "id": "approval-id",
  "action": "approve"
}
```

or:

```json
{
  "id": "approval-id",
  "action": "deny"
}
```

Current developer implementation requires local UID 0.

Approval lifecycle:

```text
pending -> approved -> consumed
pending -> denied
pending/approved -> expired
```

Consumed approvals cannot be reused.

## Memory

### `POST /v1/memory/put`

```json
{
  "agent": "main",
  "kind": "task",
  "sensitivity": "private",
  "data": {
    "note": "example"
  }
}
```

Memory owner namespace is derived from local peer UID.

### `GET /v1/memory/list`

Lists records for the current peer namespace.

### `POST /v1/memory/delete`

```json
{
  "id": "memory-record-id"
}
```

Deletes only inside the current peer namespace.

## Model Router

### `POST /v1/model/select`

```json
{
  "capability": "chat",
  "private": true,
  "offline": false
}
```

The router filters unavailable/incompatible candidates. Private or offline mode excludes non-local candidates.

If no eligible candidate exists, the daemon returns a service-unavailable response rather than silently violating the requested constraints.

## Task Graph

### `POST /v1/tasks/create`

Simple task:

```json
{
  "goal": "verify server health",
  "agent": "main"
}
```

Task with steps:

```json
{
  "goal": "inspect then restart service",
  "agent": "system-ops",
  "steps": [
    {
      "id": "inspect",
      "title": "Inspect service",
      "status": "created"
    },
    {
      "id": "restart",
      "title": "Restart service",
      "capability": "service.restart",
      "depends_on": ["inspect"],
      "status": "created"
    }
  ]
}
```

Tasks are bound to the creating peer UID.

### `GET /v1/tasks/list`

Non-root peers see only tasks owned by their peer UID. Root may inspect all tasks in the current developer implementation.

### `POST /v1/tasks/transition`

```json
{
  "id": "task-id",
  "status": "planning"
}
```

Optional completion result:

```json
{
  "id": "task-id",
  "status": "completed",
  "result": {
    "ok": true
  }
}
```

Failure/blocking state may carry an `error` string.

Allowed lifecycle is validated by the Task Graph state machine.

## Current trust model

```text
local Unix connection
  ↓
SO_PEERCRED UID
  ↓
agent identity binding
  ↓
agent capability manifest
  ↓
central policy
  ↓
approval token when required
```

The API intentionally does not treat client-provided JSON as proof of owner identity or approval.

## Future API evolution

Planned additions include:

- Planner endpoints;
- task-step scheduling;
- capability-specific Execution Broker requests/results;
- adapter lifecycle;
- richer Memory search/retrieval;
- model-provider health;
- Desktop-native approval events;
- device/fleet operations.

Breaking API changes remain possible while the project is Pre-Alpha.

---

# 中文摘要

KINGAI OS D5 Runtime API 默认只通过本机 Unix Socket 提供，不是公开 TCP 管理 API。

当前统一接口覆盖：

```text
Health
Status
Policy
Approval
Memory
Model
Task Graph
```

最重要的安全原则是：

- Owner 不能由 JSON 伪造；
- Approved 不能由 JSON 伪造；
- 本机 Peer UID 来自内核；
- Approval 与 Agent / Capability / Target Hash / Peer UID 绑定；
- Approval 一次性消费；
- Task 和 Memory 默认按 Peer UID 隔离；
- 没有合适模型时失败，不擅自违反 Private/Offline 条件。

后续 Server、Desktop、IoT、Container、OpenClaw、MCP 等都应围绕同一套 Runtime Contract 扩展，而不是建立互不兼容的接口。
