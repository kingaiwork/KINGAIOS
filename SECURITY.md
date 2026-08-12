# Security Policy

## English

KINGAI OS treats security as an operating-system property, not an optional feature.

### Supported development state

The repository is currently pre-alpha. No current branch should be treated as production-safe unless a release is explicitly marked Stable.

### Reporting vulnerabilities

Please do **not** open a public issue for an unpatched vulnerability that could put users at risk.

Preferred private contact for the public release program:

- `security@kingai.work` (planned security contact; must be operational before public Stable release)

Until the security mailbox and PSIRT workflow are operational, use GitHub private security advisories when available.

### Security architecture

KINGAI OS is designed around:

- capability-based agent authorization;
- least privilege and default deny;
- separation between AI intent and privileged execution;
- systemd/cgroup v2 isolation;
- AppArmor, seccomp and Landlock where supported;
- rootless containers for third-party workloads;
- stronger VM/MicroVM isolation for untrusted workloads;
- Secure Boot/TPM/integrity work for production images;
- signed update metadata;
- rollback-safe updates;
- SBOM and build provenance;
- security logging and auditability.

### Agent trust boundary

An agent must not be able to grant itself new privileged capabilities. Critical trust-root actions must be controlled outside the model decision path.

### Secrets

Never commit:

- model/API keys;
- Cloudflare R2 credentials;
- Secure Boot private keys;
- TUF root/private signing keys;
- release-signing private keys;
- SSH private keys;
- production tokens.

Signing roots should remain offline or in appropriately hardened key-management systems.

### Release security gates

A Stable image is expected to pass:

1. dependency and license inventory;
2. secret scanning;
3. static analysis where applicable;
4. package vulnerability checks;
5. image integrity checks;
6. SBOM generation;
7. build provenance generation;
8. clean-install test;
9. upgrade test;
10. rollback/recovery test;
11. signature verification test.

### Responsible disclosure

We intend to maintain a coordinated disclosure and PSIRT process before general availability. Security fixes should include affected versions, remediation status and release guidance.

---

## 中文

KINGAI OS 把安全视为操作系统的基础属性，而不是可选插件。

### 当前状态

仓库目前处于 Pre-Alpha。除非某个版本明确标记为 Stable，否则不能视为生产环境安全版本。

### 漏洞报告

对于尚未修复、可能影响用户安全的问题，请不要直接创建公开 Issue。

正式公开发行前计划启用：

- `security@kingai.work`

在安全邮箱和 PSIRT 流程完全启用前，优先使用 GitHub Private Security Advisory 等私密渠道。

### 安全架构

KINGAI OS 将采用 Capability 权限、默认拒绝、AI意图与高权限执行分离、systemd/cgroup v2、AppArmor、seccomp、Landlock、Rootless Container、高风险隔离环境、Secure Boot/TPM、签名更新、可回滚升级、SBOM、Provenance 和安全审计。

### Agent 信任边界

智能体不得自行给自己增加更高权限。信任根、启动密钥、发布根密钥等关键权限必须处于模型决策路径之外。

### 禁止提交的秘密

严禁把 API Key、R2 密钥、Secure Boot 私钥、TUF Root 私钥、Release Signing 私钥、SSH 私钥和生产 Token 写入 Git 仓库。

### Stable 发布安全门禁

正式 Stable 镜像必须通过依赖/许可证、Secret Scan、静态检查、漏洞检查、镜像完整性、SBOM、Provenance、全新安装、升级、回滚/恢复、签名验证等发布测试。
