# KINGAI OS Container

KINGAI OS Container is the Docker / OCI distribution of the shared KINGAI Runtime. It is designed for Docker, OCI runtimes, cloud containers, CI runners and Homelab deployments while keeping the same Policy, Approval, Task, Memory, Model and Audit core used by the bootable KINGAI OS editions.

## Runtime contract

- `kingaid` runs directly as PID 1; no systemd is required inside the container;
- the image user is `_kingai:kingai`, mapped to fixed UID/GID `10001:10001`;
- no management TCP port is declared, exposed or published by default;
- local management stays on `/run/kingai/kingaid.sock`;
- persistent runtime state lives under `/var/lib/kingai`;
- persistent audit and log data lives under `/var/log/kingai`;
- `/run/kingai` and `/tmp` are ephemeral;
- `linux/amd64` and `linux/arm64` are first-class Buildx targets;
- Policy, Approval, Task Graph/Scheduler, Memory, Model Router and Audit remain enabled;
- the healthcheck must reach the live `kingaid` Unix-socket API and cannot succeed through the CLI offline fallback.

The container profile is defined in `profiles/container.yaml` and is the machine-readable source for this deployment contract.

## Local Docker

Build:

```bash
docker build -f container/Dockerfile -t kingai-os:dev .
```

Run with persistent data and logs:

```bash
docker run -d --name kingai-os \
  -v kingai-state:/var/lib/kingai \
  -v kingai-logs:/var/log/kingai \
  kingai-os:dev
```

Check runtime status:

```bash
docker exec kingai-os kingai status --json
```

Create memory and a task:

```bash
docker exec kingai-os \
  kingai memory put task '{"note":"hello from container"}'

docker exec kingai-os \
  kingai task create main verify container runtime
```

No `-p` option is required because the management plane is intentionally local to the container through the Unix socket.

## Hardened Docker Compose

From the repository root:

```bash
docker compose -f container/compose.yaml up -d --build
```

The supplied Compose configuration:

- keeps the root filesystem read-only;
- drops all Linux capabilities;
- enables `no-new-privileges`;
- limits the process count;
- runs with the image's `_kingai` account at UID/GID `10001:10001`;
- uses tmpfs for `/run/kingai` and `/tmp`;
- persists `/var/lib/kingai` and `/var/log/kingai` in named volumes;
- publishes no management TCP port.

## Podman / OCI / Homelab

The image has no Docker-specific runtime dependency. A basic rootless Podman deployment can use the same persistence contract:

```bash
podman volume create kingai-state
podman volume create kingai-logs

podman run -d --name kingai-os \
  --read-only \
  --security-opt=no-new-privileges \
  --cap-drop=ALL \
  --tmpfs /run/kingai:rw,nosuid,nodev,noexec,mode=0750,uid=10001,gid=10001 \
  --tmpfs /tmp:rw,nosuid,nodev,noexec,mode=1777 \
  -v kingai-state:/var/lib/kingai \
  -v kingai-logs:/var/log/kingai \
  ghcr.io/kingaiwork/kingai-os:beta
```

For NAS and Homelab platforms, keep the same rules: persist only the state/log roots, do not expose an arbitrary management TCP port, do not mount the host root filesystem and do not mount the Docker/Podman control socket into KINGAI OS Container.

## Multi-architecture Buildx

Create a local `linux/amd64 + linux/arm64` OCI archive:

```bash
make container-multiarch
```

or:

```bash
bash scripts/build-container.sh
```

Default output:

```text
dist/kingai-os-<VERSION>.oci.tar
```

Validate the archive explicitly:

```bash
python3 scripts/validate-oci-archive.py \
  dist/kingai-os-<VERSION>.oci.tar \
  --platform linux/amd64 \
  --platform linux/arm64
```

Build only amd64 and load it into the local Docker image store:

```bash
KINGAI_CONTAINER_PLATFORMS=linux/amd64 \
  bash scripts/build-container.sh
```

Choose a custom OCI archive destination:

