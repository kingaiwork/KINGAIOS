# KINGAI OS D5 Local Runtime API

**Status:** D5 Alpha Runtime Foundation / Pre-Alpha  
**Transport:** local Unix domain socket  
**Default socket:** `/run/kingai/kingaid.sock`

The local Runtime API is an operating-system interface, not a public unauthenticated TCP API. Peer identity comes from Unix peer credentials and participates in authorization.

## Process health and readiness

### `GET /healthz`

Liveness only. A successful response means `kingaid` is running and can answer requests.

### `GET /readyz`

Readiness is stricter than liveness. The response contains:

```json
{
  "ready": true,
  "status": "ready",
  "components": [
    {"name":"agent_registry","required":true,"ok":true,"status":"ready"},
    {"name":"audit_and_state","required":true,"ok":true,"status":"writable"},
    {"name":"execution_broker","required":false,"ok":true,"status":"not_required"},
    {"name":"model_providers","required":false,"ok":true,"status":"not_configured"},
    {"name":"policy","required":true,"ok":true,"status":"ready"},
    {"name":"runtime_adapters","required":false,"ok":true,"status":"not_configured"}
  ]
}
```

A failed **required** component returns HTTP 503. Optional model/provider or adapter capability can be absent without blocking the local governance core.

Bootable Server/Desktop packages set `KINGAI_REQUIRE_EXECD=true`. Container keeps ExecD optional because it deliberately does not include the host privileged execution broker.

## Runtime status

### `GET /v1/status`

Sanitized local status. Current fields include:

- product/version/architecture;
- runtime readiness status;
- Policy / Approval / Task / Memory state;
- task run budget;
- constrained ExecD status;
- agent count;
- model strategy/candidate/provider health counts;
- adapter count;
- audit state.

Sensitive prompts, secrets, tokens, credentials and memory payloads must not appear in status output.

## Policy

### `POST /v1/policy/evaluate`

```json
{
  "agent": "system-ops",
  "capability": "package.install",
  "target": "nginx",
  "approval_id": "optional-one-time-approval-id"
}
```

Owner identity comes from peer UID. Clients cannot submit trusted `owner` or `approved` flags.

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

### `GET /v1/approval/list`

Developer implementation currently requires local UID 0.

### `POST /v1/approval/decision`

```json
{"id":"approval-id","action":"approve"}
```

or `deny`.

Approval is bound to Agent + Capability + Target Hash + Peer UID, expires, and can be consumed only once.

## Constrained execution

### `POST /v1/execution/run`

```json
{
  "agent":"system-ops",
  "capability":"service.restart",
  "target":"nginx.service",
  "approval_id":"optional-id"
}
```

The request passes central Policy/Approval before `kingaid` delegates to `kingai-execd`. Execution results contain an `execution_id` receipt and timing evidence. KINGAI OS does not expose a generic privileged AI shell endpoint.

## Memory

### `POST /v1/memory/put`

```json
{
  "agent":"main",
  "kind":"task",
  "sensitivity":"private",
  "metadata":{"layer":"M2"},
  "data":{"note":"example"}
}
```

### `GET /v1/memory/list`

### `POST /v1/memory/search`

### `POST /v1/memory/delete`

Memory ownership is derived from local peer UID. D5 metadata spans M0 Context through M6 Evolution; M6 data does not imply self-granted execution authority.

## Model Fabric

### `POST /v1/model/select`

```json
{"capability":"chat","private":true,"offline":false}
```

Selection now goes through the Provider Registry. Private/offline requests exclude remote candidates. No configured/eligible provider returns HTTP 503 instead of silently using a cloud service.

### `GET /v1/model/status`

Returns Provider Status records including health, consecutive failures, last error and update time.

Current default `configs/models.json` intentionally contains no provider, so the default system remains fail-closed and performs no implicit model-network discovery.

## Runtime Adapters

### `GET /v1/runtime/adapters`

Returns adapter lifecycle snapshots. Lifecycle states are:

```text
registered
starting
healthy
degraded
stopping
stopped
```

External runtimes remain replaceable adapters. A capability-probe or execution failure marks an adapter degraded rather than silently routing through it.

## Task Graph and Scheduler

### `POST /v1/tasks/create`

```json
{
  "goal":"inspect then restart service",
  "agent":"system-ops",
  "steps":[
    {"id":"inspect","title":"Inspect service","status":"created"},
    {"id":"restart","capability":"service.restart","target":"nginx.service","depends_on":["inspect"],"status":"created"}
  ]
}
```

### `GET /v1/tasks/list`

### `POST /v1/tasks/transition`

### `POST /v1/tasks/step/transition`

### `POST /v1/tasks/run`

The scheduler advances dependency-ready executable steps through Policy → Approval → constrained execution → result state.

To keep one request bounded, a `task run` call has a step-processing budget. Default: `64`; environment: `KINGAI_TASK_RUN_BUDGET`; hard maximum: `1024`. Reaching the budget returns HTTP 429 with the partially advanced task so execution can continue in another call. Context cancellation is honored before further dispatch.

## Trust path

```text
local Unix connection
  ↓
SO_PEERCRED UID
  ↓
Agent Registry
  ↓
Capability Policy
  ↓
Approval when required
  ↓
Task Scheduler / ExecD
  ↓
Result + Audit + Memory
```

## CLI surface

Relevant commands include:

```text
kingai status --json
kingai policy check ...
kingai approval ...
kingai execution run ...
kingai memory ...
kingai model select ...
kingai model status
kingai runtime adapters
kingai task ...
```

Breaking API changes remain possible while the project is Pre-Alpha.

---

# 中文摘要

KINGAI OS D5 Runtime API 默认只通过本机 Unix Socket 提供。

现在明确区分：

- `/healthz`：进程是否活着；
- `/readyz`：核心是否具备接任务条件；
- `/v1/status`：给 CLI / Desktop 使用的脱敏系统状态。

Readiness 中 Policy、Agent Registry、Audit/State 是核心条件；Server/Desktop 还要求 ExecD。模型和外部 Adapter 是可选能力，因此默认没有模型 Provider 时 OS 仍可正常 Ready，同时模型请求继续 503 Fail-Closed。

Task Scheduler 单次运行默认最多处理 64 个可执行 Step，并响应取消，防止巨大 Task 长时间占用 Runtime。

新增的 Provider Health 和 Adapter Lifecycle 只增强可观测性，不引入新的后台守护进程，也不会自动连接任何云模型或外部 Agent 框架。
