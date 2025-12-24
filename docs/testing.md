# CC-Switch CLI 测试文档

> 项目测试指南和测试结构说明
> 最后更新: 2025-12-24

## 测试概述

本项目使用 Go 标准测试框架，编写了全面的单元测试覆盖核心功能。

## 运行测试

### 运行所有测试

```bash
# 运行所有测试
go test ./...

# 运行指定包的测试
go test ./internal/utils
go test ./internal/settings
go test ./internal/i18n
go test ./internal/vscode

# 运行所有 internal 包的测试
go test ./internal/...
```

### 查看详细输出

```bash
# 详细模式
go test -v ./internal/...

# 显示测试覆盖率
go test -cover ./internal/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```

### 运行特定测试

```bash
# 运行特定测试函数
go test -run TestFileExists ./internal/utils

# 运行匹配模式的测试
go test -run "TestAtomic.*" ./internal/utils
```

## 测试结构

### 测试文件组织

```
cc-switch-cli/
├── internal/
│   ├── utils/
│   │   ├── file.go
│   │   └── file_test.go          # utils 包的测试
│   ├── settings/
│   │   ├── settings.go
│   │   └── settings_test.go      # settings 包的测试
│   ├── i18n/
│   │   ├── i18n.go
│   │   └── i18n_test.go          # i18n 包的测试
│   ├── vscode/
│   │   ├── vscode.go
│   │   └── vscode_test.go        # vscode 包的测试
│   └── testutil/
│       └── testutil.go            # 测试工具函数
```

## 已实现的测试

### 1. internal/utils 包测试

**覆盖率: 69.7%**

测试文件: `internal/utils/file_test.go`

- ✅ `TestFileExists` - 测试文件存在性检查
- ✅ `TestAtomicWriteFile` - 测试原子文件写入
  - 写入新文件
  - 覆盖已存在文件
  - 设置文件权限
  - 默认权限处理
- ✅ `TestWriteJSONFile` - 测试 JSON 文件写入
- ✅ `TestReadJSONFile` - 测试 JSON 文件读取
  - 有效 JSON
  - 无效 JSON
  - 空文件
  - 文件不存在
- ✅ `TestBackupFile` - 测试文件备份
- ✅ `TestCopyFile` - 测试文件复制

**关键功能:**
- 原子文件操作
- 文件权限管理
- JSON 序列化/反序列化
- 跨平台兼容性

### 2. internal/settings 包测试

**覆盖率: 82.4%**

测试文件: `internal/settings/settings_test.go`

- ✅ `TestNewManager` - 测试设置管理器创建
- ✅ `TestSetLanguage` - 测试语言设置
  - 设置英文
  - 设置中文
  - 设置无效语言
  - 空字符串
- ✅ `TestSetConfigDir` - 测试自定义配置目录
- ✅ `TestGet` - 测试获取所有设置
- ✅ `TestLoadExistingSettings` - 测试加载已存在的设置
- ✅ `TestSaveFilePermissions` - 测试文件权限（Unix）
- ✅ `TestGetSettingsPath` - 测试设置文件路径

**关键功能:**
- 设置持久化
- 语言切换验证
- 配置目录管理
- 文件权限 (0600)

### 3. internal/i18n 包测试

**覆盖率: 60.0%**

测试文件: `internal/i18n/i18n_test.go`

- ✅ `TestSetLanguage` - 测试设置语言
- ✅ `TestGetLanguage` - 测试获取当前语言
- ✅ `TestT` - 测试翻译函数
  - 中文翻译
  - 英文翻译
  - 不存在的 key
  - 无效语言降级
- ✅ `TestTf` - 测试格式化翻译
- ✅ `TestMessages` - 测试翻译完整性
  - 验证中英文 key 一致性

**关键功能:**
- 双语翻译（中/英）
- 翻译降级机制
- 翻译完整性验证

### 4. internal/vscode 包测试

**覆盖率: 25.0%**

测试文件: `internal/vscode/vscode_test.go`

- ✅ `TestGetVsCodeConfigPath` - 测试 VS Code 配置路径
- ✅ `TestGetCursorConfigPath` - 测试 Cursor 配置路径
- ✅ `TestSupportedApps` - 测试支持的应用列表
- ✅ `TestIsRunning` - 测试进程检测（基础）

