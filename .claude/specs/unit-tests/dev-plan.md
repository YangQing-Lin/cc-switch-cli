# 单元测试全面实施 - Development Plan

## Overview
为 cc-switch-cli 项目实现全面的单元测试覆盖，重写所有现有测试并达到 ≥90% 代码覆盖率。该项目是一个跨平台 Go CLI 工具，用于管理多个 Claude/Codex/Gemini 中转站配置，支持交互式 TUI 和命令行两种操作模式。

## Task Breakdown

### Task 1: Config Package Full Unit Test Rewrite
- **ID**: task-1
- **Description**: 重写 internal/config 包的所有单元测试，覆盖配置 CRUD、路径解析、便携模式、JSON/TOML 序列化、MCP 同步、Gemini 配置等核心逻辑
- **File Scope**:
  - `internal/config/config.go` - 配置管理器核心逻辑
  - `internal/config/provider.go` - Provider CRUD 操作
  - `internal/config/validate.go` - 配置验证逻辑
  - `internal/config/switch.go` - 配置切换流程（writeProviderConfig、SwitchProviderForApp）
  - `internal/config/path.go` - 跨平台路径解析
  - `internal/config/migration.go` - 配置版本迁移（v1→v2 格式）
  - `internal/config/types.go` - 数据结构定义
  - `internal/config/codex.go` - Codex 配置管理（auth.json + config.toml）
  - `internal/config/gemini.go` - Gemini 配置管理
  - `internal/config/mcp.go` - MCP 服务器配置管理
  - `internal/config/mcp_sync.go` - MCP 同步逻辑
  - 测试文件：`internal/config/config_test.go`, `internal/config/mcp_test.go`, `internal/config/gemini_test.go`（扩展）
- **Dependencies**: None（基础层）
- **Test Command**: `go test ./internal/config -v -cover -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total`
- **Test Focus**:
  - **CRUD 操作**：AddProvider, UpdateProvider, DeleteProvider, GetProvider, ListProviders（含多应用隔离）
  - **配置切换**：SwitchProviderForApp 原子写入、回滚机制、live 配置写入（Claude settings.json、Codex auth.json/config.toml）
  - **路径解析**：GetConfigPath、GetLiveConfigPath（便携模式 vs 正常模式）
  - **序列化**：自定义 Marshal/Unmarshal 保留未知字段
  - **迁移逻辑**：旧版本配置自动迁移、归档备份
  - **验证逻辑**：Token 长度限制（≤1000）、BaseURL 格式、Model 存在性
  - **MCP 同步**：Providers ↔ settings.json 的 mcpServers 双向同步
  - **边界条件**：空配置、损坏的 JSON、并发写入、权限错误
  - **跨平台**：Windows vs Unix 路径处理（使用 filepath.Join）

### Task 2: Foundation Packages Unit Test Rewrite
- **ID**: task-2
- **Description**: 重写基础设施层包的单元测试，包括文件操作、备份管理、i18n、便携模式检测、锁机制等
- **File Scope**:
  - `internal/utils/file.go` - 原子文件写入、JSON 读写、文件存在性检查
  - `internal/backup/backup.go` - 配置备份和恢复逻辑
  - `internal/settings/settings.go` - Claude settings.json 管理
  - `internal/i18n/i18n.go` - 多语言支持（中/英）
  - `internal/portable/portable.go` - 便携模式检测（portable.json 标记文件）
  - `internal/lock/lock.go` - 进程互斥锁
  - `internal/claude/plugin.go` - Claude 插件配置管理
  - `internal/testutil/testutil.go` - 测试工具扩展（新增 WithTempHome、CaptureOutput、MockHTTPClient 等）
  - 测试文件：`internal/utils/file_test.go`, `internal/backup/backup_test.go`, `internal/settings/settings_test.go`, `internal/i18n/i18n_test.go`, `internal/portable/portable_test.go`（扩展）
