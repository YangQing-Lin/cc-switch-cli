# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

cc-switch-cli 是一个跨平台 Go CLI 工具，用于管理多个 Claude 中转站配置和 Codex 配置。支持交互式 TUI 和命令行两种操作模式。

## 常用命令

### 开发命令

```bash
# 构建 (Linux/macOS)
go build -o ccs

# 构建 (Windows)
go build -o ccs.exe

# 运行测试
go test ./...

# 运行特定测试
go test ./internal/config -v
go test -run TestConfigManager ./internal/config

# 代码格式化
go fmt ./...

# 静态检查
go vet ./...
```

### 使用命令

```bash
# 启动交互式 TUI
./ccs          # Linux/macOS
./ccs.exe      # Windows

# 切换配置
./ccs <配置名称>

# 查看版本
./ccs version

# 添加配置
./ccs config add <name>

# 删除配置
./ccs config delete <name>

# 导出配置
./ccs export

# 导入配置
./ccs import <file>

# 备份配置
./ccs backup

# 检查配置有效性
./ccs check

# 使用自定义配置目录
./ccs --dir /path/to/config
```

## 架构设计

### 核心概念

1. **单向配置覆盖**
   - CCS 配置始终覆盖 live 配置文件
   - 切换时直接写入目标 Provider 配置到 live 文件
   - 不回填 live 配置的修改到 CCS

2. **双层配置结构**
   - **内存配置** (`~/.cc-switch/config.json`): 存储所有供应商配置
   - **Live 配置**: Claude 使用 `~/.claude/settings.json`，Codex 使用 `~/.codex/auth.json` + `~/.codex/config.toml`

3. **多应用支持**
   - 使用展平的 JSON 结构（v2 格式）
   - 顶层键为应用名（claude、codex）
   - 自动迁移旧版本配置格式

### 目录结构

```
cc-switch-cli/
├── main.go                    # 入口文件
├── cmd/                       # 命令层
│   ├── root.go               # 根命令（TUI/切换）
│   ├── config.go             # config 子命令容器
│   ├── add.go                # 添加配置
│   ├── delete.go             # 删除配置
│   ├── update.go             # 更新配置
│   ├── show.go               # 显示配置详情
│   ├── export.go             # 导出配置
│   ├── import.go             # 导入配置
│   ├── backup.go             # 备份配置
│   ├── restore.go            # 恢复配置
│   ├── check.go              # 检查配置
│   ├── migrate.go            # 迁移配置
│   ├── validate.go           # 验证配置
│   ├── version.go            # 版本信息
│   ├── codex.go              # codex 子命令容器
│   ├── codex_add.go          # Codex 添加
│   ├── codex_delete.go       # Codex 删除
│   ├── codex_switch.go       # Codex 切换
│   ├── ui.go                 # TUI 命令
│   └── settings.go           # 设置管理
├── internal/                  # 内部包
│   ├── config/               # 配置管理
│   │   ├── types.go          # 数据结构定义
│   │   └── config.go         # 核心逻辑
│   ├── i18n/                 # 国际化
│   │   └── i18n.go           # 语言切换
│   ├── tui/                  # 交互式界面
│   │   ├── tui.go            # Bubble Tea 模型
│   │   └── styles.go         # 样式定义
│   ├── backup/               # 备份管理
│   │   └── backup.go         # 备份/恢复逻辑
│   ├── portable/             # 便携版支持
│   │   └── portable.go       # 便携模式检测
│   ├── vscode/               # VSCode 集成
│   │   └── vscode.go         # VSCode/Cursor 检测
│   ├── settings/             # 设置管理
│   │   └── settings.go       # 应用设置
│   ├── version/              # 版本管理
│   │   └── version.go        # 版本信息
│   └── utils/                # 工具函数
│       └── file.go           # 文件操作
└── test/                     # 测试
    └── integration/          # 集成测试
        └── basic_test.go
```

### 关键数据结构