```bash
KINGAI_CONTAINER_OUTPUT=dist/kingai-os.oci.tar \
KINGAI_CONTAINER_PLATFORMS=linux/amd64,linux/arm64 \
  bash scripts/build-container.sh
```

Publish a developer image to a registry:

```bash
KINGAI_CONTAINER_IMAGE=ghcr.io/kingaiwork/kingai-os \
KINGAI_CONTAINER_TAG=dev \
KINGAI_CONTAINER_PUSH=1 \
  bash scripts/build-container.sh
```

Registry publishing enables BuildKit SBOM and provenance attestations. Multi-architecture builds require a Buildx builder capable of executing the runtime stage for both target architectures; on a Linux host this normally uses binfmt/QEMU for the non-native architecture.

## Runtime smoke test

```bash
make container-test
```

The test validates the runtime contract, not merely whether Docker can build the image. It verifies:

- image configuration uses `_kingai:kingai`;
- the actual process UID/GID is `10001:10001`;
- PID 1 is `kingaid`;
- no Docker `ExposedPorts` are declared;
- no TCP listener is present in the container network namespace;
- read-only root filesystem operation;
- zero Linux capabilities and `no-new-privileges`;
- Policy, Approval, Task, Memory, Model and Audit are enabled;
- Memory and Task state survive container deletion/recreation when the same volumes are reused.

## Kubernetes / cloud container platforms

A secure single-replica example is included at:

```text
container/kubernetes.yaml
```

Apply it after selecting the desired immutable image version and storage class:

```bash
kubectl apply -f container/kubernetes.yaml
```

The reference manifest provides:

- `runAsNonRoot` with UID/GID `10001`;
- `RuntimeDefault` seccomp;
- `allowPrivilegeEscalation: false`;
- read-only root filesystem;
- all Linux capabilities dropped;
- separate state and log PVCs;
- size-limited memory-backed volumes for `/run/kingai` and `/tmp`;
- startup, readiness and liveness probes that must reach `kingaid` through the local Unix socket;
- no Service and no container management port;
- an ingress-deny NetworkPolicy for the KINGAI pod.

For production, pin the image by digest after release verification instead of relying on a moving channel tag.

Do not horizontally scale multiple independent `kingaid` replicas over one local state volume. A clustered backend must define explicit coordination, consistency and ownership semantics first.

## Persistence

Persistent roots:

```text
/var/lib/kingai
/var/log/kingai
```

Current runtime state includes:

```text
/var/lib/kingai/approvals
/var/lib/kingai/memory
/var/lib/kingai/tasks
/var/lib/kingai/models
/var/log/kingai
```

Ephemeral runtime paths:

```text
/run/kingai
/tmp
```

For host bind mounts, ensure writable ownership for UID/GID `10001:10001` or provide an equivalent platform ownership policy.

## Configuration and secrets

Default read-only configuration is shipped under `/etc/kingai`:

```text
/etc/kingai/policy.json
/etc/kingai/agents.json
/etc/kingai/models.json
/etc/kingai/system.json
```

Production deployments may mount reviewed configuration over those paths. Model/API credentials must be injected through the orchestrator secret mechanism; they should never be baked into the image, committed to Git or written into public status output.

## Security boundary

Container mode deliberately does not turn KINGAI into a network-exposed root service.

The default deployment does **not**:

- expose a management TCP port;
- run `kingaid` as root;
- require systemd;
- mount `/var/run/docker.sock` or a Podman control socket;
- mount the host root filesystem;
- request privileged mode or host namespaces.

If host-level privileged operations are ever required, they must be delegated through a separately reviewed constrained Execution Broker so Policy, Approval and Audit controls remain enforceable.

## CI gates

`.github/workflows/container.yml` provides the container pre-release gate:

1. hardened amd64 runtime/persistence smoke test;
2. vulnerability scan that blocks fixable `CRITICAL` findings;
3. real `linux/amd64 + linux/arm64` OCI archive build;
4. OCI index/manifest validation proving both required platforms are present.

