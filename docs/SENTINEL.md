# KINGAI OS Sentinel

KINGAI OS Sentinel is the cloud-server security edition of KINGAI OS. It keeps the existing KINGAI governance core and adds a lean, visual security operations layer for proactive defense, threat intelligence, exposure discovery, detection, forensics, incident response and authorized security validation.

## Product position

Sentinel is not a separate operating-system family. It is a Server-derived edition that reuses the same Policy, Approval, Task, Memory, Model, Audit and controlled-execution foundations.

```text
Global security intelligence
CISA KEV · NVD · FIRST EPSS · local telemetry
                 │
                 ▼
┌──────────────────────────────────────────────┐
│ KINGAI OS Sentinel                          │
│ Visual Console · AI Triage · Correlation    │
│ Exposure · Detection · Forensics · Response │
└──────────────────────┬───────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────┐
│ KINGAI Governance                            │
│ Scope · Capability · Policy · Approval       │
│ Task · Audit · Memory · Model                │
└──────────────────────┬───────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────┐
│ Governed execution                           │
│ Host read-mostly · Rootless container · VM   │
│ Authorized validation sandbox                │
└──────────────────────────────────────────────┘
```

## Design goals

- cloud-server first, headless and efficient;
- visual web console without a heavyweight desktop environment;
- loopback-only management console by default;
- current threat intelligence cached locally for offline correlation;
- small trusted base with heavyweight tools delivered as Sentinel Packs;
- rootless containers for disposable tool execution whenever practical;
- active validation limited to explicitly authorized assets;
- unknown targets fail closed;
- privileged changes remain approval-gated and auditable;
- AI can triage, correlate, prioritize and recommend remediation without silently acquiring authority.

## Built-in base

The core image includes the existing KINGAI Server runtime plus a focused set of network, DNS/TLS, packet, audit and rootless-container utilities. The base deliberately avoids installing every specialist security package into the host namespace.

Sentinel Pack classes are defined in `sentinel/packs/catalog.json`:

- Core Defense;
- Threat Intelligence;
- Forensics;
- Malware Analysis;
- Authorized Validation;
- Cloud Security.

This split keeps the cloud image smaller, makes updates less fragile, and lets high-risk or dependency-heavy tools run in disposable sandboxes instead of permanently expanding the trusted computing base.

## Visual console

`kingai-sentinel` serves the read-only console from:

```text
http://127.0.0.1:9443
```

Remote access should use an SSH tunnel or an authenticated reverse proxy. Sentinel does not open a public management listener by default.

The console currently reports:

- node/runtime status;
- feed readiness and last local update time;
- authorization-scope state without exposing scope contents;
- security capability map;
- KINGAI AI/governance posture.

The console service runs without Linux capabilities and with systemd hardening.

## Threat-intelligence pipeline

A systemd timer refreshes official public intelligence once per day with randomized delay. Downloads are validated before atomic replacement of the local cache.

Default feeds:

- CISA Known Exploited Vulnerabilities (KEV);
- NVD CVE API KEV subset;
- FIRST EPSS current scores.

Local cache:

```text
/var/lib/kingai/sentinel/feeds/
```

The feed registry is stored at:

```text
/usr/share/kingai/sentinel/feeds/feeds.json
```

## Authorization boundary

Sentinel defaults to defensive mode. Active validation is disabled for unknown targets.

Example scope file:

```text
/etc/kingai/sentinel-scope.example.json
```

Operational scope file:

```text
/etc/kingai/sentinel-scope.json
```

A scope should represent assets owned by the operator, assets covered by explicit contractual authorization, or isolated laboratory/CTF ranges. Scope records are expected to expire; the default design target is no more than 24 hours for active testing authorization.

Privileged actions continue through KINGAI capability policy and Approval Broker rather than being granted by a dashboard toggle.

## Build

Requirements are the same as the Server rootfs builder plus the packages already documented by `scripts/build-rootfs.sh`.

Build amd64:

```bash
sudo --preserve-env=PATH bash scripts/build-sentinel-rootfs.sh amd64 dist
```

Build arm64:

```bash
sudo --preserve-env=PATH bash scripts/build-sentinel-rootfs.sh arm64 dist
```

Artifacts:

```text
dist/KINGAI-OS-sentinel-amd64-rootfs.tar.zst
dist/KINGAI-OS-sentinel-amd64-rootfs.tar.zst.sha256
dist/KINGAI-OS-sentinel-amd64-rootfs.tar.zst.spdx.json
```

The builder starts from the validated Server rootfs path, adds the Sentinel security layer, regenerates the package inventory and SPDX SBOM, enforces the Sentinel size budget, and then creates the final archive.

## First boot

Useful checks:

```bash
systemctl status kingaid
systemctl status kingai-sentinel
systemctl status kingai-threat-intel.timer
curl http://127.0.0.1:9443/healthz
```

Run an immediate intelligence refresh:

```bash
sudo systemctl start kingai-threat-intel.service
```

For remote administration, tunnel the local console rather than binding it directly to the public interface.

## Security model

Sentinel follows the KINGAI principle:

> Intelligence may be autonomous. Authority remains controlled, auditable and revocable.

The security edition extends that principle to testing workflows: broad tool coverage is compatible with a strong authorization boundary. The system is designed to help defenders validate systems they are permitted to test, not to remove ownership, consent or audit controls.
