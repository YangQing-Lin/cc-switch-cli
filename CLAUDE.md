# cc-switch-cli 项目文档

## 项目概述

cc-switch-cli 是一个用 Go 语言编写的跨平台命令行工具，用于管理多个 Claude 中转站配置。本项目严格遵循《跨平台CLI应用设计要点.md》中总结的最佳实践。

## 架构设计

### 目录结构

```
cc-switch-cli/
├── main.go                    # 程序入口（极简设计，仅 7 行代码）
├── cmd/                       # 命令行接口层
│   ├── root.go               # 根命令（列表和切换配置）
│   ├── config.go             # config 子命令容器
│   ├── add.go                # 添加配置子命令
│   └── delete.go             # 删除配置子命令
├── internal/                  # 内部逻辑（不对外暴露）
│   ├── config/               # 配置管理核心
│   │   ├── types.go          # 数据结构定义
│   │   └── config.go         # 业务逻辑
│   └── utils/                # 工具函数
│       └── file.go           # 文件操作工具
├── go.mod                    # 依赖管理
├── go.sum                    # 依赖校验
├── README.md                 # 用户文档
└── CLAUDE.md                 # 项目技术文档（本文件）
```

### 设计原则

1. **入口简洁**：main.go 只负责调用命令执行器
2. **职责分离**：命令定义（cmd）、业务逻辑（internal/config）、工具函数（internal/utils）完全分离
3. **internal 包约束**：使用 Go 的 internal 目录约定，防止内部实现被外部引用
4. **按功能分层**：cmd 层处理用户交互，config 层处理核心逻辑，utils 层提供基础能力

## 命令结构

```
cc-switch                          # 根命令：列出所有配置
├── <config_name>                  # 位置参数：切换到指定配置
└── config                         # 子命令组
    ├── add <name>                 # 添加新配置
    └── delete <name>              # 删除配置
```

## 核心功能

### 1. 配置管理

**数据结构**（internal/config/types.go）：

- `Config`：单个中转站配置
- `ConfigStore`：所有配置的集合和当前激活配置
- `ClaudeSettings`：Claude 设置文件结构（支持保留未知字段）

**业务逻辑**（internal/config/config.go）：

- `NewManager()`：创建配置管理器
- `AddConfig()`：添加新配置
- `DeleteConfig()`：删除配置
- `GetConfig()`：获取指定配置
- `ListConfigs()`：列出所有配置
- `SwitchConfig()`：切换配置

### 2. 双层配置结构

**应用配置**（~/.cc-switch/configs.json）：
- 存储所有中转站配置
- 记录当前激活的配置
- 包含元数据（创建时间、更新时间）

**目标配置**（~/.claude/settings.json）：
- Claude 实际使用的配置文件
- 只包含当前生效的配置
- 由 cc-switch 动态更新

### 3. JSON 序列化设计

**问题**：如何保留 Claude settings.json 中的未知字段？

**解决方案**：自定义 JSON 序列化/反序列化

```go
// UnmarshalJSON: 解析时保存未知字段到 Extra map
// MarshalJSON: 序列化时合并 Extra map 中的字段
```

**优势**：
- 非侵入性：不破坏用户的其他配置
- 向前兼容：即使 Claude 添加新字段也不会丢失
- 灵活性：只更新需要管理的字段

## 跨平台兼容性

### 1. 路径处理

**原则**：使用 `filepath.Join()` 而非手动拼接路径

```go
// ✅ 正确做法
filepath.Join(home, ".cc-switch", "configs.json")

// ❌ 错误做法
home + "/.cc-switch/configs.json"
```

**实践**：
- 使用 `os.UserHomeDir()` 获取用户主目录（跨平台）
- 所有路径拼接都通过 `filepath.Join()` 处理
- Windows 和 Unix 系统使用相同的代码逻辑

### 2. 文件权限处理

```go
// 配置文件权限为 0600（仅用户可读写）- 安全考虑
WriteJSONFile(configPath, store, 0600)

// Claude 设置文件权限为 0644（用户可读写，组和其他用户可读）
WriteJSONFile(settingsPath, settings, 0644)
```

### 3. 用户输入处理

**敏感信息输入**（隐藏密码）：

