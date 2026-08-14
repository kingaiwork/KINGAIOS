# Building KINGAI OS

**Development line:** D5 Alpha Runtime Foundation

KINGAI OS is built from one shared codebase into four official distribution forms: Server, Desktop, IoT/Edge and Container.

## Requirements

Core development:

```text
Go 1.25+
Bash
GNU coreutils
Git
```

Bootable image work additionally uses tools such as `mmdebstrap`, `xorriso`, GRUB tooling, SquashFS and QEMU depending on the target workflow.

Container work requires Docker with Buildx.

## Core build

```bash
make check
make build
```

Cross-build foundation binaries:

```bash
bash scripts/build-foundation.sh
```

Outputs are written to `out/foundation/`.

## Official profiles

```text
profiles/server.yaml
profiles/desktop.yaml
profiles/iot.yaml
profiles/container.yaml
```

`desktop` is the personal-computer/PC edition. There is no separate `pc` build profile.

## Server rootfs

```bash
sudo bash scripts/build-rootfs.sh server amd64 dist
```

The rootfs pipeline also has an arm64 composition path. A rootfs target does not automatically mean a bootable release artifact has passed all architecture-specific gates.

## Desktop / personal-computer rootfs

```bash
sudo bash scripts/build-rootfs.sh desktop amd64 dist
```

Desktop installs the Plasma 6 / KWin Wayland / SDDM / Qt 6 foundation plus KINGAI desktop assets and first-run experience.

## Server or Desktop Live ISO

Build rootfs first, then:

```bash
sudo bash scripts/build-live-iso.sh server dist
sudo bash scripts/build-live-iso.sh desktop dist
```

The current Live ISO script is amd64-focused. Other architectures require separate bootloader and hardware validation before they are advertised as release targets.

## IoT / Edge image

```bash
sudo bash scripts/build-iot-image.sh arm64 dist
sudo bash scripts/build-iot-image.sh amd64 dist
```

Generic Edge images are architecture artifacts. Concrete device support still requires a validated Device Pack.

## Docker / OCI Container

Single-architecture local build:

```bash
docker build -f container/Dockerfile -t kingai-os:dev .
```

Multi-architecture Buildx verification:

```bash
KINGAI_CONTAINER_PLATFORMS=linux/amd64,linux/arm64 \
  bash scripts/build-container.sh
```

Export an OCI archive:

```bash
KINGAI_CONTAINER_OUTPUT=dist/kingai-os.oci.tar \
KINGAI_CONTAINER_PLATFORMS=linux/amd64,linux/arm64 \
  bash scripts/build-container.sh
```

The container image:

- runs `kingaid` directly instead of requiring systemd;
- runs as non-root `_kingai` by default;
- exposes no management TCP port;
- persists `/var/lib/kingai` and `/var/log/kingai` as volume targets;
- uses `/run/kingai/kingaid.sock` for local management.

## D5 runtime development

Core runtime components are ordinary Go packages and are tested with:

```bash
go test ./...
go vet ./...
```

Foundation CI also performs a local Unix-socket integration test covering:

- policy evaluation;
- peer identity isolation;
- Memory put/list;
- Task Graph create/transition/list;
- approval request/approve/consume;
- one-time approval reuse rejection;
- model-router fail-closed behavior when no candidate exists;
- audit and sanitized public status.

The Container CI job additionally builds the Docker image, starts `kingaid`, and exercises CLI runtime operations inside the image.

## Build stages for bootable OS artifacts

```text
1. Source/version resolution
2. Base rootfs construction
3. KINGAI binaries
4. Profile package composition
5. Policy / agent / model configuration
6. KINGAI branding
7. Boot / installer composition
8. Desktop or Edge specialization
9. SBOM / package inventory
10. Checksum / manifest
11. VM boot/install verification
12. Update / rollback / recovery validation
13. Release-gate freshness check
14. Release routing
```

## Release truth

`release/gates.json` is fail-closed. A source feature must not be described as production-ready merely because it compiles.

Release channels:

```text
dev -> beta -> rc -> stable
```

## Artifact routing

```bash
bash scripts/publish-artifact.sh <artifact>
```

The publishing helper computes SHA-256 and can route large artifacts to Cloudflare R2 when the required credentials and release gates are present.

See:

- `docs/R2.md`
- `docs/RELEASE-POLICY.md`
- `docs/STATUS.md`
- `docs/PLATFORMS.md`

---

# 中文

KINGAI OS 当前从同一个仓库构建四种正式形态：

```text
Server
Desktop（个人电脑 / PC）
IoT / Edge
Container（Docker / OCI）
```

## Core

```bash
make check
make build
bash scripts/build-foundation.sh
```

## Server

```bash
sudo bash scripts/build-rootfs.sh server amd64 dist
sudo bash scripts/build-live-iso.sh server dist
```

## Desktop / PC

```bash
sudo bash scripts/build-rootfs.sh desktop amd64 dist
sudo bash scripts/build-live-iso.sh desktop dist
```

Desktop 就是个人电脑版本，不另外创建 `pc` Profile。

## IoT / Edge

```bash
sudo bash scripts/build-iot-image.sh arm64 dist
sudo bash scripts/build-iot-image.sh amd64 dist
```

通用镜像不等于具体硬件已经支持；具体设备必须通过 Device Pack 和真实启动验证。

## Docker / OCI

```bash
docker build -f container/Dockerfile -t kingai-os:dev .
```

多架构：

```bash
KINGAI_CONTAINER_PLATFORMS=linux/amd64,linux/arm64 \
  bash scripts/build-container.sh
```

Container 默认非 root 运行 `kingaid`，不开放管理 TCP 端口，Memory / Task / Approval / Audit 数据使用持久化目录。

## D5 CI

当前 CI 已覆盖 Runtime 闭环测试和 Docker 实际启动测试。没有通过 CI、VM 或 Release Gate 的能力不会被写成 Stable 已完成。