**关键功能:**
- 跨平台路径解析
- 多编辑器支持
- 进程检测

### 5. internal/config 包测试

**覆盖率: 32.1%**

测试文件: `internal/config/config_test.go`

- ✅ `TestNewManager` - 测试配置管理器创建
- ✅ `TestAddProvider` - 测试添加Provider
  - Claude提供商
  - Codex提供商
- ✅ `TestListProviders` - 测试列出Providers
- ✅ `TestGetProvider` - 测试获取指定Provider
- ✅ `TestDeleteProvider` - 测试删除Provider
- ✅ `TestUpdateProvider` - 测试更新Provider
- ✅ `TestConfigPersistence` - 测试配置持久化
- ✅ `TestConfigFileFormat` - 测试配置文件格式
- ✅ `TestProviderID` - 测试Provider ID唯一性
- ✅ `TestMultiAppIsolation` - 测试多应用隔离

**关键功能:**
- Provider CRUD 操作
- 配置持久化验证
- 多应用独立管理
- 配置文件格式验证
- ID 唯一性保证

### 6. 测试工具包

文件: `internal/testutil/testutil.go`

提供通用测试辅助函数：

- `CreateTempDir` - 创建临时测试目录
- `CreateTempFile` - 创建临时测试文件
- `AssertFileExists` - 断言文件存在
- `AssertFileNotExists` - 断言文件不存在
- `AssertFileContent` - 断言文件内容
- `AssertFileMode` - 断言文件权限

## 测试策略

### 1. 单元测试

- **隔离性**: 每个测试独立运行，使用临时目录
- **可重复性**: 测试可以多次运行得到相同结果
- **跨平台**: 处理 Windows/macOS/Linux 的差异

### 2. 测试用例设计

使用表驱动测试 (Table-Driven Tests):

```go
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{
    {
        name:    "正常情况",
        input:   "test",
        want:    "test",
        wantErr: false,
    },
    // 更多测试用例...
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // 测试逻辑
    })
}
```

### 3. 跨平台测试

处理平台差异:

```go
if runtime.GOOS == "windows" {
    t.Skip("跳过 Windows 上的权限测试")
}
```

### 4. 临时环境

使用 Go 1.15+ 的 `t.TempDir()` 自动清理:

```go
func TestExample(t *testing.T) {
    tmpDir := t.TempDir() // 测试结束后自动删除
    // 使用临时目录进行测试
}
```

## 测试覆盖率

### 当前覆盖率 (2025-12-24)

| 包 | 覆盖率 | 状态 |
|---|------:|:----:|
| main | 100.0% | ✅ |
| cmd | 78.3% | ⚠️ |
| internal/backup | 88.2% | ⚠️ |
| internal/claude | 80.3% | ⚠️ |
| internal/config | 90.0% | ✅ |
| internal/i18n | 100.0% | ✅ |
| internal/lock | 90.5% | ✅ |
| internal/portable | 86.7% | ⚠️ |
| internal/settings | 94.1% | ✅ |
| internal/template | 91.9% | ✅ |
| internal/testutil | 79.5% | ⚠️ |
| internal/tui | 89.9% | ✅ |
| internal/utils | 80.3% | ⚠️ |
| internal/version | 91.3% | ✅ |
| **总计** | **89.7%** | ⚠️ |

### CI 覆盖率门禁

项目设置了 **89%** 的覆盖率阈值，低于此值的 PR 将无法合并。

```bash
# 本地检查覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

## 集成测试

### 已实现的集成测试

**测试文件: `test/integration/basic_test.go`**

集成测试验证了多个组件协同工作的场景：

#### 1. TestBasicProviderOperations
测试基本的 Provider 操作流程：
- 添加 Provider
- 列出 Providers
- 获取指定 Provider

#### 2. TestProviderPersistence
测试配置持久化：
- 创建配置管理器
- 添加 Provider
- 重新创建管理器（模拟重启）
- 验证 Provider 仍然存在

#### 3. TestMultiAppSupport
测试多应用支持：
- 为 Claude 和 Codex 分别添加 Provider
- 验证两个应用的 Provider 相互独立
- 验证配置正确保存

#### 4. TestConfigFileStructure
测试配置文件结构：
- 创建 Provider
- 读取并解析配置文件
- 验证 v2 配置格式正确
- 验证 Apps 结构完整

#### 5. TestProviderValidation
测试 Provider 验证：
- 测试空名称 Provider
- 测试空 token Provider
- 记录当前验证行为

#### 6. TestConcurrentAccess
测试并发访问安全性：
- 快速连续添加多个 Provider
- 验证所有 Provider 都正确保存
- 重新加载验证数据完整性

### 运行集成测试

```bash
# 运行所有集成测试
go test -v ./test/integration/...

