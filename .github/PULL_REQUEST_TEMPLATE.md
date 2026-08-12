## English

### What changed
Describe the change and the subsystem it affects.

### Why
Explain the user, security, reliability or maintainability reason.

### Risk
- [ ] No privileged execution path changed
- [ ] No release/signing path changed
- [ ] No privacy-sensitive data handling changed
- [ ] No third-party license/model redistribution changed

If any box above cannot be checked, describe the additional review required.

### Validation
- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] relevant image/rootfs/desktop/IoT smoke test
- [ ] rollback or failure behavior considered

---

## 中文

### 修改内容
说明本次修改以及受影响的子系统。

### 修改原因
说明用户价值、安全、可靠性或可维护性原因。

### 风险
- [ ] 未修改特权执行路径
- [ ] 未修改发行/签名路径
- [ ] 未修改隐私敏感数据处理
- [ ] 未改变第三方许可证/模型再发行条件

如有任一项无法勾选，请说明需要的额外审核。

### 验证
- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] 已运行对应镜像/RootFS/Desktop/IoT 冒烟测试
- [ ] 已考虑失败与回滚行为