```go
// 使用 golang.org/x/term 包处理密码输入
fd := int(os.Stdin.Fd())
bytePassword, err := term.ReadPassword(fd)

// 降级处理：如果隐藏输入失败，使用明文输入
if err != nil {
    reader := bufio.NewReader(os.Stdin)
    input, err := reader.ReadString('\n')
    // ...
}
```

## 用户体验设计

### 1. 双模式交互

**模式 1：命令行参数**（适合脚本化）
```bash
cc-switch config add myconfig --apikey sk-xxx --base-url https://api.example.com
```

**模式 2：交互式输入**（适合手动操作）
```bash
cc-switch config add myconfig
请输入 API Token: ********  # 隐藏输入
请输入 Base URL: https://api.example.com
请输入 Max Tokens (可选，直接回车跳过): 32000
```

### 2. 信息脱敏

**安全的 Token 显示**：

```go
func MaskToken(token string) string {
    if len(token) <= 8 {
        return "****"
    }
    return token[:4] + "..." + token[len(token)-4:]
}
```

输出示例：`sk-ab...xyz9`

### 3. 友好的输出格式

```
配置列表:
─────────
● default              Token: sk-X...YHuz  URL: https://api.example.com  Model: claude-3-opus
○ backup               Token: sk-a...b123  URL: https://backup.api.com
```

**设计要点**：
- 使用符号区分状态（● 当前激活，○ 未激活）
- 信息对齐，易于扫描
- 关键信息突出

### 4. 确认机制

**危险操作的二次确认**：

```bash
即将删除以下配置:
  名称: myconfig
  Token: sk-X...YHuz
  URL: https://api.example.com

确定要删除这个配置吗? (y/N):
```

**强制模式**（跳过确认，适合脚本）：
```bash
cc-switch config delete myconfig --force
```

## 错误处理与健壮性

### 1. 错误包装与上下文

```go
if err := manager.AddConfig(newConfig); err != nil {
    return fmt.Errorf("添加配置失败: %w", err)
}
```

**优势**：
- 保留原始错误信息
- 添加上下文描述
- 支持 `errors.Is` 和 `errors.As` 判断

### 2. 容错设计

**配置文件不存在时自动初始化**：

```go
func (m *Manager) Load() error {
    if !utils.FileExists(m.configPath) {
        m.store = &ConfigStore{
            Configs:   []Config{},
            UpdatedAt: time.Now(),
        }
        return m.Save()
    }
    // ...
}
```

**备份文件时的宽松处理**：

```go
func BackupFile(path string) error {
    if !FileExists(path) {
        return nil  // 原文件不存在不算错误
    }
    // ...
}
```

### 3. 输入验证

```go
func (c *Config) Validate() error {
    // 验证配置名称
    if c.Name == "" {
        return fmt.Errorf("配置名称不能为空")
    }

    // 验证 Token 格式
    if !strings.HasPrefix(c.AnthropicAuthToken, "sk-") {
        return fmt.Errorf("API Token 格式错误，应以 'sk-' 开头")
    }

    // 验证 URL 格式
    if _, err := url.Parse(c.AnthropicBaseURL); err != nil {
        return fmt.Errorf("无效的 Base URL: %w", err)
    }

    // 验证 MaxTokens 是否为数字
    if c.ClaudeCodeMaxTokens != "" {
        if _, err := strconv.Atoi(c.ClaudeCodeMaxTokens); err != nil {
            return fmt.Errorf("Max Tokens 必须是数字")
        }
    }

    return nil
}
```

## 依赖管理

### 核心依赖

```go
require (
    github.com/spf13/cobra v1.9.1          // 命令行框架
    golang.org/x/term v0.34.0              // 终端控制（隐藏输入）
)
```

**设计考虑**：
- 优先使用标准库
- 引入依赖前评估必要性
- 选择维护活跃、社区认可的库

### 最小化依赖原则

**核心依赖只有 2 个**：
- `cobra`：命令行框架，行业标准
- `term`：隐藏密码输入，官方扩展包

## 安全性考虑

### 1. 敏感信息保护

**文件权限**：
```go
// 配置文件设置为 0600（仅所有者可读写）
WriteJSONFile(configPath, store, 0600)
```

**显示脱敏**：
```go
// 列表中只显示部分 token
Token: sk-X...YHuz
```

