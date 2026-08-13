# KINGAI OS Repository Synchronization Contract

## English

KINGAI OS intentionally separates the public operating-system repository from the private website source repository.

- Public OS source: `kingaiwork/KINGAIOS`
- Private website source: `kingaiwork/Kingaiosweb`
- Website: https://os.kingai.work

The repositories are **not mirrors**. Synchronization is allowlist-only.

### Public OS → private website

`.github/workflows/sync-to-web.yml` synchronizes selected public facts into `Kingaiosweb/sync/os/` after normal `main` updates and release events.

Approved sources include, when present:

- `VERSION`
- `README.md`
- `SECURITY.md`
- `llms.txt`
- `docs/STATUS.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`
- `docs/DESKTOP.md`
- `docs/ROADMAP.md`
- `docs/HEALTH-INTELLIGENCE.md`
- `seo/GEO-KNOWLEDGE-GRAPH.md`
- `release/gates.json`

Required repository secret in `KINGAIOS`:

- `KINGAI_WEB_SYNC_TOKEN`: fine-grained token with Contents read/write access to `kingaiwork/Kingaiosweb` only.

### Private website → public OS

The private repository synchronizes **only** its `public-export/` directory into `KINGAIOS/web/public/`. It also creates a `source.json` recording the private website source commit without exposing private source content.

Required repository secret in `Kingaiosweb`:

- `KINGAI_OS_SYNC_TOKEN`: fine-grained token with Contents read/write access to `kingaiwork/KINGAIOS` only.

### Loop prevention

- Commits touching only `web/public/**` do not trigger OS → web synchronization.
- Commits touching only `sync/os/**` do not trigger web → OS synchronization.

### Privacy boundary

No private website source, Cloudflare credential, analytics secret, customer data, unpublished roadmap, internal configuration or legal draft may be placed in `public-export/`.

Every synchronization commit includes source provenance through `source.json` and a `[sync-os]` or `[sync-web]` commit prefix.

---

## 中文

KINGAI OS 将公开操作系统源码仓与私有官网源码仓严格分离：

- 公开系统仓：`kingaiwork/KINGAIOS`
- 私有官网仓：`kingaiwork/Kingaiosweb`
- 官网：https://os.kingai.work

两个仓库不是镜像关系，而是**严格白名单双向同步**。

公开 OS 仓只把版本、状态、公开文档、安全资料、llms/GEO、发行门禁等同步到私有网站仓的 `sync/os/`。

私有网站仓只允许 `public-export/` 中明确批准公开的内容同步回公开 OS 仓的 `web/public/`。

任何私有网页源码、Cloudflare 凭据、内部配置、客户数据、未发布计划或法律草稿都不得进入 `public-export/`。

需要一次性配置两个最小权限 Fine-grained GitHub Token：

- `KINGAIOS` 中配置 `KINGAI_WEB_SYNC_TOKEN`，只允许写 `Kingaiosweb`。
- `Kingaiosweb` 中配置 `KINGAI_OS_SYNC_TOKEN`，只允许写 `KINGAIOS`。

同步提交使用 `[sync-os]` / `[sync-web]` 前缀，并通过独立管理目录避免双向死循环。