- **Dependencies**: None（基础层）
- **Test Command**: `go test ./internal/utils ./internal/backup ./internal/settings ./internal/i18n ./internal/portable ./internal/lock ./internal/claude ./internal/testutil -v -cover -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total`
- **Test Focus**:
  - **原子写入**：WriteJSONFile 使用临时文件 + 重命名的原子性
  - **备份管理**：创建备份、列出备份、恢复备份、清理旧备份
  - **便携模式**：IsPortable 检测逻辑、GetDataDir 路径切换
  - **i18n**：语言切换、翻译键缺失时的 fallback
  - **锁机制**：进程独占锁、超时释放、跨平台兼容性
  - **文件操作**：权限设置（0600 vs 0644）、跨平台路径处理
  - **错误处理**：磁盘满、权限不足、文件损坏等异常场景
  - **测试工具**：WithTempHome（临时 HOME 环境变量）、CaptureOutput（捕获 stdout/stderr）

### Task 3: Template and TUI Unit Tests
- **ID**: task-3
- **Description**: 为模板管理和 TUI 组件编写单元测试，重点覆盖模板 CRUD、diff 生成、目标文件管理和 Bubble Tea 模型状态转换
- **File Scope**:
  - `internal/template/manager.go` - 模板管理器（CRUD）
  - `internal/template/diff.go` - 配置 diff 计算（使用 go-udiff）
  - `internal/template/types.go` - 模板数据结构
  - `internal/template/targets.go` - 模板应用目标管理
  - `internal/tui/helpers.go` - TUI 辅助函数
  - `internal/tui/styles.go` - Lipgloss 样式定义
  - `internal/tui/tui.go` - Bubble Tea 主模型
  - `internal/tui/*.go` - 各个 TUI 视图（list, form, delete, backup, template_manager 等）
  - 新建测试文件：`internal/template/manager_test.go`, `internal/template/diff_test.go`, `internal/tui/helpers_test.go`
- **Dependencies**: task-1（config 包）
- **Test Command**: `go test ./internal/template ./internal/tui -v -cover -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total`
- **Test Focus**:
  - **模板 CRUD**：创建、读取、更新、删除模板配置
  - **Diff 生成**：两个配置之间的差异计算（go-udiff 格式）
  - **目标管理**：模板应用到多个 Provider 的逻辑
  - **TUI 模型**：Bubble Tea 的 Update/View 方法状态转换
  - **按键处理**：使用 BubbleTeaTestHelper 模拟 tea.KeyMsg 序列
  - **视图渲染**：验证输出字符串包含预期内容（避免完整 UI 断言）
  - **错误处理**：模板不存在、应用失败时的 UI 反馈
  - **表驱动测试**：覆盖不同的按键序列和状态组合

### Task 4: CLI Commands and Entry Point Tests
- **ID**: task-4
- **Description**: 为 Cobra 命令和主入口编写单元测试，验证命令行参数解析、stdout/stderr 输出和退出码
- **File Scope**:
  - `cmd/*.go` - 所有 Cobra 命令（root, config, add, delete, switch, backup, restore, template, check, version, update, portable 等）
  - `main.go` - 主入口函数
  - 新建测试文件：`cmd/root_test.go`, `cmd/config_test.go`, `cmd/template_test.go`, `main_test.go`
- **Dependencies**: task-1, task-2, task-3（依赖所有底层包）
- **Test Command**: `go test ./cmd -v -cover -coverprofile=coverage.out && go test . -v -cover -coverprofile=main_coverage.out && go tool cover -func=coverage.out | grep total`
- **Test Focus**:
  - **命令解析**：Cobra flags 和 args 解析正确性
  - **输出捕获**：使用 CaptureOutput 验证 stdout/stderr 内容
  - **退出码**：验证成功/失败场景的退出码（0 vs 1）
  - **子命令**：config add/delete/list、template save/apply/delete、backup/restore 等
  - **错误处理**：缺少参数、无效配置、权限错误时的错误消息
  - **交互式模式**：--dir、--portable 等全局 flags
  - **集成测试**：完整的命令执行流程（使用临时配置目录）
  - **表驱动测试**：覆盖不同的参数组合和边界条件

