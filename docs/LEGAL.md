# KINGAI OS Legal & Compliance Baseline

## English

This document defines engineering controls for licensing, trademarks, model redistribution, privacy, security response and global distribution. It is not a substitute for jurisdiction-specific legal advice.

### 1. Distribution identity

KINGAI OS is distributed under KINGAI branding. Upstream software remains under its respective copyright and license terms.

The project must not present a modified Ubuntu image as an official Ubuntu product. Any public KINGAI OS image derived from Ubuntu source lineage must follow applicable Canonical trademark/IP policy and the licenses of redistributed components.

### 2. Source traceability

For every redistributed package, the build system should retain or reference:

- package name and version;
- upstream source location;
- license identifier(s);
- copyright notices;
- KINGAI patches;
- binary hash;
- source hash;
- build recipe;
- redistribution status.

### 3. Copyleft obligations

Where a license requires corresponding source or other distribution obligations, the release process must publish or provide the required material for the exact distributed binary, including KINGAI modifications where applicable.

### 4. KINGAI-owned code

The project may use Apache-2.0 for selected KINGAI-owned open components, but each directory/package must carry an explicit licensing decision before a public Stable release. Enterprise/cloud services may use separate commercial terms.

### 5. Models and datasets

Model weights, datasets, voices, fonts, icons and other content are not assumed redistributable merely because they are publicly downloadable.

A model/content manifest should record:

```text
publisher
name
version
license
commercial_use
redistribution
derivatives
region
source
hash
```

If redistribution rights are unclear, the asset must not be embedded in an official ISO. It should be downloaded by the user from its authorized source after installation.

### 6. Trademarks and visual identity

KINGAI OS uses KINGAI-owned names and original visual assets. Desktop modes are named KINGAI Intelligence, KINGAI Flow and KINGAI Classic.

Do not use third-party company names as official KINGAI editions, and do not copy proprietary icons, wallpapers, logos or protected UI assets.

### 7. Privacy by design

The OS should provide explicit controls for:

- local-only operation;
- cloud synchronization;
- telemetry;
- memory retention;
- user data export;
- user data deletion;
- model/provider disclosure where relevant;
- organization policy.

Sensitive local memory should not be sent to a cloud model unless policy and user/organization authorization permit it.

### 8. AI transparency and auditability

Where required by product design or law, KINGAI should be able to record or expose:

```text
agent_id
model_id
model_version
provider
timestamp
policy_version
tool_calls
human_approval
generated_by_ai
```

### 9. Security response

Before public Stable/GA release, KINGAI should operate a PSIRT-style process with:

- private vulnerability intake;
- affected-version tracking;
- coordinated disclosure;
- patch and advisory workflow;
- security contact;
- incident timeline recording;
- SBOM/VEX capability where appropriate.

### 10. Export-control review

Because a modern operating system contains cryptographic and security functionality, formal worldwide commercial release should include an export-control classification/review appropriate to the actual KINGAI product and distribution method.

### 11. Release legal gate

A Stable release must not be signed if any mandatory legal metadata is missing.

Minimum release gate:

```text
[ ] package/license inventory complete
[ ] third-party notices generated
[ ] corresponding-source obligations resolved
[ ] model/content redistribution checked
[ ] trademark/branding review complete
[ ] SBOM generated
[ ] security contact operational
[ ] privacy disclosures match actual behavior
[ ] release artifacts and source are traceable
```

---

## 中文

本文定义 KINGAI OS 在许可证、商标、模型再发行、隐私、安全响应和全球发行方面必须落到工程流程中的合规基线，不替代具体国家/地区律师的正式法律意见。

### 1. 发行身份

KINGAI OS 使用 KINGAI 自有品牌发行。所有上游软件继续遵守各自版权和许可证。公开发行时不得把修改后的 Ubuntu 描述为官方 Ubuntu 产品；使用 Ubuntu 技术血缘的 KINGAI OS 必须遵守 Canonical 适用的商标/IP政策及所有被再发行组件的许可证。

### 2. 源码可追溯

每个被再发行的软件包必须能够追踪包名、版本、上游源码、许可证、版权、KINGAI 补丁、二进制哈希、源码哈希、构建方式和再发行状态。

### 3. Copyleft义务

如果许可证要求提供 Corresponding Source 或其他再发行义务，必须针对实际发布的二进制和 KINGAI 修改满足要求，而不是只保存一个无关的上游源码链接。

### 4. KINGAI自有代码

KINGAI 自有开放组件可优先考虑 Apache-2.0，但在 Stable 前每个组件必须有明确许可证。企业/云端服务可以另外使用商业条款。

### 5. 模型与内容

模型、数据集、语音、字体、图标等即使可以公开下载，也不能默认认为允许我们重新打包进 ISO。权限不清晰的资产不得进入官方镜像，应由用户安装后从授权来源自行获取。

### 6. 商标与视觉

官方桌面名称使用 KINGAI Intelligence、KINGAI Flow、KINGAI Classic。不得把第三方公司名称作为 KINGAI 官方 Edition，也不得复制第三方专有 Logo、图标、壁纸和受保护 UI 资产。

### 7. Privacy by Design

系统必须提供 Local-only、云同步、Telemetry、Memory保存期限、数据导出、数据删除、模型/Provider披露和组织策略控制。

### 8. AI透明度与审计

在产品或法律需要时，系统应能够追踪 Agent、模型、版本、Provider、时间、策略版本、工具调用、人工批准和AI生成标记。

### 9. 安全响应

公开 Stable/GA 之前必须建立私密漏洞入口、影响版本追踪、协调披露、补丁/公告、安全联系人、事件时间线和必要的 SBOM/VEX 流程。

### 10. 出口管制

现代 OS 包含密码学和安全功能。正式全球商业发行前，需要根据 KINGAI OS 的真实功能和发行方式进行适当的美国出口管制分类/审查。

### 11. Stable法律发布门禁

许可证清单、第三方 NOTICE、对应源码、模型再发行、商标审查、SBOM、安全联系人、隐私披露和发布可追溯性任一关键项缺失，都不得签发 Stable。
