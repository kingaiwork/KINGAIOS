# Building KINGAI OS

## English

KINGAI OS currently builds the D4 Developer Foundation core. Bootable production images are introduced only after the rootfs, boot trust, installer and rollback stages are validated.

### Core build

Requirements:

```text
Go 1.23+
Bash
GNU coreutils
```

Run:

```bash
make check
make build
```

Cross-build foundation binaries:

```bash
bash scripts/build-foundation.sh
```

Outputs are written to `out/foundation/` and are ignored by Git.

### Image profiles

Official image composition is defined in:

```text
profiles/server.yaml
profiles/desktop.yaml
profiles/iot.yaml
```

These profiles define architecture, KINGAI features, image format and size budgets.

### Ubuntu 26.04 development bootstrap

The 26.04 LTS generation is Resolute Raccoon. During the Developer Foundation stage, rootfs experiments may use Ubuntu's Resolute package/archive infrastructure. Public Stable images will add KINGAI package provenance, branding, license inventory, signing and release gates before being considered releasable.

### Image build stages

```text
1. Source manifest resolution
2. Base/rootfs construction
3. KINGAI core installation
4. Profile package composition
5. Security policy installation
6. KINGAI branding
7. Boot/installer composition
8. Image optimization
9. SBOM and provenance
10. Signature/checksum
11. Install/boot test
12. Upgrade/rollback/recovery test
13. Release routing
```

An image that has not passed the relevant gates must not be labeled Stable.

### Artifact routing

Use:

```bash
bash scripts/publish-artifact.sh <artifact>
```

The script computes SHA256 and routes files at or above 2 GiB to Cloudflare R2 when credentials are supplied through the environment.

See `docs/R2.md`.

---

## 中文

KINGAI OS 当前首先构建 D4 Developer Foundation 核心。只有 RootFS、Boot Trust、安装器、升级和回滚流程经过验证后，才进入正式可启动发行镜像阶段。

### Core 构建

```bash
make check
make build
bash scripts/build-foundation.sh
```

输出保存在 `out/foundation/`，不会提交到 Git。

### 三个镜像 Profile

```text
profiles/server.yaml
profiles/desktop.yaml
profiles/iot.yaml
```

分别控制服务器、桌面和 IoT/Edge 的架构、功能、格式和体积预算。

### 镜像流水线

正式镜像按照：源码解析 → RootFS → KINGAI Core → Profile → 安全策略 → 品牌 → Boot/Installer → 精简 → SBOM/Provenance → 签名 → 安装/启动测试 → 升级/回滚测试 → 发布分流。

没有通过相应门禁的镜像不得标记为 Stable。

### 大文件发布

```bash
bash scripts/publish-artifact.sh <artifact>
```

脚本会计算 SHA256；达到 2 GiB 的大文件在 R2 环境变量配置完成后自动走 Cloudflare R2。详见 `docs/R2.md`。
