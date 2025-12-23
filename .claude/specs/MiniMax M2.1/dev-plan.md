# 测试覆盖增强 - 开发计划

## 概述
为 cc-switch-cli 项目核心模块补充单元测试，目标将整体测试覆盖率提升至 90% 以上。当前覆盖率存在多个模块测试缺失或不足的问题，主要集中在 config、template、tui 和 version 模块。

## 任务分解

### 任务 A: config 模块核心功能测试
- **ID**: task-a
- **Description**: 为 `internal/config` 核心模块编写测试用例，覆盖以下文件：`switch.go`（配置切换逻辑）、`provider.go`（供应商管理）、`validate.go`（配置验证）、`path.go`（路径处理）、`mcp_sync.go`（MCP 同步）
- **File Scope**: `internal/config/switch_test.go`, `internal/config/provider_test.go`, `internal/config/validate_test.go`, `internal/config/path_test.go`, `internal/config/mcp_sync_test.go`
- **Dependencies**: None
- **Test Command**: `GOCACHE=$PWD/.gocache go test ./internal/config/... -cover -v`
- **Test Focus**:
  - Happy Path: 成功切换 Claude/Codex/Gemini 配置、正确写入配置文件、更新当前 Provider 指针
  - Edge Cases: 配置不存在、目标应用不存在、文件权限错误、空配置场景
  - Error Handling: 写入失败回滚、路径获取失败、JSON 解析失败、原子写入失败
  - State Transitions: Provider 状态变更、App.Current 指针更新

### 任务 B: template 模块功能测试
- **ID**: task-b
- **Description**: 为 `internal/template` 模块编写测试用例，覆盖以下文件：`manager.go`（模板管理器）、`diff.go`（差异生成）、`targets.go`（目标路径管理）
- **File Scope**: `internal/template/manager_test.go`, `internal/template/diff_test.go`, `internal/template/targets_test.go`
- **Dependencies**: None
- **Test Command**: `GOCACHE=$PWD/.gocache go test ./internal/template/... -cover -v`
- **Test Focus**:
  - Happy Path: 成功加载内置/用户模板、添加/删除模板、生成 Diff、获取目标路径
  - Edge Cases: 模板 ID 冲突、空模板内容、分类边界条件、目标路径不存在
  - Error Handling: 配置文件解析失败、文件写入失败、模板不存在、原子写入失败
  - State Transitions: 模板添加后的内存状态、删除后的持久化状态

### 任务 C: tui 模块状态机测试
- **ID**: task-c
- **Description**: 为 `internal/tui` 模块编写 Bubble Tea 模型测试，验证各状态机的 Update 分支和消息处理
- **File Scope**: `internal/tui/app_select_test.go`, `internal/tui/list_test.go`, `internal/tui/form_test.go`, `internal/tui/delete_test.go`, `internal/tui/template_manager_test.go`
- **Dependencies**: None
- **Test Command**: `GOCACHE=$PWD/.gocache go test ./internal/tui/... -cover -v`
- **Test Focus**:
  - Happy Path: 正常状态转换、正确消息路由、成功执行操作
  - Edge Cases: 边界索引选择、空列表状态、快速连续输入
  - Error Handling: 无效消息类型、处理失败场景、取消操作
  - State Transitions: 各状态的完整生命周期、导航前后的状态变化

### 任务 D: version 版本更新测试
- **ID**: task-d
- **Description**: 为 `internal/version` 模块编写测试用例，覆盖版本检查、下载更新、二进制安装、临时目录清理等核心功能
- **File Scope**: `internal/version/version_test.go`
- **Dependencies**: None
- **Test Command**: `GOCACHE=$PWD/.gocache go test ./internal/version/... -cover -v`
- **Test Focus**:
  - Happy Path: 版本比较逻辑正确、无更新时返回 false、检查更新超时处理
  - Edge Cases: 最新版本、格式异常版本号、网络超时、HTTP 错误码
  - Error Handling: 网络请求失败、解压失败、平台不匹配、文件复制失败、备份恢复
  - State Transitions: 临时目录创建与清理、备份与回滚流程

### 任务 E: 辅助模块错误分支测试
- **ID**: task-e
- **Description**: 为 `internal/lock`、`internal/claude`、`internal/utils`、`internal/backup`、`internal/i18n`、`internal/settings` 模块补充错误分支测试
- **File Scope**: `internal/lock/lock_test.go`, `internal/claude/plugin_test.go`, `internal/utils/file_test.go`, `internal/backup/backup_test.go`
- **Dependencies**: None
- **Test Command**: `GOCACHE=$PWD/.gocache go test ./internal/lock/... ./internal/claude/... ./internal/utils/... ./internal/backup/... -cover -v`
- **Test Focus**:
  - Happy Path: 锁获取与释放、文件操作成功、备份创建成功
  - Edge Cases: 锁超时判定、便携模式边界、临时文件清理
  - Error Handling: 锁文件创建失败、文件读取/写入失败、权限错误、进程检测失败

## 验收标准
- [ ] 所有 5 个任务完成测试文件创建
- [ ] `go test ./internal/... -cover` 整体覆盖率 ≥ 90%
- [ ] config 模块覆盖率 ≥ 90%
- [ ] template 模块覆盖率 ≥ 90%
- [ ] tui 模块覆盖率 ≥ 80%（状态机测试复杂度高）
- [ ] version 模块覆盖率 ≥ 90%
- [ ] 所有测试用例通过（`go test` 无失败）
- [ ] 测试隔离性：每个测试使用 `t.TempDir()` 和 `t.Setenv()` 避免污染
- [ ] 无竞态条件：并行测试时不会相互干扰

## 技术说明
- **测试隔离**: 必须使用 `t.TempDir()` 创建临时目录，使用 `t.Setenv()` 模拟环境变量
- **HTTP Mock**: version 模块使用 `httptest.NewServer()` 模拟 GitHub API 响应
- **TUI 测试**: 使用 Bubble Tea 的 `tea.Model` 测试模式，直接调用 `Update()` 函数模拟消息
- **覆盖率绕过**: 测试命令使用 `GOCACHE=$PWD/.gocache` 规避权限问题
- **便携模式**: 测试中通过 `portable.json` 标记文件模拟便携模式
- **时间模拟**: lock 模块的锁超时测试使用 `time.Sleep()` 或 Mock 方式验证