**隐藏输入**：
```go
// 输入 API Token 时不回显
term.ReadPassword(int(os.Stdin.Fd()))
```

### 2. 配置备份机制

```go
func BackupFile(path string) error {
    if !FileExists(path) {
        return nil
    }

    backupPath := path + ".backup"
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("读取原文件失败: %w", err)
    }

    if err := os.WriteFile(backupPath, data, 0644); err != nil {
        return fmt.Errorf("创建备份失败: %w", err)
    }

    return nil
}
```

**设计特点**：
- 每次切换配置前自动备份
- 简单可靠的备份策略（覆盖式单备份）
- 容错设计：原文件不存在时不报错

## 构建与发布

### 单个平台构建

```bash
go build -o cc-switch
```

### 多平台构建

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o cc-switch-linux-amd64

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o cc-switch-darwin-amd64

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o cc-switch-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o cc-switch-windows-amd64.exe
```

### 构建优化

```bash
# 减小二进制体积
go build -ldflags="-s -w" -o cc-switch

# 静态编译（不依赖系统库）
CGO_ENABLED=0 go build -o cc-switch
```

### 安装方式

**方式 1：go install（推荐）**
```bash
go install github.com/YangQing-Lin/cc-switch-cli@latest
```

**方式 2：源码编译**
```bash
git clone https://github.com/YangQing-Lin/cc-switch-cli.git
cd cc-switch-cli
go build -o cc-switch
```

## 使用示例

### 添加配置

```bash
# 交互式添加
cc-switch config add myconfig

# 通过参数添加
cc-switch config add myconfig \
  --apikey sk-xxx \
  --base-url https://api.example.com \
  --model claude-3-opus \
  --max-tokens 32000
```

### 列出配置

```bash
cc-switch
```

### 切换配置

```bash
cc-switch myconfig
```

### 删除配置

```bash
# 需要确认
cc-switch config delete myconfig

# 强制删除（不需要确认）
cc-switch config delete myconfig --force
```

## 开发规范

### 代码风格

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 使用 `golint` 检查代码质量

### 提交规范

- 使用语义化提交信息
- 格式：`<type>: <description>`
- 类型：feat, fix, docs, style, refactor, test, chore

### 测试策略

**推荐的测试结构**：

```
cc-switch-cli/
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   ├── config_test.go        # 单元测试
│   │   ├── types.go
│   │   └── types_test.go
│   └── utils/
│       ├── file.go
│       └── file_test.go
└── cmd/
    └── e2e_test.go                # 端到端测试
```

## 未来改进方向

### 1. 测试覆盖

- [ ] 添加单元测试
- [ ] 添加集成测试
- [ ] 添加跨平台测试

### 2. 功能增强

- [ ] 支持配置导入/导出
- [ ] 支持配置模板
- [ ] 支持配置分组
- [ ] 添加配置验证（测试连接）

### 3. 性能优化

- [ ] 添加并发保护（文件锁）
- [ ] 添加配置缓存
- [ ] 优化大量配置场景

### 4. 开发体验

- [ ] 添加调试日志选项
- [ ] 添加详细的错误信息
- [ ] 添加进度提示

### 5. 发布自动化

- [ ] 使用 GoReleaser 自动发布
- [ ] 添加 GitHub Actions CI/CD
- [ ] 提供多种安装方式（Homebrew, Scoop 等）

## 总结

cc-switch-cli 项目严格遵循《跨平台CLI应用设计要点.md》中的最佳实践：

**架构方面**：
- ✅ 清晰的分层结构
- ✅ 模块化的命令设计
- ✅ 良好的关注点分离

**跨平台方面**：
- ✅ 正确的路径处理
- ✅ 合理的权限设置
- ✅ 降级策略（输入处理）

**用户体验方面**：
- ✅ 双模式交互（CLI + 交互式）
- ✅ 信息脱敏和安全保护
- ✅ 友好的输出格式

**数据管理方面**：
- ✅ 双层配置设计
- ✅ 自动备份机制
- ✅ 保留未知字段的 JSON 处理

**安全性方面**：
- ✅ 敏感信息保护
- ✅ 文件权限控制
- ✅ 输入验证

本项目可作为构建高质量跨平台 CLI 工具的参考实现。
