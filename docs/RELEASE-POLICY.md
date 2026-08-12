# KINGAI OS Release Policy

## English

KINGAI OS uses four public engineering channels:

`dev → beta → rc → stable`

### Dev
May expose incomplete functionality. Developer Preview images may be non-installable and may not use production Secure Boot keys.

### Beta
Requires repeatable image builds, core CI, image-specific smoke tests, dependency/license inventory and documented known limitations.

### RC
Requires install-to-disk validation, production-grade signed update metadata, Secure Boot release chain, A/B update/rollback tests, recovery validation and no open release-blocking security defects.

### Stable
Requires all RC gates plus reproducible release provenance, SBOM, signatures/checksums, support lifecycle, release notes, legal/compliance review and protected release authority.

A Stable release is never created merely because a version number is changed.

---

# KINGAI OS 发布策略

## 中文

KINGAI OS 使用四级公开工程渠道：

`dev → beta → rc → stable`

### Dev
允许功能尚未完整。Developer Preview 可以暂时不可安装，也不使用生产 Secure Boot 私钥。

### Beta
必须具备可重复镜像构建、核心 CI、对应镜像冒烟测试、依赖/许可证清单和明确已知限制。

### RC
必须通过写盘安装、生产级签名更新元数据、Secure Boot 发行链、A/B 更新与回滚、Recovery 验证，并且不存在阻断发行的安全缺陷。

### Stable
必须满足全部 RC 门禁，并具备可追溯构建 provenance、SBOM、签名/校验、支持生命周期、Release Notes、法律/合规审核和受保护的发行权限。

Stable 绝不会因为“修改版本号”而自动产生。
