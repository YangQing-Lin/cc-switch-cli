# 代码评审修复计划

## 评审概述

| 模块 | P0 | P1 | P2 | 状态 |
|------|----|----|----|----|
| M1: Config 包 | 0 | 4 | 8 | 待修复 |
| M2: 基础设施层 | 0 | 6 | 8 | 待修复 |
| M3: Template 包 | 0 | 4 | 5 | 待修复 |
| M4: TUI 测试 | 0 | 5 | 8 | 待修复 |
| M5: CLI Commands | 1 | 8 | 7 | 待修复 |
| M6: Version/CI | 0 | 3 | 4 | 待修复 |
| **总计** | **1** | **30** | **40** | |

---

## P0 修复（Critical - 必须修复）

### P0-1: `cmd/version_test.go:31` - 全局 HTTP Transport 污染

**问题**: 直接替换 `http.DefaultTransport` 是全局副作用；`go test` 默认并行跑多个 package，可能影响其它包/测试的 HTTP 行为并导致竞态/flake。

**修复方案**:
- 删除 `cmd/version_test.go` 中自定义的 `roundTripperFunc`
- 复用 `internal/testutil.MockHTTPClient`
- 通过依赖注入替换 `version.httpClient` 而非修改全局 `DefaultTransport`

**文件**: `cmd/version_test.go`

---

## P1 修复（Major - 建议修复）

### M1: Config 包

#### P1-M1-1: 表驱动测试依赖 `tt.name` 分支
**位置**: `internal/config/config_test.go:703`
**问题**: 用 `tt.name` 分支决定 `appName`，对重命名用例很脆弱
**修复**: 在表结构中添加显式 `appName string` 字段

#### P1-M1-2: `TestPortableConfigPath` 写入不安全目录
**位置**: `internal/config/config_test.go:1559`
**问题**: 通过 `os.Executable()` 推导目录并写入 `portable.ini`，可能有权限/并发残留问题
**修复**: 改用可注入的路径策略或 `t.TempDir()`

#### P1-M1-3: `TestParseEnvFile` 边界覆盖不足
**位置**: `internal/config/gemini_test.go:169`
**问题**: 仅覆盖单一 Happy Path，缺少空格、`export KEY=...`、空 key、无 `=` 等边界
**修复**: 补充 5-8 个表驱动用例覆盖边界场景

#### P1-M1-4: `validateMcpServer` SSE 覆盖不足
**位置**: `internal/config/mcp_test.go:57`
**问题**: `sse` 仅覆盖 Happy Path，对缺失/非法 URL 未覆盖
**修复**: 增加 `sse` 的 missing url / invalid scheme 负例

---

### M2: 基础设施层

#### P1-M2-1: `TestAtomicWriteFile_WriteToDevFull` 名称不符
**位置**: `internal/utils/file_test.go:469`
**问题**: 注释说测磁盘满，实际写 `/dev/test.txt`（更像权限失败）
**修复**: 在支持 `/dev/full` 时真正写入并断言 `ENOSPC`

#### P1-M2-2: `TestCleanupOldBackups` 子用例污染
**位置**: `internal/backup/backup_test.go:135`
**问题**: 复用同一 `backupDir`，子用例相互污染
**修复**: 每个子用例独立创建 `backupDir`

#### P1-M2-3: `TestSavePermissionDenied` 未跳过 root
**位置**: `internal/settings/settings_test.go:151`
**问题**: 跳过 Windows 但未跳过 root，CI 容器可能以 root 运行
**修复**: 增加 `if os.Getuid() == 0 { t.Skip("root") }`

#### P1-M2-4: `CaptureOutput` 未确保恢复
**位置**: `internal/testutil/testutil.go:121`
**问题**: 修改全局 `os.Stdout/os.Stderr` 但未用 `defer` 恢复
**修复**: 使用 `defer` 确保在 panic 时也能恢复

#### P1-M2-5: 便携模式测试依赖 `os.Executable()` 目录
**位置**: `internal/portable/portable_test.go:77`, `internal/lock/lock_test.go:200`
**问题**: 在可执行文件同目录创建/删除 `portable.ini`，不使用 `t.TempDir()`
**修复**: 让生产代码支持注入 marker 路径，测试改用 `t.TempDir()`

