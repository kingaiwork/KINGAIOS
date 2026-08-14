# KINGAI OS Container

KINGAI OS Container is the Docker/OCI distribution form of the shared KINGAI Runtime. It targets Docker, OCI runtimes, cloud container platforms, CI and homelab deployments while preserving the same Policy, Approval, Task, Memory, Model and Audit core used by the bootable KINGAI OS editions.

## Runtime contract

- `kingaid` runs directly as PID 1; systemd is not required;
- default runtime user is non-root UID/GID `10001:10001`;
- no management TCP port is exposed or published by default;
- local control remains on `/run/kingai/kingaid.sock`;
- persistent runtime data lives under `/var/lib/kingai`;
- persistent audit/log data lives under `/var/log/kingai`;
- amd64 and arm64 are first-class Buildx targets;
- Policy, Approval, Task Graph/Scheduler, Memory, Model Router and Audit remain enabled;
- the image healthcheck requires a live `kingaid` Unix-socket API rather than accepting the CLI offline fallback.

## Local Docker build

```bash
docker build -f container/Dockerfile -t kingai-os:dev .
```

Run with persistent volumes:

```bash
docker run -d --name kingai-os \
  -v kingai-state:/var/lib/kingai \
  -v kingai-logs:/var/log/kingai \
  kingai-os:dev
```

Check status:

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

## Hardened Docker Compose

From the repository root:

```bash
docker compose -f container/compose.yaml up -d --build
```

The supplied Compose profile keeps the root filesystem read-only, drops all Linux capabilities, enables `no-new-privileges`, limits process count, runs as `10001:10001`, and uses tmpfs only for `/run/kingai` and `/tmp`. It does not publish a TCP port.

## Multi-architecture Buildx

Build amd64 + arm64 as a local OCI archive:

```bash
bash scripts/build-container.sh
```

By default the archive is written to:

```text
dist/kingai-os-<VERSION>.oci.tar
```

Build one architecture and load it into the local Docker image store:

```bash
KINGAI_CONTAINER_PLATFORMS=linux/amd64 \
  bash scripts/build-container.sh
```

Choose an explicit OCI destination:

```bash
KINGAI_CONTAINER_OUTPUT=dist/kingai-os.oci.tar \
KINGAI_CONTAINER_PLATFORMS=linux/amd64,linux/arm64 \
  bash scripts/build-container.sh
```

Publish a multi-architecture image to a registry:

```bash
KINGAI_CONTAINER_IMAGE=ghcr.io/kingaiwork/kingai-os \
KINGAI_CONTAINER_TAG=dev \
KINGAI_CONTAINER_PUSH=1 \
  bash scripts/build-container.sh
```

Registry publishing enables BuildKit provenance and SBOM attestations. Multi-architecture local or registry builds require a Buildx builder capable of executing the Ubuntu runtime stage for both target architectures; on Linux this normally means binfmt/QEMU is installed for the non-native architecture.

## Container smoke test

```bash
make container-test
```

The smoke test verifies the image contract rather than only checking that the Docker build succeeded. It asserts:

- image user is `10001:10001`;
- no Docker `ExposedPorts` are present;
- PID 1 is `kingaid`;
- the runtime starts with a read-only root filesystem, zero Linux capabilities and `no-new-privileges`;
- no TCP listener appears in the container network namespace;
- Policy, Approval, Task, Memory, Model and Audit report enabled;
- Memory and Task data survive container deletion/recreation when the same persistent volumes are reused.

## Kubernetes / cloud container example

A secure single-replica example is provided at:

```text
container/kubernetes.yaml
```

Apply it after selecting the desired image tag and storage class for your cluster:

```bash
kubectl apply -f container/kubernetes.yaml
```

The manifest intentionally creates no Service and declares no container port. It uses `runAsNonRoot`, UID/GID 10001, `RuntimeDefault` seccomp, a read-only root filesystem, dropped capabilities, separate persistent claims for state and logs, and in-memory volumes for `/run/kingai` and `/tmp`.

For distributed or horizontally scaled deployments, do not point multiple independent `kingaid` replicas at the same local state volume. A future clustered state backend must define its own consistency and coordination model first.

## Persistence

Runtime-owned persistent paths are rooted at:

```text
/var/lib/kingai
/var/log/kingai
```

Current state includes:

```text
/var/lib/kingai/approvals
/var/lib/kingai/memory
/var/lib/kingai/tasks
/var/lib/kingai/models
/var/log/kingai
```

The Unix socket and sanitized runtime status under `/run/kingai` are ephemeral.

When using host bind mounts instead of Docker/Kubernetes managed volumes, ensure the host directories are writable by UID/GID `10001:10001` or provide an equivalent runtime ownership policy.

## Configuration

Default read-only configuration is shipped under `/etc/kingai`:

```text
/etc/kingai/policy.json
/etc/kingai/agents.json
/etc/kingai/models.json
/etc/kingai/system.json
```

Production deployments may mount reviewed configuration files over these paths. Secrets and provider credentials should be injected through the platform secret mechanism; they should not be baked into the image or committed to the repository.

## Security boundary

Container mode intentionally does not turn KINGAI into a network-exposed root service. The image does not declare an `EXPOSE` instruction, does not publish a management port in Compose or Kubernetes, and communicates with the local CLI through the Unix socket.

Mounting `/var/run/docker.sock`, the host root filesystem, privileged devices, or unrestricted host namespaces is not part of the default design. If host-level privileged operations are required, they must go through a separately reviewed constrained Execution Broker rather than bypassing Policy and Approval controls.

## CI and GHCR

`.github/workflows/container.yml` runs an amd64 runtime smoke test first and then verifies an amd64/arm64 Buildx build. Manual workflow dispatch can publish a developer tag to GHCR with SBOM and provenance enabled. Production-like tags such as `stable` and `latest` remain blocked from the developer publishing path.

## Release status

The Container edition is part of the D5 runtime development line. A successful CI build is not by itself a production release approval. Stable publishing still requires release governance, vulnerability review, signed provenance/image policy and promotion gates.

---

# 中文

KINGAI OS Container 是 KINGAI Runtime 面向 Docker / OCI / 云端容器 / CI / Homelab 的发行形态，与 Server、Desktop、IoT 共用同一套核心能力。

默认约束：

- `kingaid` 直接作为 PID 1，不依赖 systemd；
- UID/GID `10001:10001` 非 root 运行；
- 默认不开放、不声明、不映射管理 TCP；
- CLI 通过 `/run/kingai/kingaid.sock` 本地 Unix Socket 管理；
- Policy / Approval / Task / Memory / Model / Audit 保持启用；
- 数据统一持久化到 `/var/lib/kingai`；
- 审计与日志持久化到 `/var/log/kingai`；
- amd64 / arm64 使用 Buildx 构建；
- 默认不挂载 Docker Socket、不挂宿主机根目录、不启用 privileged。

推荐开发验证：

```bash
make container-test
```

多架构 OCI：

```bash
make container-multiarch
```

Docker Compose：

```bash
docker compose -f container/compose.yaml up -d --build
```

Kubernetes 示例：

```bash
kubectl apply -f container/kubernetes.yaml
```

如果未来需要操作宿主机高权限资源，应通过独立、受限并经过 Policy / Approval 审核的 Execution Broker，而不是直接给 KINGAI Container 无限宿主机权限。
