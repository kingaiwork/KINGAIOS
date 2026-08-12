# Contributing to KINGAI OS

## English

KINGAI OS welcomes engineering contributions that strengthen reliability, security, portability, AI/agent governance, local-first intelligence and long-term maintainability.

### Engineering rules

1. Keep privileged authority behind the Policy + Capability boundary.
2. Never commit credentials, signing keys, production secrets, private prompts or user data.
3. New network listeners require explicit threat-model review; local IPC is preferred for local capabilities.
4. Third-party models, firmware, fonts, themes, icons and software must have traceable licensing before inclusion in an official image.
5. A change to boot, installer, update, signing, sandbox or authorization code requires stronger tests than ordinary UI/documentation changes.
6. Stable releases must never be produced from an unverified local artifact.

### Before opening a PR

Run:

```bash
go test ./...
go vet ./...
bash -n scripts/*.sh desktop/welcome/*.sh
```

Then run the relevant GitHub image smoke workflow for changes that affect Server, Desktop or IoT/Edge images.

---

# 参与 KINGAI OS 开发

## 中文

KINGAI OS 欢迎能够提升可靠性、安全性、跨平台能力、AI/智能体治理、本地优先智能和长期可维护性的工程贡献。

### 工程规则

1. 特权必须始终位于 Policy + Capability 边界之后。
2. 禁止提交凭据、签名私钥、生产 Secret、私人 Prompt 或用户数据。
3. 新增网络监听必须进行威胁模型审核；本机能力优先使用本地 IPC。
4. 第三方模型、固件、字体、主题、图标和软件进入官方镜像前必须有可追溯许可证。
5. Boot、Installer、Update、Signing、Sandbox、Authorization 的修改必须有比普通 UI/文档更强的测试。
6. Stable 版本不得由未经验证的本地文件直接发布。
