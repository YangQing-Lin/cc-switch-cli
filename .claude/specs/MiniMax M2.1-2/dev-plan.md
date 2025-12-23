# cc-switch-cli 单元测试开发计划

## 概述

为 cc-switch-cli 项目添加全面单元测试，将代码覆盖率从当前的 12.6% 提升至 90% 以上。

## 任务分解

### Task 1: 核心配置与切换模块测试
- **ID**: task-1
- **Description**: 为 `internal/config` 和 `internal/lock` 包编写单元测试，覆盖配置切换、持久化、迁移、验证和 MCP 同步功能
- **File Scope**: `internal/config/**/*.go`, `internal/lock/**/*.go`
- **Dependencies**: None
- **Test Command**: `go test ./internal/config/... ./internal/lock/... -v --coverprofile=coverage_task1.out && go tool cover -func=coverage_task1.out`
- **Test Focus**:
  - 正常路径：配置切换成功、配置保存、配置迁移
  - 边界情况：空配置、单个/多个 Provider 配置
  - 错误处理：无效配置、文件权限错误、迁移冲突
  - 并发场景：锁竞争、超时处理

### Task 2: 版本更新与备份模块测试
- **ID**: task-2
- **Description**: 为 `internal/version`、`internal/backup` 和 `internal/settings` 包编写单元测试，覆盖版本检查、自更新、配置备份和设置管理
- **File Scope**: `internal/version/**/*.go`, `internal/backup/**/*.go`, `internal/settings/**/*.go`
- **Dependencies**: None
- **Test Command**: `go test ./internal/version/... ./internal/backup/... ./internal/settings/... -v --coverprofile=coverage_task2.out && go tool cover -func=coverage_task2.out`
- **Test Focus**:
  - 正常路径：版本比较、备份创建/恢复、设置读取/写入
  - 边界情况：最新版本检测、空备份文件、大文件处理
  - 错误处理：网络错误、文件不存在、权限不足

### Task 3: 模板、工具与国际化模块测试
- **ID**: task-3
- **Description**: 为 `internal/template`、`internal/utils` 和 `internal/i18n` 包编写单元测试
- **File Scope**: `internal/template/**/*.go`, `internal/utils/**/*.go`, `internal/i18n/**/*.go`
- **Dependencies**: None
- **Test Command**: `go test ./internal/template/... ./internal/utils/... ./internal/i18n/... -v --coverprofile=coverage_task3.out && go tool cover -func=coverage_task3.out`
- **Test Focus**:
  - 正常路径：模板渲染、工具函数正确性、语言切换
  - 边界情况：特殊字符处理、路径边界值、超长字符串
  - 错误处理：模板语法错误、文件路径无效、编码问题

### Task 4: Claude 插件与便携模式模块测试
- **ID**: task-4
- **Description**: 为 `internal/claude` 和 `internal/portable` 包编写单元测试，覆盖 Claude 插件集成和便携模式功能
- **File Scope**: `internal/claude/**/*.go`, `internal/portable/**/*.go`
- **Dependencies**: None
- **Test Command**: `go test ./internal/claude/... ./internal/portable/... -v --coverprofile=coverage_task4.out && go tool cover -func=coverage_task4.out`
- **Test Focus**:
  - 正常路径：便携模式检测、插件初始化
  - 边界情况：便携标记文件存在/不存在、跨平台路径处理
  - 错误处理：不可写目录、配置文件损坏

## 验收标准
- [ ] 所有测试任务代码覆盖率 ≥90%
- [ ] `go test ./...` 全部测试通过
- [ ] 测试覆盖正常路径、边界情况和错误场景
- [ ] 无竞态条件（运行 `go test -race ./...` 通过）
- [ ] 测试代码遵循项目规范（KISS、注释适度）

## 技术说明
- 测试命令统一使用 `-coverprofile` 生成覆盖率报告
- 建议使用表格驱动测试（table-driven tests）模式
- 对于文件操作使用 `os.TempDir()` 创建临时测试目录
- 使用 `github.com/stretchr/testify` 断言库（如果项目已集成）或标准 `testing` 包
- 并发测试使用 `sync.WaitGroup` 和 `goroutine`
