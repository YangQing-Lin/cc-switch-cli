# 单元测试补充计划 - Development Plan

## Overview
为 cc-switch-cli 项目补充单元测试，将代码覆盖率从当前的 12.6% 提升至 90% 以上，覆盖配置管理、TUI 界面、CLI 命令、模板系统、备份功能等核心模块。

## Task Breakdown

### Task 1: 核心配置与基础设施测试
- **ID**: task-1
- **Description**: 为核心配置管理、文件工具、锁机制、便携模式、设置管理等基础模块补充单元测试
- **File Scope**:
  - `internal/config/*.go` (migration.go, mcp_sync.go, mcp.go, validate.go, provider.go, switch.go, path.go, codex.go, gemini.go)
  - `internal/utils/*.go` (补充边界场景测试)
  - `internal/settings/*.go` (补充异常场景测试)
  - `internal/lock/*.go`
  - `internal/portable/*.go` (补充 portable 检测与路径测试)
- **Dependencies**: None
- **Test Command**:
  ```bash
  go test ./internal/config ./internal/utils ./internal/settings ./internal/lock ./internal/portable -cover -coverprofile=coverage-task1.out -coverpkg=./internal/config,./internal/utils,./internal/settings,./internal/lock,./internal/portable
  go tool cover -func=coverage-task1.out | grep total
  ```
- **Test Focus**:
  - 配置迁移正确性（旧格式 -> 新格式）
  - MCP 配置同步与验证逻辑
  - 多应用配置隔离（Claude/Codex/Gemini）
  - 配置切换流程与回滚机制
  - Gemini 环境变量配置生成
  - 文件原子写入失败场景（权限错误、磁盘满）
  - 进程锁竞争条件（并发访问配置文件）
  - 便携模式检测逻辑（portable.json 存在/不存在）
  - 配置路径解析（标准模式 vs 便携模式）

### Task 2: 模板、TUI、i18n 与 Claude 插件测试
- **ID**: task-2
- **Description**: 为模板管理器、TUI 组件、国际化系统、Claude 插件等模块补充单元测试
- **File Scope**:
  - `internal/template/*.go` (diff.go, types.go, targets.go, manager.go)
  - `internal/tui/*.go` (styles.go, helpers.go, tui.go, list.go, form.go - 核心逻辑测试)
  - `internal/i18n/*.go` (补充多语言切换测试)
  - `internal/claude/*.go` (plugin.go)
- **Dependencies**: None
- **Test Command**:
  ```bash
  go test ./internal/template ./internal/i18n ./internal/claude -cover -coverprofile=coverage-task2.out -coverpkg=./internal/template,./internal/i18n,./internal/claude
  go tool cover -func=coverage-task2.out | grep total
  ```
- **Test Focus**:
  - 模板 Diff 算法准确性（字段增删改）
  - 模板目标解析（Claude/Codex 配置路径）
  - 模板应用与保存流程
  - i18n 语言切换逻辑（中文/英文）
  - i18n 占位符替换正确性
  - Claude 插件配置解析与生成
  - TUI 核心辅助函数测试（格式化、验证、错误处理）

### Task 3: CLI 命令层测试
- **ID**: task-3
- **Description**: 为所有 CLI 命令补充单元测试，覆盖参数解析、错误处理、业务逻辑调用
- **File Scope**:
  - `cmd/*.go` (所有命令文件)
  - `main.go`
- **Dependencies**: task-1 (依赖配置管理核心逻辑测试完成)
- **Test Command**:
  ```bash
  go test ./cmd -cover -coverprofile=coverage-task3.out -coverpkg=./cmd
  go tool cover -func=coverage-task3.out | grep total
  ```
- **Test Focus**:
  - 命令参数解析（有效/无效参数）
  - 子命令路由正确性（config add/delete/update，template apply/save/delete）
  - 错误处理链路（配置文件不存在、权限错误、参数缺失）
  - Codex/Gemini 命令独立性（环境变量配置、模型参数）
  - 便携模式 flag 传递（--dir 参数）
  - 命令别名与简写支持

### Task 4: 版本更新与集成测试
- **ID**: task-4
- **Description**: 为版本检测、自动更新功能补充测试，并增加关键流程的集成测试
- **File Scope**:
  - `internal/version/*.go` (补充网络请求 Mock、解压流程测试)
  - 新增 `test/integration/*.go` (配置切换完整流程、模板应用流程)
- **Dependencies**: None
- **Test Command**:
  ```bash
  go test ./internal/version -cover -coverprofile=coverage-task4-unit.out -coverpkg=./internal/version
  go test ./test/integration -cover -coverprofile=coverage-task4-integration.out
  go tool cover -func=coverage-task4-unit.out | grep total
  ```
- **Test Focus**:
  - GitHub API 版本检测（Mock HTTP 响应）
  - 下载进度跟踪（文件大小计算、进度回调）
  - 压缩包解压逻辑（tar.gz/zip 格式）
  - 二进制替换与权限保持
  - 集成测试：完整配置切换流程（添加 Provider -> 切换 -> 验证 live 文件）
  - 集成测试：模板保存与应用流程（保存模板 -> 新建配置 -> 应用模板 -> 验证字段）

## Acceptance Criteria
- [ ] 所有核心配置管理逻辑通过单元测试（internal/config）
- [ ] 文件工具、锁机制、便携模式覆盖边界场景（internal/utils, lock, portable）
- [ ] 模板系统与 i18n 测试覆盖核心逻辑
- [ ] 所有 CLI 命令参数解析与错误处理通过测试
- [ ] 版本更新功能通过 Mock 测试
- [ ] 关键业务流程通过集成测试验证
- [ ] 总体代码覆盖率 ≥90%（执行 `go test ./... -cover` 验证）
- [ ] 所有测试在 Linux/macOS/Windows 平台通过（跨平台兼容性）

## Technical Notes
- **测试框架**: 使用 Go 标准库 `testing` 包，无需引入第三方框架
- **Mock 策略**:
  - 文件系统操作使用 `t.TempDir()` 创建隔离环境
  - HTTP 请求使用 `httptest.Server` 或自定义 `http.RoundTripper` Mock
  - 时间依赖使用可注入的时间提供者接口（如需要）
- **TUI 测试限制**: Bubble Tea 组件依赖终端交互，仅测试核心业务逻辑函数，不测试 Update/View 方法
- **并发测试**: 使用 `go test -race` 检测锁机制的竞态条件
- **覆盖率计算**:
  - 单个任务覆盖率通过 `-coverprofile` 输出
  - 总体覆盖率通过 `go test ./... -coverprofile=coverage.out` 汇总
  - 使用 `go tool cover -html=coverage.out` 生成 HTML 报告查看未覆盖代码
- **跨平台注意事项**:
  - 文件权限测试在 Windows 上跳过（使用 `runtime.GOOS` 判断）
  - 路径分隔符使用 `filepath.Join()` 而非硬编码
  - 进程检测命令区分 Windows (`tasklist`) 和 Unix/Linux (`pgrep`)
- **测试数据隔离**: 每个测试用例使用独立的临时目录，避免状态污染
- **错误场景覆盖**: 重点测试配置文件损坏、权限错误、磁盘满、网络超时等异常场景