```go
// 多应用配置（顶层）
type MultiAppConfig struct {
    Version int                        // 配置版本（2）
    Apps    map[string]ProviderManager // 应用名 -> 供应商管理器
}

// 单应用的供应商管理器
type ProviderManager struct {
    Providers map[string]Provider // ID -> Provider
    Current   string              // 当前激活的 Provider ID
}

// 供应商配置
type Provider struct {
    ID             string                 // 唯一标识
    Name           string                 // 显示名称
    SettingsConfig map[string]interface{} // 完整配置 JSON
    WebsiteURL     string                 // 网站 URL
    Category       string                 // 分类标签
    CreatedAt      int64                  // 创建时间（毫秒）
}
```

### 配置切换流程

切换配置时的两步流程（`SwitchProviderForApp`）：

1. **写入目标配置** (`writeProviderConfig`)
   - 从目标 Provider 的 SettingsConfig 提取配置
   - 写入 live 配置文件
   - 使用原子写入和回滚机制确保事务性

2. **持久化切换** (`Save`)
   - 更新 `app.Current` 为目标 Provider ID
   - 保存到 `~/.cc-switch/config.json`
   - 创建 `.bak.cli` 备份

### 跨平台注意事项

1. **路径处理**: 始终使用 `filepath.Join()` 而非字符串拼接
2. **文件权限**: 敏感配置使用 `0600`，一般配置使用 `0644`
3. **进程检测**: Windows 使用 `tasklist`，Unix/Linux 使用 `pgrep`
4. **用户目录**: 使用 `os.UserHomeDir()` 获取跨平台主目录
5. **开发环境**: 主要在 WSL2 (Ubuntu 20.04) 上开发和测试

### 配置文件格式

**Claude 配置** (`~/.claude/settings.json`):
```json
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "sk-xxx",
    "ANTHROPIC_BASE_URL": "https://api.example.com",
    "CLAUDE_CODE_MODEL": "claude-3-opus",
    "CLAUDE_CODE_MAX_TOKENS": "32000"
  },
  "permissions": {
    "allow": [],
    "deny": []
  }
}
```

**Codex 配置** (`~/.codex/auth.json` + `~/.codex/config.toml`):
```json
{
  "OPENAI_API_KEY": "sk-xxx"
}
```
```toml
model_provider = "custom"
model = "gpt-5-codex"
model_reasoning_effort = "high"

[model_providers.custom]
name = "custom"
base_url = "https://api.example.com"
wire_api = "responses"
```

### 重要实现细节

1. **配置迁移**: 自动检测并迁移 v1 和 v2-old 格式到 v2
2. **回滚机制**: 写入失败时自动恢复 `.rollback` 备份
3. **原子写入**: 使用临时文件 + 重命名确保原子性
4. **损坏恢复**: 配置文件损坏时自动备份并创建默认配置
5. **便携模式**: 检测 `portable.json` 标记文件，使用程序所在目录存储配置
6. **预填充功能**: 仅在无任何配置时，TUI 添加配置界面会从 `~/.claude/settings.json` 预填充 Token/BaseURL/Model 并显示提示

## 开发规范

1. **错误处理**: 使用 `fmt.Errorf("context: %w", err)` 包装错误
2. **JSON 序列化**: 使用自定义 Marshal/Unmarshal 保留未知字段
3. **文件操作**: 通过 `internal/utils` 统一处理
4. **多语言支持**: 通过 `internal/i18n` 实现（当前支持中/英）
5. **TUI 开发**: 使用 Bubble Tea 框架，状态机模式

## 测试策略

- **单元测试**: 覆盖 `internal/` 包的核心逻辑
- **集成测试**: 测试完整的配置切换流程
- **跨平台测试**: 在 Windows/Linux/macOS 上验证

## 便携版支持

检测 `portable.json` 文件（与可执行文件同目录）启用便携模式：
- 配置存储在 `./data/.cc-switch/`
- Live 配置存储在 `./data/.claude/` 和 `./data/.codex/`
- 适合 U 盘或绿色部署

- 代码修改完成之后，编译之前都需要对整个项目执行一次 go fmt ./... 命令来格式化代码，然后执行 go test ./... 运行单元测试，最后执行 go build 进行编译测试
