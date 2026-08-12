# KINGAI OS Recovery Architecture

## English

KINGAI OS Recovery is a protected repair environment, not a general-purpose root shell.

### Recovery responsibilities

A future production recovery image may:

- inspect A/B slot health and boot metadata;
- verify KINGAI update manifests, hashes and release signatures;
- select a previously verified system slot for rollback;
- repair boot metadata from trusted templates;
- export user-requested diagnostic bundles after sanitization;
- unlock encrypted STATE only with explicit authorized recovery credentials;
- reinstall a verified KINGAI system image without silently preserving untrusted system binaries.

### Recovery must not

- accept unsigned system images as trusted;
- disable Secure Boot, verified-image policy or update signature checks automatically;
- expose production signing keys;
- upload private files, prompts, memory content or credentials without explicit consent;
- execute arbitrary downloaded scripts as root;
- modify both A and B system slots in a single unverified transaction.

### Production gates

Recovery is not considered production-ready until automated VM tests verify:

1. valid rollback from B to A;
2. refusal of a tampered slot/image;
3. recovery after interrupted update staging;
4. bootloader restoration;
5. encrypted STATE behavior;
6. preservation of user data when requested and safe;
7. destructive reinstall only after explicit confirmation;
8. post-recovery boot health confirmation.

The Developer Foundation intentionally contains no automated destructive recovery command.

---

# KINGAI OS Recovery 恢复架构

## 中文

KINGAI OS Recovery 是受保护的系统修复环境，不是一个普通的无限制 root shell。

### Recovery 允许承担的职责

未来生产恢复环境可以：

- 检查 A/B Slot 健康与启动元数据；
- 验证 KINGAI 更新 manifest、哈希与发行签名；
- 回滚到之前已验证的系统 Slot；
- 使用可信模板修复启动元数据；
- 在去敏感化后导出用户主动请求的诊断信息；
- 仅在明确授权的恢复凭据下解锁加密 STATE；
- 使用已验证 KINGAI 镜像重新安装，而不是静默保留未知系统二进制。

### Recovery 禁止

- 把未签名镜像当作可信系统；
- 自动关闭 Secure Boot、镜像验证或更新签名验证；
- 暴露生产签名私钥；
- 未经明确同意上传私人文件、Prompt、记忆正文或凭据；
- 以 root 执行任意下载脚本；
- 在一个未经验证的事务中同时修改 A、B 两个系统 Slot。

### 生产门禁

Recovery 在以下 VM 自动化测试全部通过前，不得标记为生产可用：

1. B→A 正常回滚；
2. 篡改镜像必须拒绝；
3. 更新中断后的恢复；
4. Bootloader 恢复；
5. 加密 STATE 行为；
6. 用户数据在安全条件下可保留；
7. 破坏性重装必须明确确认；
8. 恢复后启动健康确认。

Developer Foundation 当前不提供自动化破坏性 Recovery 命令。
