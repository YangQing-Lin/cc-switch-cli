# CC-Switch CLI 测试文档

> 项目测试指南和测试结构说明
> 最后更新: 2025-10-02

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

### 5. 测试工具包

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

## 测试覆盖率目标

| 包 | 当前覆盖率 | 目标覆盖率 |
|---|-----------|----------|
| internal/utils | 69.7% | 80% |
| internal/settings | 82.4% | 85% |
| internal/i18n | 60.0% | 70% |
| internal/vscode | 25.0% | 50% |
| **平均** | **59.3%** | **70%** |

## 待完成的测试

### 单元测试

- [ ] internal/config 包完整测试
  - Provider CRUD 操作
  - 配置验证
  - 切换流程

### 集成测试

- [ ] 完整切换流程测试
- [ ] 配置迁移测试
- [ ] 回滚机制测试
- [ ] 跨平台兼容性测试

## 持续集成

### GitHub Actions 配置示例

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.21']

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go }}

      - name: Run tests
        run: go test -v -cover ./...

      - name: Generate coverage
        run: go test -coverprofile=coverage.out ./...
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
