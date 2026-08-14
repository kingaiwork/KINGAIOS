# KINGAI Intelligence — Agents Center

Agents Center is the peer-aware Agent identity view inside KINGAI Intelligence.

It exists to answer two different questions without confusing them:

1. Which Agent identities are defined by this KINGAI OS installation?
2. Which of those identities may the current local peer actually use?

Visibility is not authority.

## Private data path

Agents Center reads only the per-user Desktop snapshot:

```text
KINGAI Intelligence
        ▲
        │ 0600
$XDG_RUNTIME_DIR/kingai/desktop-private.json
        ▲
        │ ordinary user process
kingai-desktop-bridge
        ▲
        │ GET /v1/desktop/private
        │ AF_UNIX
kingaid
        │
        ├── SO_PEERCRED UID
        ├── Agent Registry Definitions()
        └── agentIdentityAllowed(agentID, username, uid)
```

The authorization label shown in the UI is derived by `kingaid` after the Unix peer identity is known. The bridge does not invent or upgrade Agent authority.

## Snapshot schema

The Desktop Agent summary contains only:

```text
id
authorization role
capability count
authorized_for_peer
```

The actual capability names are deliberately excluded from the Desktop snapshot. A user seeing a privileged Agent therefore does not receive a convenient capability inventory or any reusable authorization material.

The UI renders each identity as either:

```text
AUTHORIZED
LOCKED
```

`LOCKED` means the Agent is part of the system definition but the current peer identity cannot use that Agent identity.

For example, an ordinary desktop user may see `system-ops` as a privileged system Agent while it remains locked. This does not weaken the runtime check.

## Execution remains independently authorized

Even an Agent shown as `AUTHORIZED` does not imply every action is allowed.

Every requested operation still follows:

```text
Peer identity
  ↓
Requested Agent identity
  ↓
Agent manifest capability declaration
  ↓
Capability Policy
  ↓
Approval when required
  ↓
Constrained execution
  ↓
Audit
```

Agents Center is explanatory UI over the security model, not a second authority system.

## Registry integrity

`internal/agent.Registry` now retains read-only Agent definitions in addition to the capability lookup map.

`Definitions()` returns a defensive copy. Modifying the returned definitions or capability slice cannot modify the authorization registry.

Duplicate capability names are removed while the registry is built. If duplicate Agent IDs are supplied, the latest definition replaces the earlier one consistently in both the inspection metadata and authorization map.

## Desktop redaction contract

The server-side Desktop Snapshot Builder converts full Agent definitions to `AgentSummary` objects containing:

- bounded Agent ID;
- bounded role string;
- declared capability count;
- peer authorization boolean.

It does not copy capability names.

Repository tests deliberately define capabilities such as filesystem and service-management operations and assert those names never appear in the serialized Desktop Agent summary.

## UI behavior

`desktop/intelligence/AgentsCenter.qml`:

- reads only `desktop-private.json` from `StandardPaths.RuntimeLocation`;
- validates snapshot schema and product name;
- marks data stale after the Desktop Bridge stops updating;
- rejects unreasonable future timestamps;
- shows Agent ID, role, capability count and Authorized/Locked state;
- never connects directly to the raw Agent, Task, Memory or Approval APIs.

## Product principle

KINGAI OS Desktop should make authority easier to understand without making authority easier to bypass.

That is why the UI can explain that a privileged Agent exists while still showing it as locked for the current user.