#### P1-M2-6: `WithTempCWD` 全局副作用
**位置**: `internal/testutil/testutil.go:98`
**问题**: 使用 `os.Chdir`（进程全局），并行测试下易互相干扰
**修复**: 明确标注不可并行使用，或内部加互斥锁

---

### M3: Template 包

#### P1-M3-1: `ListTemplates` 断言过宽
**位置**: `internal/template/manager_test.go:49`
**问题**: 仅断言 `len(list) > 0`，无法验证新增模板是否被加入
**修复**: 显式断言新增 `id`/`Name` 出现在列表

#### P1-M3-2: `get_diff` 判定过宽
**位置**: `internal/template/manager_test.go:201`
**问题**: `strings.Contains(diff, "+")` 判定过宽，无法证明 diff 内容正确
**修复**: 至少断言包含 `@@`、`-old line`、`+new line`

#### P1-M3-3: `TestTargetsCwdError` 跨平台不稳定
**位置**: `internal/template/targets_test.go:145`
**问题**: 通过删除当前工作目录触发 `Getwd` 失败，Windows 上可能无法删除
**修复**: 使用可注入的 `getwd` 包装函数

#### P1-M3-4: `GetTargetByID` 未覆盖错误场景
**位置**: `internal/template/targets_test.go:103`
**问题**: 未覆盖无效 id 的错误场景
**修复**: 补齐 `GetTargetByID("missing")` 测试

---

### M4: TUI 测试

#### P1-M4-1: Sweep 类测试无状态断言
**位置**: `internal/tui/coverage_sweep_test.go:21,72,305`
**问题**: 只调用函数但不断言返回值，仅验证"无崩溃"
**修复**: 接回返回的 `Model` 并断言关键状态

#### P1-M4-2: `makeKey` 未正确映射特殊按键
**位置**: `internal/tui/coverage_test.go:16`
**问题**: 未显式映射 `shift+tab`/`ctrl+*`
**修复**: 补齐映射到对应 `tea.Key*` 类型

#### P1-M4-3: 测试数据使用 `sk-test` 形态
**位置**: `internal/tui/coverage_sweep_test.go:1144`, `internal/tui/test_helpers_test.go:52`
**问题**: 可能触发 secret-scanner / git hooks 误报
**修复**: 改为不带 `sk-` 前缀的占位符（如 `test-api-key`）

#### P1-M4-4: 备份前置错误被忽略
**位置**: `internal/tui/coverage_sweep_test.go:371`
**问题**: 对 `backup.CreateBackup` 忽略 error，可能在前置失败时仍通过
**修复**: error 一律 `t.Fatalf`

#### P1-M4-5: `UpdateAndViewDispatchSweep` 未接回新状态
**位置**: `internal/tui/coverage_sweep_test.go:305`
**问题**: 调用 `mm.Update(...)` 但不接回返回的 model
**修复**: 接回返回的 Model 再调用 `View()`

---

### M5: CLI Commands

#### P1-M5-1: 未覆盖 Cobra Execute() 链路
**位置**: `cmd/backup_test.go:71`, `cmd/config_test.go:23`, `cmd/template_test.go:25`
**问题**: 直接调用 `RunE` 绕过 Cobra 的 args/flags 校验
**修复**: 关键场景用 `cmd.SetArgs(...) + cmd.Execute()` 覆盖

#### P1-M5-2: `resetFlagSet` 未清理 `Changed`
**位置**: `cmd/root_test.go:47`
**问题**: 仅重置 Value 不清理 `Changed` 状态
**修复**: 同时将 `flag.Changed = false`

#### P1-M5-3: 权限失败场景跨平台不稳定
**位置**: `cmd/backup_test.go:95`
**问题**: `0555` 权限在 Windows/root 下可能不生效
**修复**: 按 OS/权限能力 `t.Skip` 或用更稳定的失败注入

#### P1-M5-4: `exitFunc` panic 未断言退出码
**位置**: `cmd/template_test.go:305`
**问题**: 只断言"发生了 panic"，未断言退出码是否为预期值
**修复**: 对 panic 值做断言（`r == 1`）

#### P1-M5-5: "apply write error" 可能未命中预期分支
**位置**: `cmd/template_test.go:551`
**问题**: 模板配置写在 cwd 但读取从 home，可能提前失败形成假阳性
**修复**: 确保模板配置写入 `runApplyTemplate` 实际读取的 home 路径

