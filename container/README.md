# KINGAI OS Container

KINGAI OS Container is the Docker/OCI distribution form of the shared KINGAI Runtime. It is designed for cloud services, development, CI, homelab and service composition while preserving the same Policy, Approval, Task, Memory, Model and Audit core used by the bootable editions.

## Design

- no systemd requirement inside the image;
- `kingaid` is the container entrypoint;
- non-root `_kingai` runtime by default;
- no management TCP port exposed by default;
- Unix socket at `/run/kingai/kingaid.sock`;
- persistent state under `/var/lib/kingai`;
- persistent audit/log data under `/var/log/kingai`;
- amd64 and arm64 Buildx targets;
- same D5 runtime APIs as the Server/Desktop/IoT core.

## Build locally

```bash
docker build -f container/Dockerfile -t kingai-os:dev .
```

Run:

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

Create memory:

```bash
docker exec kingai-os \
  kingai memory put task '{"note":"hello from container"}'
```

Create a task:

```bash
docker exec kingai-os \
  kingai task create main verify container runtime
```

## Docker Compose

From the repository root:

```bash
docker compose -f container/compose.yaml up -d --build
```

The example Compose configuration:

- drops all Linux capabilities;
- enables `no-new-privileges`;
- uses a read-only root filesystem;
- uses tmpfs for `/run/kingai` and `/tmp`;
- persists KINGAI state and logs in named volumes.

## Multi-architecture build

```bash
KINGAI_CONTAINER_PLATFORMS=linux/amd64,linux/arm64 \
  bash scripts/build-container.sh
```

Export OCI archive:

```bash
KINGAI_CONTAINER_OUTPUT=dist/kingai-os.oci.tar \
KINGAI_CONTAINER_PLATFORMS=linux/amd64,linux/arm64 \
  bash scripts/build-container.sh
```

## Security boundary

Container mode intentionally does not turn KINGAI into a network-exposed root service.

The current image runs as `_kingai`, exposes no management TCP port and uses the same local Unix-socket model as the OS editions.

When privileged host operations are required in the future, they must go through a separately reviewed constrained Execution Broker. Mounting the Docker socket or host root filesystem into the KINGAI Container is not part of the default design because doing so would bypass the intended capability boundary.

## Persistence

Recommended persistent paths:

```text
/var/lib/kingai/approvals
/var/lib/kingai/memory
/var/lib/kingai/tasks
/var/log/kingai
```

The runtime socket and sanitized status under `/run/kingai` are ephemeral.

## Production note

The Container edition is currently part of the D5 Alpha Runtime development line. CI build/start verification is not the same as a signed production container release. Stable container publishing will require its own SBOM/provenance, vulnerability, signing and release-governance gates.

---

# 中文

KINGAI OS Container 是 KINGAI Runtime 的 Docker / OCI 发行形态，不是另一套独立产品。

它与 Server、Desktop、IoT 共用：

- Policy；
- Approval；
- Task Graph；
- Memory；
- Model Router；
- Audit；
- `kingai` CLI；
- `kingaid` 核心。

默认设计坚持：

- 非 root；
- 不开放管理 TCP；
- 不默认挂载 Docker Socket；
- 不默认挂载宿主机根目录；
- 状态和日志持久化；
- amd64/arm64 多架构构建。

如果未来需要操作宿主机高权限资源，必须通过单独设计和审核的受限 Execution Broker，而不是直接给容器无限制宿主机权限。
