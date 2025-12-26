# 单元测试评审回放（fix-plan）记录

日期：2025-12-26  
范围：对照 `.claude/specs/unit-tests/fix-plan.md`，抽查关键修复点，确认“注入点/全局副作用/断言质量/跨平台 skip 策略”符合预期。

## 评审摘要

- 结论：关键问题已通过“可注入 + 可恢复”的方式修复；本次未发现需要新增代码修复的缺口。
- 运行证据：
  - 统一入口：`./test.sh`（生成 `coverage.out`，`-covermode=count`）
  - 覆盖率解析：`go tool cover -func=coverage.out`

## 重点抽查项

### 1) cmd/version_test.go：避免全局 HTTP 副作用（P0）

- 关注点：避免替换 `http.DefaultTransport` / `http.DefaultClient` 等全局对象，防止跨测试污染与竞态。
- 当前实现：
  - `cmd/version_test.go` 使用 `checkForUpdateFunc` 注入 stub，测试只验证命令输出，不触碰全局 HTTP transport。
- 参考：
  - `cmd/version_test.go:24`

### 2) internal/testutil：stdout/stderr 捕获与恢复（P1）

- 关注点：`os.Stdout/os.Stderr` 属于进程级全局对象，必须确保在失败/异常路径也能恢复。
- 当前实现：
  - `CaptureOutput` 使用 `defer` 恢复 `os.Stdout/os.Stderr`，并使用 `sync.Once` 确保 pipe 只关闭一次。
- 参考：
  - `internal/testutil/testutil.go:120`
  - `internal/testutil/testutil_test.go:139`

### 3) internal/testutil：临时切换 CWD 的副作用边界（P1）

- 关注点：`os.Chdir` 是进程级全局副作用，若与 `t.Parallel()` 混用会导致 flake。
- 当前实现：
  - `WithTempCWD` 在注释中明确“不得与并行测试一起使用”，并通过 `t.Cleanup` 恢复原工作目录。
- 参考：
  - `internal/testutil/testutil.go:96`
  - `internal/testutil/testutil_test.go:114`

### 4) internal/tui sweep：断言质量（P1）

- 关注点：sweep 类测试不能只验证“不崩溃”，需要接回 `Update` 返回的 model 并断言关键状态（mode / err / cursor 等）。
- 当前实现：
  - `internal/tui/coverage_sweep_test.go` 用例均提供 `verify` 回调，对关键字段做 `t.Fatalf` 断言。
- 参考：
  - `internal/tui/coverage_sweep_test.go:21`

### 5) 跨平台权限与 root：skip 策略

- 关注点：Windows 权限模型不同；root 环境下权限失败场景可能不稳定，需要显式 skip。
- 当前实现：
  - `cmd/isRoot()` 通过 build tag 区分 Windows/Unix。
  - 多处测试对 Windows 或 root 环境进行 `t.Skip`（例如备份、settings 权限用例等）。
- 参考：
  - `cmd/os_unix_test.go:1`
  - `cmd/os_windows_test.go:1`
  - `cmd/backup_test.go:90`
  - `internal/settings/settings_test.go:153`

