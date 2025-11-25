# cc-switch-cli

一个轻量级的命令行工具，用于管理多个 Claude API 配置并支持快速切换。

## 功能特性

- 🖥️ **交互式 TUI** - 基于 Bubble Tea 的现代化终端用户界面
- 🔄 **快速切换** - 通过单个命令在不同的 Claude API 配置之间切换
- 📝 **配置管理** - 添加、删除和查看多个 API 配置
- 🌍 **跨平台** - 支持 Windows、macOS、Linux
- 🎯 **多应用支持** - 同时管理 Claude Code 和 Codex CLI 配置
- 💼 **便携模式** - 支持配置存储在程序目录，便于 USB 携带

## 编译

需要 Go 1.25.1 或更高版本：

```bash
# 克隆项目
git clone https://github.com/YangQing-Lin/cc-switch-cli.git
cd cc-switch-cli

# Windows
go build -o ccs.exe

# macOS / Linux
go build -o ccs
```

## 使用方法

### 交互式 TUI 界面（推荐）

启动交互式终端用户界面：

```bash
ccs
# 或
ccs ui
```

**TUI 功能特性：**
- 📋 **可视化列表** - 清晰展示所有配置
- ⌨️ **键盘导航** - 使用方向键选择配置
- ✏️ **即时编辑** - 按 `e` 快速编辑配置
- ➕ **快速添加** - 按 `a` 添加新配置
- 🗑️ **安全删除** - 按 `d` 删除配置（带确认）
- 🔄 **一键切换** - 按 `Enter` 切换到选中的配置

**TUI 快捷键：**

| 快捷键 | 功能 |
|--------|------|
| `↑` / `k` | 向上移动光标 |
| `↓` / `j` | 向下移动光标 |
| `Enter` | 切换到选中的配置 |
| `a` | 添加新配置 |
| `e` | 编辑选中的配置 |
| `d` | 删除选中的配置 |
| `t` | 切换应用（Claude/Codex） |
| `r` | 刷新列表 |
| `q` / `Ctrl+C` | 退出 |

在表单编辑模式下：
- `Tab` / `Shift+Tab` / `↑` / `↓` - 切换输入框焦点
- 直接输入 - 编辑当前聚焦的输入框
- `Enter` / `Ctrl+S` - 保存并提交
- `ESC` - 取消并返回

### 命令行模式

#### 快速切换配置

```bash
ccs <配置名称>
```

示例：
```bash
ccs 88code
```

输出：
```
✓ Switched to configuration: 88code
  Token: 88_e7...***
  URL: https://www.example.org/api
```

### 便携版模式

便携版模式允许将配置文件存储在程序所在目录，适用于 USB 便携设备。

#### 查看便携版状态

```bash
ccs portable        # 或 ccs port / ccs p
```

#### 启用便携版模式

```bash
ccs portable on     # 或 ccs port on / ccs p on
```

在程序目录创建 `portable.ini` 标记文件，配置将存储在 `<程序目录>/.cc-switch/config.json`。

#### 禁用便携版模式

```bash
ccs portable off    # 或 ccs port off / ccs p off
```

删除 `portable.ini` 标记文件，配置将使用用户主目录 `~/.cc-switch/config.json`。

## 配置文件

**普通模式：**
- **Windows**: `%USERPROFILE%\.cc-switch\config.json`
- **macOS/Linux**: `~/.cc-switch/config.json`

**便携版模式：**
- **所有平台**: `<程序目录>\.cc-switch\config.json`

## 开发

### 项目结构

```
cc-switch-cli/
├── main.go                 # 入口点
├── cmd/                    # 命令行界面
├── internal/               # 内部实现
│   ├── config/            # 配置管理
│   ├── tui/               # TUI 界面
│   ├── i18n/              # 国际化
│   └── utils/             # 工具函数
└── go.mod                 # 依赖管理
```

### 技术栈

- **CLI 框架**: [Cobra](https://github.com/spf13/cobra)
- **TUI 框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **TUI 组件**: [Bubbles](https://github.com/charmbracelet/bubbles)
- **样式美化**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行单元测试（带覆盖率）
go test -cover ./internal/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```

## 与 GUI 版本的兼容性

cc-switch-cli 与 [cc-switch](https://github.com/farion1231/cc-switch) GUI 版本完全兼容：
- ✅ 共享相同的配置文件格式
- ✅ 可以互换使用
- ✅ 配置更改实时同步

## 许可证

MIT 许可证

## 相关项目

- [cc-switch](https://github.com/farion1231/cc-switch) - 带有图形界面的 GUI 版本
