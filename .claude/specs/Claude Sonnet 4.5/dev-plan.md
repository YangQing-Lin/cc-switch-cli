# 单元测试补充 - 开发计划

## 概述
将 cc-switch-cli 项目的测试覆盖率从当前的 10.87% 提升至 90% 以上，补齐核心业务逻辑、文件操作、CLI/TUI 交互等模块的单元测试。

## 任务分解

### Task 1: 补齐 config 模块低覆盖区与测试工具包
- **ID**: task-1
- **描述**: 为 internal/config/ 中的 mcp_sync、validate、path、provider 子模块编写测试用例，覆盖 MCP 同步、配置校验、路径规范化、Provider 操作等逻辑；同时为 internal/testutil/ 编写测试以确保测试工具包自身的正确性
- **文件范围**:
  - `internal/config/mcp_sync.go` 及对应测试文件
  - `internal/config/validate.go` 及对应测试文件
  - `internal/config/path.go` 及对应测试文件
  - `internal/config/provider.go` 及对应测试文件
  - `internal/testutil/*.go` 及对应测试文件
- **依赖**: 无
- **测试命令**:
  ```bash
  go test ./internal/config -v -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | grep total
  go test ./internal/testutil -v -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | grep total
  ```
- **测试焦点**:
  - MCP 同步：从 Claude settings 读取 MCP 配置、写入 CCS Provider、双向同步流程、错误处理
  - 配置校验：空值检测、格式校验、Provider 依赖检查、Codex 特殊字段验证
  - 路径规范化：空路径处理、相对路径展开、用户目录替换、跨平台兼容性
  - Provider 操作：添加/删除/查找/切换 Provider、重复名称处理、不存在的 Provider 处理
  - 测试工具包：Mock 文件系统、临时目录创建、配置生成器的正确性

### Task 2: 补测模板系统与差异计算
- **ID**: task-2
- **描述**: 为 internal/template/ 的 diff、manager、targets 模块编写测试，覆盖模板渲染、差异计算、多目标配置生成、文件读写异常等场景
- **文件范围**:
  - `internal/template/diff.go` 及对应测试文件
  - `internal/template/manager.go` 及对应测试文件
  - `internal/template/targets.go` 及对应测试文件
- **依赖**: 无
- **测试命令**:
  ```bash
  go test ./internal/template -v -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | grep total
  ```
- **测试焦点**:
  - 差异计算：内容一致性检测、diff 格式输出、空文件/不存在文件处理、字符编码兼容性
  - 模板管理器：模板注册、变量替换、嵌套模板、渲染错误处理、缓存机制
  - 多目标渲染：批量生成 Claude/Codex 配置、目标路径冲突检测、原子写入失败回滚、权限错误处理

### Task 3: 补测 Claude 插件与锁文件流程
- **ID**: task-3
- **描述**: 为 internal/claude/ 的插件配置读写和 internal/lock/ 的文件锁机制编写测试，覆盖并发保护、进程检测、配置解析、错误恢复等场景
- **文件范围**:
  - `internal/claude/*.go` 及对应测试文件
  - `internal/lock/*.go` 及对应测试文件
- **依赖**: 无
- **测试命令**:
  ```bash
  go test ./internal/claude -v -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | grep total
  go test ./internal/lock -v -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | grep total
  ```
- **测试焦点**:
  - Claude 插件：读取 MCP 配置、合并插件设置、权限校验、无效 JSON 处理、原子更新
  - 文件锁机制：获取/释放锁、超时机制、死锁检测、进程崩溃后的孤儿锁清理、跨平台兼容性（Windows tasklist vs Unix pgrep）

### Task 4: 补测版本更新流程
- **ID**: task-4
- **描述**: 为 internal/version/ 的版本检查、下载、解压、校验逻辑编写测试，覆盖网络错误、文件损坏、权限问题、跨平台二进制替换等场景
- **文件范围**:
  - `internal/version/*.go` 及对应测试文件
- **依赖**: 无
- **测试命令**:
  ```bash
  go test ./internal/version -v -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | grep total
  ```
- **测试焦点**:
  - 版本检查：GitHub API 请求、语义化版本比较、无网络时的降级处理、频率限制
  - 下载流程：断点续传、校验和验证、临时文件清理、磁盘空间不足处理
  - 二进制替换：自我更新逻辑、Windows 文件锁、权限提升、回滚机制（.bak 文件）
  - 错误路径：无效的版本号、损坏的压缩包、不兼容的平台、校验和不匹配

### Task 5: 补测 CLI 与 TUI 关键路径
- **ID**: task-5
- **描述**: 为 cmd/ 的命令行处理和 internal/tui/ 的 TUI 模型编写测试，覆盖参数解析、子命令执行、错误输出、TUI 状态机变更等场景
- **文件范围**:
  - `cmd/*.go` 及对应测试文件
  - `internal/tui/**/*.go` 及对应测试文件
- **依赖**: task-1（需要 config 模块的 Provider 管理和路径逻辑）
- **测试命令**:
  ```bash
  go test ./cmd -v -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | grep total
  go test ./internal/tui -v -coverprofile=coverage.out -covermode=atomic
  go tool cover -func=coverage.out | grep total
  ```
- **测试焦点**:
  - CLI 命令：参数解析（Cobra SetArgs）、子命令路由、错误输出捕获、帮助信息生成、便携模式标志位处理
  - TUI 模型：状态变更（菜单导航、表单输入、确认弹窗）、按键处理、Init/Update/View 生命周期、错误状态渲染
  - 集成场景：`ccs config add` 的完整流程、`ccs <name>` 快速切换、TUI 启动与退出、配置校验失败的用户提示

## 验收标准
- [ ] 所有任务完成后，项目整体测试覆盖率达到 ≥90%
- [ ] 每个子模块（internal/config, internal/template, internal/claude, internal/lock, internal/version, cmd, internal/tui）的覆盖率均 ≥90%
- [ ] 所有新增测试用例通过 `go test ./...`
- [ ] 测试覆盖正常路径、边界条件、错误处理、状态转换四大类场景
- [ ] 跨平台测试通过（至少在 Linux 和 Windows 环境验证）
- [ ] 测试代码无冗余断言，每个测试函数单一职责
- [ ] 所有文件操作使用 `t.TempDir()` 创建临时目录，避免污染开发环境

## 技术注意事项
- **测试框架**: 使用 Go 标准库 `testing`，保持与现有测试代码风格一致
- **Mock 策略**:
  - 文件系统操作：使用 `t.TempDir()` 创建隔离环境
  - 外部依赖：采用小接口（如 `io.Reader`/`io.Writer`）+ 函数注入实现 Mock
  - 网络请求：使用 `httptest.Server` 模拟 API 响应
- **CLI 测试**: 使用 Cobra 的 `cmd.SetArgs()` + `bytes.Buffer` 捕获输出
- **TUI 测试**: 专注模型状态变更，不做渲染快照对比；使用 `tea.Batch` 和 `tea.Cmd` 模拟消息传递
- **跨平台兼容**: 路径操作统一使用 `filepath.Join()`，避免硬编码路径分隔符；进程检测逻辑需分别测试 Windows 和 Unix 分支
- **原子性验证**: 关键配置写入需验证临时文件 + 重命名机制，失败时回滚逻辑需覆盖
- **并发安全**: 文件锁测试需包含多 goroutine 竞争场景，验证互斥性
- **便携模式**: 需在 `portable.json` 存在和不存在两种场景下测试配置目录路径
- **Codex 分离**: Codex 的 auth.json 和 config.toml 需分别测试读写逻辑