### Task 5: Version Update and Installation Logic
- **ID**: task-5
- **Description**: 为版本更新和自更新逻辑编写单元测试，使用 HTTP mock 和归档生成工具
- **File Scope**:
  - `internal/version/version.go` - 版本检查、GitHub API 调用、归档下载和解压逻辑
  - 测试文件：`internal/version/version_test.go`（扩展）
- **Dependencies**: task-2（依赖 testutil 的 MockHTTPClient 和 CreateTestArchive）
- **Test Command**: `go test ./internal/version -v -cover -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total`
- **Test Focus**:
  - **HTTP Mocking**：使用 MockHTTPClient 模拟 GitHub API 响应（latest release、asset download）
  - **归档提取**：CreateTestArchive 生成 .zip/.tar.gz 测试文件，验证 extractBinary 逻辑
  - **版本比较**：比较当前版本和最新版本（语义化版本）
  - **平台检测**：getArchiveNameForPlatform 根据 GOOS/GOARCH 生成正确的文件名
  - **错误处理**：网络超时、无效归档、权限错误、磁盘空间不足
  - **自更新流程**：下载、验证、替换可执行文件的完整流程
  - **跨平台兼容性**：Windows (.zip) vs Unix (.tar.gz) 的归档格式处理

## Test Utilities Enhancement (in internal/testutil/testutil.go)

为支持上述测试任务，需要在 `internal/testutil/testutil.go` 中新增以下工具函数：

1. **WithTempHome(t *testing.T, setupFunc func(homeDir string))**
   - 设置临时 HOME/USERPROFILE 环境变量（使用 t.Setenv）
   - 在 setupFunc 中允许创建 .claude/settings.json、.codex/auth.json 等测试文件

2. **WithTempCWD(t *testing.T, setupFunc func(cwd string))**
   - 临时更改工作目录（用于测试便携模式）
   - 自动恢复原始 CWD

3. **CaptureOutput(t *testing.T, fn func()) (stdout, stderr string)**
   - 捕获函数执行期间的 stdout 和 stderr 输出
   - 用于测试 CLI 命令的打印内容

4. **MockHTTPClient(t *testing.T, responses map[string]MockResponse) *http.Client**
   - 返回一个使用自定义 RoundTripper 的 http.Client
   - MockResponse 结构体包含 StatusCode、Body、Headers

5. **CreateTestArchive(t *testing.T, format string, files map[string][]byte) string**
   - 生成临时的 .zip 或 .tar.gz 归档文件
   - files 参数为 map[文件路径]文件内容
   - 返回归档文件路径

6. **BubbleTeaTestHelper(t *testing.T, model tea.Model, keys []string) tea.Model**
   - 构造 tea.KeyMsg 序列并依次调用 model.Update
   - 返回最终状态的 model
   - 用于测试 TUI 的按键交互

## Acceptance Criteria

- [ ] Task 1: internal/config 包代码覆盖率 ≥90%
- [ ] Task 2: 所有基础设施包（utils, backup, settings, i18n, portable, lock, claude, testutil）代码覆盖率 ≥90%
- [ ] Task 3: internal/template 和 internal/tui 包代码覆盖率 ≥90%（TUI 视图渲染可豁免至 ≥80%）
- [ ] Task 4: cmd 包和 main.go 代码覆盖率 ≥90%
- [ ] Task 5: internal/version 包代码覆盖率 ≥90%
- [ ] 所有测试使用表驱动方式（table-driven tests）
- [ ] 所有测试使用 t.TempDir() 和 t.Setenv() 确保隔离性
- [ ] 所有文件操作和 HTTP 调用使用 mock（无外部依赖）
- [ ] 运行 `go test ./... -race` 无数据竞争
- [ ] 运行 `go test ./... -v` 所有测试通过
- [ ] 运行 `go fmt ./...` 代码格式化一致
- [ ] 每个测试文件包含 Happy Path、Edge Cases、Error Handling 三类场景

## Technical Notes

### 关键技术决策

1. **配置格式版本管理**
   - 使用 MultiAppConfig.Version = 2 标识新展平格式
   - 自动迁移旧版本（嵌套 "apps" 键）到新格式（顶层 claude/codex/gemini 键）
   - 迁移前创建归档备份（archive/ 目录）