# 运行特定集成测试
go test -v ./test/integration/ -run TestBasicProviderOperations

# 带超时运行
go test -v ./test/integration/... -timeout=60s
```

### 集成测试覆盖的功能

✅ Provider CRUD 操作
✅ 配置持久化
✅ 多应用支持
✅ 配置文件格式验证
✅ 并发访问保护
✅ 数据完整性验证

## 待完成的测试

### 单元测试

- [ ] internal/config 包完整测试
  - 更多边界情况测试
  - 错误处理测试

### 高级集成测试

- [ ] 完整切换流程测试（包含实际文件写入）
- [ ] 配置迁移测试（v1 到 v2）
- [ ] 回滚机制测试（切换失败恢复）
- [ ] 跨平台路径处理测试

## 持续集成

### GitHub Actions 工作流

项目配置了自动化测试工作流 (`.github/workflows/test.yml`)：

**触发条件:**
- 推送到 `master` 或 `dev` 分支
- 提交 PR 到 `master` 分支

**工作流程:**

1. **多平台测试** (`test` job)
   - 在 Ubuntu、macOS、Windows 上并行运行
   - 执行 `go vet` 静态检查
   - 执行 `gofmt` 格式检查 (仅 Linux)
   - 运行带 race detector 的测试
   - 编译验证

2. **覆盖率门禁** (`coverage` job)
   - 生成覆盖率报告
   - 检查覆盖率是否达到 89% 阈值
   - 上传覆盖率报告到 Artifacts
   - 在 PR Summary 中显示覆盖率表格

### 本地验证

在提交前执行以下命令确保 CI 能通过：

```bash
# 格式化代码
go fmt ./...

# 静态检查
go vet ./...

# 运行带 race detector 的测试
go test -race -v ./...

# 检查覆盖率
go test ./... -coverprofile=coverage.out
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Coverage: ${COVERAGE}%"
```

## 测试最佳实践

### 1. 命名规范

- 测试文件: `xxx_test.go`
- 测试函数: `TestXxx(t *testing.T)`
- 子测试: `t.Run("测试场景描述", func(t *testing.T) {...})`

### 2. 断言清晰

```go
if got != want {
    t.Errorf("函数名() = %v, want %v", got, want)
}
```

### 3. 使用辅助函数

```go
func TestExample(t *testing.T) {
    helper := func(t *testing.T) {
        t.Helper() // 标记为辅助函数
        // 辅助逻辑
    }
    helper(t)
}
```

### 4. 清理资源

```go
t.Cleanup(func() {
    // 清理代码
})
```

## 故障排除

### Windows 权限测试

Windows 不支持 Unix 风格的文件权限，相关测试会自动跳过:

```go
if runtime.GOOS == "windows" {
    t.Skip("跳过 Windows 上的权限测试")
}
```

### 临时文件清理

使用 `t.TempDir()` 和 `t.Cleanup()` 确保测试后自动清理。

### 环境变量

测试中修改环境变量使用 `t.Setenv()`:

```go
t.Setenv("HOME", tmpDir)
```

## 贡献指南

### 添加新测试

1. 在对应包目录创建 `*_test.go` 文件
2. 使用表驱动测试设计
3. 考虑跨平台兼容性
4. 使用临时目录隔离测试
5. 运行 `go test -v` 验证

### 提高覆盖率

```bash
# 查看未覆盖的代码
go test -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out
```

## 参考资源

- [Go Testing 官方文档](https://golang.org/pkg/testing/)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Test Coverage](https://go.dev/blog/cover)