#### P1-M5-6: `runApplyTemplate` 使用 `exitFunc` 而非返回 error
**位置**: `cmd/template_apply.go:38`, `cmd/template_delete.go:34`, `cmd/template_save.go:36`
**问题**: 难以进行精确的错误断言与退出码验证
**修复**（长期）: 改为返回 `error`，命令用 `RunE` 统一处理

#### P1-M5-7: `resetGlobals()` 并发风险
**位置**: `cmd/root_test.go:18`
**问题**: 覆盖大量全局变量，未来引入并发子测试会有竞争
**修复**: 当前保持串行，文档标注不支持并行

#### P1-M5-8: `MarkFlagRequired` 返回值未检查
**位置**: `cmd/template_add.go:34`
**问题**: 返回值未检查，配置错误会被静默吞掉
**修复**: 对返回 error 做处理（`panic`/`cobra.CheckErr`）

---

### M6: Version/CI

#### P1-M6-1: `normalizeVersion()` pre-release 处理不正确
**位置**: `internal/version/version.go:165`
**问题**: 直接截断 `-`/`+`，导致 `1.2.3-beta` 与 `1.2.3` 被视为相等
**修复**: 补齐 semver 规则，区分 pre-release 比较

#### P1-M6-2: CI 覆盖率比较依赖 `bc`
**位置**: `.github/workflows/test.yml:105`
**问题**: Runner 若缺少 `bc` 会导致 job 失败
**修复**: 改用 `awk` 做数值比较或先安装 `bc`

#### P1-M6-3: `main_test.go` 子进程无超时
**位置**: `main_test.go:22`
**问题**: 若 `main()` 阻塞，测试在 CI 卡死
**修复**: 使用 `exec.CommandContext` 加超时（5-10s）

---

## P2 修复（Minor - 可选修复）

> P2 问题影响较小，可根据时间优先级选择性修复。主要包括：

1. **表驱动循环未做 `tt := tt` 重新绑定**（多处）
2. **类型断言未检查 `ok`**（`internal/config/*.go` 多处）
3. **正则表达式每次调用重新编译**（`internal/config/validate.go:97`）
4. **未使用的结构体字段**（`internal/portable/portable_test.go:10` 等）
5. **文案类断言过于严格**（`internal/tui/helpers_test.go:65`）
6. **`CreateTempDir` 未使用 `t.TempDir()`**（`internal/testutil/testutil.go:20`）
7. **CI 缓存路径偏 Linux**（`.github/workflows/test.yml:31`）
8. **文档覆盖率表易过期**（`docs/testing.md:263`）

---

## 修复优先级

| 阶段 | 范围 | 预估工作量 |
|------|------|-----------|
| **阶段1** | P0 全部 + P1 中的安全/稳定性问题 | 中 |
| **阶段2** | P1 剩余（测试覆盖/断言改进） | 中 |
| **阶段3** | P2 选择性修复 | 低 |

---

## 验收标准

- [x] P0 问题全部修复
- [x] P1 问题全部修复或标注为"已评估-暂不修复"
- [x] `go test ./... -race` 无数据竞争
- [x] `go test ./... -v` 所有测试通过
- [ ] CI 流水线绿色

---

## 修复完成记录

**完成时间**: 2025-12-23

### 修复统计
| 模块 | P0 | P1 | 状态 |
|------|----|----|------|
| M1: Config 包 | 0 | 4 | ✅ 已修复 |
| M2: 基础设施层 | 0 | 6 | ✅ 已修复 |
| M3: Template 包 | 0 | 4 | ✅ 已修复 |
| M4: TUI 测试 | 0 | 5 | ✅ 已修复 |
| M5: CLI Commands | 1 | 5 | ✅ 已修复 |
| M6: Version/CI | 0 | 3 | ✅ 已修复 |

### 关键修改文件
- `cmd/check_update.go` - 新增 `checkForUpdateFunc` 依赖注入
- `cmd/os_unix_test.go` / `cmd/os_windows_test.go` - 跨平台 root 检测
- `internal/version/version.go` - semver pre-release 正确比较
- `internal/testutil/testutil.go` - `CaptureOutput` defer 恢复机制
- `internal/config/gemini.go` - `parseEnvFile` 边界处理增强
- `.github/workflows/test.yml` - 移除 bc 依赖改用 awk
- `main_test.go` - 子进程超时保护