2. **原子写入保证**
   - 所有配置文件写入使用 WriteJSONFile（临时文件 + Rename）
   - 确保断电或进程崩溃时不丢失数据
   - 切换配置时使用两阶段提交（writeProviderConfig + Save）

3. **跨平台路径处理**
   - 始终使用 filepath.Join 而非字符串拼接
   - 便携模式检测：优先查找可执行文件目录的 portable.json

4. **测试隔离性**
   - 使用 t.TempDir() 创建临时配置目录
   - 使用 t.Setenv() 设置临时环境变量（HOME、USERPROFILE）
   - 所有测试不依赖真实 ~/.claude 或 ~/.codex 目录

5. **TUI 测试策略**
   - 避免测试 Lipgloss 渲染的完整输出（脆弱且难维护）
   - 重点测试状态转换逻辑和业务数据更新
   - 使用 strings.Contains 验证关键内容片段

6. **覆盖率要求**
   - 核心包（config、utils、backup）要求 ≥90%
   - TUI 视图渲染代码允许降至 ≥80%（View 方法复杂度高）
   - 使用 `go tool cover -func=coverage.out | grep total` 验证

### 约束条件

1. **Go 版本**: go 1.24.0（使用 range over function 和其他新特性）
2. **测试框架**: 使用标准库 testing，无第三方测试框架
3. **并发安全**: 所有测试需通过 `-race` 检测
4. **性能预算**: 单个测试文件运行时间 <5s（使用 `go test -timeout 10s`）
5. **操作系统兼容性**: 测试需在 Linux/macOS/Windows 上通过（CI 环境）
6. **配置隔离**: 测试不得修改用户实际配置文件（~/.claude、~/.codex、~/.gemini）

### 测试编写规范

1. **命名约定**
   - 测试函数：`Test<FunctionName>` 或 `Test<Scenario>`
   - 表驱动测试：使用结构体切片，包含 name/input/want/wantErr 字段
   - 辅助函数：添加 `t.Helper()` 标记

2. **错误断言**
   - 使用 `if (err != nil) != tt.wantErr` 模式
   - 验证错误消息包含预期关键字（使用 strings.Contains）

3. **文件断言**
   - 使用 testutil 的 AssertFileExists、AssertFileContent、AssertFileMode
   - 验证文件权限（Unix 系统使用 0600/0644）

4. **表驱动模板**
   ```go
   tests := []struct {
       name    string
       input   InputType
       want    OutputType
       wantErr bool
   }{
       {"happy path", validInput, expectedOutput, false},
       {"edge case: empty input", emptyInput, defaultOutput, false},
       {"error: invalid format", invalidInput, nil, true},
   }
   for _, tt := range tests {
       t.Run(tt.name, func(t *testing.T) {
           got, err := FunctionUnderTest(tt.input)
           if (err != nil) != tt.wantErr {
               t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
               return
           }
           if !reflect.DeepEqual(got, tt.want) {
               t.Errorf("got = %v, want %v", got, tt.want)
           }
       })
   }
   ```

5. **并行测试**
   - 独立的测试用例使用 `t.Parallel()`
   - 共享状态的测试（如 Manager 实例）避免并行

### 示例覆盖率目标

运行以下命令验证覆盖率：

```bash
# 单个包
go test ./internal/config -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# 全项目
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
```

预期输出示例：
```
github.com/YangQing-Lin/cc-switch-cli/internal/config     total:  (statements)  92.3%
github.com/YangQing-Lin/cc-switch-cli/internal/utils      total:  (statements)  94.1%
github.com/YangQing-Lin/cc-switch-cli/internal/template   total:  (statements)  88.7%
```

### 参考资源

- Go 测试最佳实践: https://go.dev/doc/code#Testing
- 表驱动测试: https://go.dev/wiki/TableDrivenTests
- Bubble Tea 测试: https://github.com/charmbracelet/bubbletea/tree/master/examples
- go-udiff 文档: https://github.com/aymanbagabas/go-udiff