The general Foundation CI also exercises the shared runtime so changes cannot silently diverge Container from the Server/Desktop/IoT core.

Run local release-policy regressions with:

```bash
bash scripts/test-container-release-policy.sh
```

## Signed GHCR release pipeline

`.github/workflows/container-release.yml` defines the controlled GHCR release path for:

```text
beta  -> <version>-beta.<n> + :beta
rc    -> <version>-rc.<n>   + :rc
stable-> <version>          + :stable + :latest
```

The release pipeline:

- requires the release commit to be contained in `main`;
- builds and pushes amd64/arm64 by digest;
- emits BuildKit SBOM/provenance;
- generates a CycloneDX SBOM;
- blocks fixable CRITICAL vulnerabilities;
- creates GitHub provenance and SBOM attestations for the pushed digest;
- signs the digest with Sigstore/Cosign keyless OIDC signing;
- immediately verifies the resulting signature against the exact release workflow identity;
- uploads release metadata, CycloneDX evidence and SHA-256 checksums.

Stable promotion is fail-closed and remains disabled while `production_signing_ready` is false in `release/gates.json`.

The first requested preview release is recorded in `.release/container-beta.json` as `0.1.0-beta.1`. The request record does not claim the image has been published; `container_signed_release` stays false until a release workflow actually completes successfully.

A published digest can be verified with Cosign using the identity recorded by the release workflow. Always deploy the verified digest in high-assurance environments.

---

# 中文

KINGAI OS Container 是 KINGAI Runtime 面向 **Docker / OCI / 云端容器 / CI / Homelab** 的统一发行形态，与 Server、Desktop、IoT 共用同一套核心能力，而不是另一套独立系统。

## 默认运行约束

- `kingaid` 直接作为 PID 1，不依赖 systemd；
- 镜像用户 `_kingai:kingai`，固定 UID/GID `10001:10001`；
- 默认不声明、不开放、不映射管理 TCP；
- CLI 通过 `/run/kingai/kingaid.sock` 本地 Unix Socket 管理；
- Policy / Approval / Task / Memory / Model / Audit 全部保留；
- 状态统一持久化到 `/var/lib/kingai`；
- 审计与日志持久化到 `/var/log/kingai`；
- `/run/kingai` 与 `/tmp` 保持临时；
- amd64 / arm64 使用 Buildx 作为正式多架构目标；
- 默认不挂 Docker/Podman Socket、不挂宿主机根目录、不启用 privileged。

## 推荐验证

运行完整容器 Smoke：

```bash
make container-test
```

构建 amd64 + arm64 OCI：

```bash
make container-multiarch
```

Docker Compose：

```bash
docker compose -f container/compose.yaml up -d --build
```

Kubernetes：

```bash
kubectl apply -f container/kubernetes.yaml
```

当前 Kubernetes 基线已经包含非 root、只读根目录、Drop ALL、RuntimeDefault seccomp、启动/就绪/存活探针、状态/日志 PVC、临时内存卷和默认拒绝入站流量的 NetworkPolicy，并且仍然没有创建管理 Service 或管理 TCP 端口。

## CI 与发布

Container CI 不再只是检查“Docker 能不能 build”，而是同时检查：

- 实际非 root 运行；
- `kingaid` 是否为 PID 1；
- 是否意外出现 TCP 监听；
- 六大核心能力是否存在；
- Memory / Task 是否可以跨容器重建保持；
- 是否存在可修复的 Critical 漏洞；
- OCI 归档是否真实包含 amd64 和 arm64。

正式发布链使用 GHCR + BuildKit SBOM/Provenance + CycloneDX + GitHub Attestation + Sigstore/Cosign Keyless Signature。Beta、RC、Stable 采用独立通道规则，其中 Stable 在生产签名门禁未正式打开之前会直接失败，不能绕过。

如果未来需要容器操作宿主机高权限资源，应通过独立、受限并经过 Policy / Approval / Audit 审核的 Execution Broker，而不是直接给 KINGAI Container 无限宿主机权限。
