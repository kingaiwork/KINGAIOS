# KINGAI OS IoT Device Handler Protocol

This protocol is the privileged hardware boundary for KINGAI OS IoT / Edge.

`kingaid` remains non-root and keeps `PrivateDevices=yes`. It never receives a generic shell, raw device access, or broad Linux capabilities. A Device Pack declares exact `device.*` capabilities and exact resources. After Agent authorization, central Policy, and Approval, the in-process Device Broker forwards the request over a private local Unix socket to the declared hardware handler.

## Trust path

```text
Agent / Task
    |
    v
Peer Identity
    |
    v
Agent capability eligibility
    |
    v
Central Policy + Approval
    |
    v
Device Pack exact capability + resource
    |
    v
KINGAI Device Broker (non-root, AF_UNIX only)
    |
    v
/run/kingai-device/<handler>.sock
    |
    v
Board-specific least-privileged handler
    |
    v
GPIO / I2C / SPI / camera / NPU / GPU / motor / power / firmware API
```

## Manifest installation

Production manifests live in:

```text
/etc/kingai/device-packs/*.json
```

They must be regular files, owned by root, and not writable by group or other users. Device Pack v2 validation must pass. A pack is accepted only for the running CPU architecture. If the manifest contains `board_ids`, `KINGAI_DEVICE_BOARD_ID` must be set to an exact matching board ID.

`security.signed_manifest=true` is a release attestation, not a substitute for provisioning verification. The installer or fleet provisioning layer must cryptographically verify the release metadata before placing the root-owned manifest on the device.

Changing Device Packs requires a `kingaid` restart so the capability and policy index is rebuilt from trusted local state.

## Handler socket

A manifest handler such as:

```json
{
  "id": "device.gpio.read",
  "handler": "gpio-read",
  "resources": ["gpio:17"]
}
```

maps only to:

```text
/run/kingai-device/gpio-read.sock
```

The handler ID is a logical identifier. It is never interpreted as an executable path or shell command.

The `/run/kingai-device` namespace must be root-owned and not group/world writable. Each handler socket must be a Unix socket, must not be a symlink, must be mode `0600`, and must be owned by the same unprivileged UID that runs `kingaid` (`_kingai` in the standard image). A privileged handler can bind the socket and then `chown` it to `_kingai` before accepting requests. This prevents unrelated local users or groups from bypassing Policy by connecting directly.

Handlers should additionally verify `SO_PEERCRED` and accept only the expected `_kingai` UID.

## HTTP over AF_UNIX

Handlers implement:

```text
POST /v1/execute
Content-Type: application/json
```

Request:

```json
{
  "agent": "system-ops",
  "capability": "device.gpio.read",
  "target": "gpio:17",
  "arguments": {}
}
```

Response:

```json
{
  "ok": true,
  "data": {
    "value": 1
  },
  "message": ""
}
```

The Device Broker generates the trusted execution ID and timestamps. Handler-supplied agent, capability, target, execution ID, and timing fields are not trusted as authority.

The handler must validate the request again. Defense in depth is required even though the Broker already enforces exact manifest resources.

## Resource rules

Resources are exact identifiers. Wildcards, shell syntax, and command substitution are forbidden by the Device Pack validator.

Good examples:

```text
gpio:17
i2c:/dev/i2c-1@0x48
camera:front
npu:0
motor:left
power:system
```

A request for `gpio:18` is denied when only `gpio:17` is declared. The Broker never expands `*`, paths, globs, environment variables, or shell expressions.

## Handler privilege model

Use the smallest privilege set that can operate the hardware:

- telemetry handler: dedicated unprivileged user plus read-only device group;
- GPIO/I2C/SPI handler: only the corresponding device group or narrowly scoped capability;
- camera/NPU/GPU handler: vendor device nodes and runtime groups only;
- robot motor/power handler: isolated service with explicit safety interlocks;
- firmware/update handler: separate high-risk service, explicit approval, signed input verification, and rollback support.

Do not implement a handler as `sh -c`, arbitrary command execution, a generic root RPC endpoint, or a pass-through to `/dev`.

## Failure behavior

KINGAI OS fails closed for Device Packs:

- wrong CPU architecture -> pack rejected;
- wrong or missing required board ID -> pack rejected;
- duplicate capability across packs -> runtime initialization rejected;
- untrusted manifest ownership or mode -> pack rejected;
- undeclared resource -> request denied;
- missing, symlinked, incorrectly owned, or overly permissive handler socket -> request denied;
- handler timeout or protocol error -> execution fails and the Task remains failed/blocked according to the Scheduler.

Server and Desktop editions do not enable this embedded Device Broker by default and continue to use their normal constrained ExecD path.
