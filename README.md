# cc-switch-cli

一个轻量级的命令行工具，用于管理多个Claude API配置并支持快速切换。

## 功能特性

- 🖥️ **交互式 TUI** - 基于 Bubble Tea 的现代化终端用户界面，支持键盘导航和可视化操作
- 🔄 **快速切换** - 通过单个命令在不同的Claude API配置之间切换
- 📝 **配置管理** - 添加、删除和查看多个API配置
- 🔐 **安全存储** - 显示时API令牌会被遮掩，配置文件受权限保护
- 🌍 **跨平台** - 支持Windows、macOS、Linux和其他操作系统
- 💡 **交互式输入** - 支持命令行参数和交互式提示
- 🎨 **用户友好** - 清晰的列表显示和直观的状态指示器
- 🎯 **多应用支持** - 同时管理 Claude Code 和 Codex CLI 配置

## 安装

### 从源码构建

需要Go 1.25.1或更高版本：

```bash
git clone https://github.com/YangQing-Lin/cc-switch-cli.git
cd cc-switch-cli

# Windows
go build -o ccs.exe

# macOS / Linux
go build -o ccs
```

### 下载预编译二进制文件

从[发布页面](https://github.com/YangQing-Lin/cc-switch-cli/releases)下载适合您操作系统的预编译二进制文件。

### 配置环境变量

为了在任何目录下都能使用 `ccs` 命令,需要将其添加到系统环境变量:

#### Windows

**方法1: 通过 PowerShell**
```powershell
# 将二进制文件移动到用户目录
mkdir -Force $env:USERPROFILE\bin
move ccs.exe $env:USERPROFILE\bin\

# 添加到 PATH (当前会话)
$env:Path += ";$env:USERPROFILE\bin"

# 永久添加到 PATH
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\bin", "User")
```

**方法2: 通过系统设置**
1. 将 `ccs.exe` 复制到一个目录,如 `C:\Program Files\ccs\`
2. 右键点击"此电脑" → "属性" → "高级系统设置"
3. 点击"环境变量"
4. 在"用户变量"中找到 `Path`,点击"编辑"
5. 点击"新建",添加 `C:\Program Files\ccs\`
6. 点击"确定"保存

#### macOS

```bash
# 将二进制文件移动到 /usr/local/bin
sudo mv ccs /usr/local/bin/

# 或者移动到用户目录
mkdir -p ~/bin
mv ccs ~/bin/

# 添加到 PATH (如果使用 ~/bin)
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc   # zsh
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc  # bash

# 重新加载配置
source ~/.zshrc  # 或 source ~/.bashrc
```

#### Linux

```bash
# 方法1: 系统级安装 (需要 sudo)
sudo mv ccs /usr/local/bin/
sudo chmod +x /usr/local/bin/ccs

# 方法2: 用户级安装
mkdir -p ~/.local/bin
mv ccs ~/.local/bin/
chmod +x ~/.local/bin/ccs

# 添加到 PATH (如果 ~/.local/bin 不在 PATH 中)
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

验证安装:
```bash
ccs version
```

### 版本更新与重新编译

当项目有新版本发布时,可以通过以下步骤更新:

```bash
# 1. 进入项目目录
cd cc-switch-cli

# 2. 拉取最新代码
git pull origin main

# 3. 重新编译
# Windows
go build -o ccs.exe

# macOS / Linux
go build -o ccs

# 4. 替换旧版本 (如果已配置环境变量)
# Windows (PowerShell)
move -Force ccs.exe $env:USERPROFILE\bin\ccs.exe

# macOS / Linux
sudo mv ccs /usr/local/bin/ccs  # 系统级
# 或
mv ccs ~/.local/bin/ccs  # 用户级

# 5. 验证新版本
ccs version
```

## 使用方法

### 交互式 TUI 界面 (推荐)

启动交互式终端用户界面:

```bash
ccs
# 或明确指定
ccs ui
```

**TUI 功能特性:**

- 📋 **可视化列表** - 清晰展示所有配置，一目了然
- ⌨️ **键盘导航** - 使用方向键选择配置
- ✏️ **即时编辑** - 按 `e` 快速编辑配置
- ➕ **快速添加** - 按 `a` 添加新配置
- 🗑️ **安全删除** - 按 `d` 删除配置(带确认)
- 🔄 **一键切换** - 按 `Enter` 切换到选中的配置
- 🎨 **友好界面** - 美观的色彩和布局设计

**TUI 快捷键:**

| 快捷键 | 功能 |
|--------|------|
| `↑` / `k` | 向上移动光标 |
| `↓` / `j` | 向下移动光标 |
| `Enter` | 切换到选中的配置 |
| `a` | 添加新配置 |
| `e` | 编辑选中的配置 |
| `d` | 删除选中的配置 |
| `t` | 切换应用（Claude/Codex） |
| `c` | 切换到 Claude |
| `x` | 切换到 Codex |
| `r` | 刷新列表 |
| `q` / `Ctrl+C` | 退出 |

在表单编辑模式下:
- `Tab` / `Shift+Tab` / `↑` / `↓` - 切换输入框焦点
- 直接输入 - 编辑当前聚焦的输入框
- `Enter` / `Ctrl+S` - 保存并提交
- `ESC` - 取消并返回

### 命令行模式

#### 列出所有配置

```bash
# 由于默认启动TUI，使用其他命令查看列表
ccs config show
```

#### 切换配置

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

### 添加新配置

#### 方法1：交互模式

```bash
ccs config add my-config
```

程序将提示您输入：
- API令牌（隐藏输入）
- 基础URL
- 类别（可选）

#### 方法2：命令行参数

```bash
ccs config add my-config \
  --apikey "sk-ant-xxxxx" \
  --base-url "https://api.example.com" \
  --category "custom"
```

支持的类别类型：
- `official` - 官方API
- `cn_official` - 官方中国区
- `aggregator` - 聚合服务
- `third_party` - 第三方服务
- `custom` - 自定义（默认）

### 删除配置

```bash
ccs config delete <配置名称>
```

添加`--force`或`-f`标志跳过确认：

```bash
ccs config delete my-config --force
```

### Codex CLI 配置管理 🆕

#### 添加 Codex 配置

```bash
ccs codex add my-codex \
  --apikey "sk-ant-xxxxx" \
  --base-url "https://api.anthropic.com" \
  --model "claude-3-5-sonnet-20241022"
```

#### 列出 Codex 配置

```bash
ccs codex list
```

#### 切换 Codex 配置

```bash
ccs codex switch my-codex
```

输出：
```
✓ 已切换到 Codex 配置: my-codex
  Base URL: https://api.anthropic.com
  API Key: sk-a...***
  Model: claude-3-5-sonnet-20241022

已更新文件:
  - C:\Users\username\.codex\config.yaml
  - C:\Users\username\.codex\api.json
```

#### 更新 Codex 配置

```bash
ccs codex update my-codex \
  --model "claude-opus-4-20250514" \
  --apikey "sk-new-key"
```

#### 删除 Codex 配置

```bash
ccs codex delete my-codex -f
```

**Codex 配置特点：**
- 🔄 **双文件管理** - 自动维护 `config.yaml` 和 `api.json`
- ⚡ **原子操作** - 事务性写入，失败自动回滚
- 🎯 **模型支持** - 可自定义使用的 Claude 模型
- 🛡️ **SSOT 模式** - 与 Rust 后端架构完全一致

### 配置备份与恢复 🆕

#### 导出配置

```bash
# 导出到默认文件（cc-switch-export-<timestamp>.json）
ccs export

# 导出到指定文件
ccs export --output my-config.json

# 导出时过滤特定应用或配置
ccs export --app claude --pretty
```

#### 导入配置（自动备份）

```bash
# 从文件导入（自动创建备份）
ccs import --from-file my-config.json

# 输出示例：
# ✓ 已创建备份: backup_20251006_143528
# ✓ 导入配置: Claude官方-1
# 导入完成: 1 个配置已导入, 0 个配置已跳过
```

**重要特性：**
- ✅ **自动备份** - 导入前自动备份当前配置到 `~/.cc-switch/backups/`
- ✅ **备份格式** - `backup_YYYYMMDD_HHMMSS.json`（与 GUI v3.4.0 一致）
- ✅ **自动清理** - 仅保留最近10个备份，旧备份自动删除

#### 备份管理

```bash
# 手动创建备份
ccs backup

# 列出所有备份
ccs backup list

# 从备份恢复配置
ccs backup restore backup_20251006_143528

# 恢复输出示例：
# ✓ 已创建恢复前备份: backup_20251006_143639_pre-restore.json
# ✓ 配置已从备份恢复: backup_20251006_143528.json
```

**备份特点：**
- 📦 **安全恢复** - 恢复前自动备份当前配置
- 🔍 **格式验证** - 恢复前验证备份文件有效性
- 📊 **详细信息** - 显示备份时间、大小、路径

### 便携版模式 🆕

便携版模式允许您将配置文件存储在程序所在目录，适用于 USB 便携设备或不想污染用户主目录的场景。

#### 启用便携版模式

```bash
# 启用便携版模式（在程序目录创建 portable.ini）
ccs portable enable

# 输出示例：
# ✓ 便携版模式已启用
#   标记文件: D:\Programs\cc-switch\portable.ini
#
# 配置目录: D:\Programs\cc-switch\.cc-switch
```

#### 查看便携版状态

```bash
# 查看当前是否为便携版模式
ccs portable status

# 输出示例：
# 便携版模式状态
# ==============
#
# ✓ 便携版模式：已启用
#
# 程序目录: D:\Programs\cc-switch
# 标记文件: D:\Programs\cc-switch\portable.ini (存在)
#
# 配置文件: D:\Programs\cc-switch\.cc-switch\config.json
# 配置状态: 已存在
```

#### 禁用便携版模式

```bash
# 禁用便携版模式（删除 portable.ini）
ccs portable disable

# 输出示例：
# ✓ 便携版模式已禁用
#   已删除: D:\Programs\cc-switch\portable.ini
#
# 配置文件将使用: C:\Users\username\.cc-switch\config.json
```

**便携版特点：**
- 📁 **独立配置** - 配置文件存储在程序目录的 `.cc-switch` 子目录
- 🔄 **即时切换** - 创建/删除 `portable.ini` 文件即可切换模式
- 💼 **便于携带** - 整个程序目录可以复制到 USB 设备
- 🔄 **迁移友好** - 支持使用导入/导出功能在普通模式和便携版间迁移配置

### Claude 插件集成 🆕

管理 Claude 桌面应用的插件配置（`~/.claude/config.json`），适用于需要使用第三方 API 服务的场景。

#### 查看插件状态

```bash
# 查看 Claude 插件配置状态
ccs claude-plugin status

# 输出示例：
# Claude 插件配置状态
# ===================
#
# 配置文件: C:\Users\username\.claude\config.json
# 文件状态: ✓ 存在
# 配置状态: ✓ 已应用（由 cc-switch 管理）
```

#### 应用插件配置

```bash
# 应用 Claude 插件配置（用于第三方 API）
ccs claude-plugin apply

# 输出示例：
# ✓ Claude 插件配置已应用
#   配置文件: C:\Users\username\.claude\config.json
#
# 说明:
#   - 已写入 primaryApiKey 字段
#   - 第三方 API 服务现在可以正常工作
```

#### 移除插件配置

```bash
# 移除 Claude 插件配置（恢复官方 API）
ccs claude-plugin remove

# 输出示例：
# ✓ Claude 插件配置已移除
#   配置文件: C:\Users\username\.claude\config.json
#
# 说明:
#   - 已删除 primaryApiKey 字段
#   - 其他配置字段已保留
#   - 官方 API 服务现在可以正常工作
```

#### 检查配置状态

```bash
# 检查配置是否已正确应用
ccs claude-plugin check

# 输出示例：
# ✓ Claude 插件配置已正确应用
#   状态: 由 cc-switch 管理
```

**插件集成特点：**
- 🔧 **精准管理** - 只管理 `primaryApiKey` 字段，不影响其他配置
- 🛡️ **安全操作** - 使用原子文件写入，保证配置安全
- 🔄 **灵活切换** - 轻松在官方 API 和第三方 API 间切换
- 📝 **状态可查** - 随时查看当前配置状态

## 配置文件

配置文件位置：

**普通模式：**
- **Windows**: `%USERPROFILE%\.cc-switch\config.json`
- **macOS/Linux**: `~/.cc-switch/config.json`

**便携版模式：**
- **所有平台**: `<程序目录>\.cc-switch\config.json`
- 需要在程序目录下创建 `portable.ini` 文件以启用便携版模式

配置文件格式：
```json
{
  "version": 2,
  "claude": {
    "providers": {
      "uuid-xxx": {
        "id": "uuid-xxx",
        "name": "config-name",
        "settingsConfig": {
          "env": {
            "ANTHROPIC_AUTH_TOKEN": "your-api-token",
            "ANTHROPIC_BASE_URL": "api-endpoint"
          }
        },
        "category": "custom",
        "createdAt": 1234567890
      }
    },
    "current": "active-config-id"
  }
}
```

## 与cc-switch GUI版本的兼容性

cc-switch-cli与[cc-switch](https://github.com/farion1231/cc-switch) GUI版本完全兼容：

- ✅ 共享相同的配置文件格式
- ✅ 支持相同的配置结构
- ✅ 可以互换使用
- ✅ 配置更改实时同步
- ✅ 备份格式完全兼容（v0.5.0 与 GUI v3.4.0 对齐）

您可以同时使用CLI和GUI版本，因为它们读取和写入相同的配置文件。CLI和GUI创建的备份也可以互相恢复。

## 安全注意事项

1. **文件权限** - 配置文件默认为600权限（仅所有者可读/写）
2. **令牌遮掩** - 显示时API令牌会自动遮掩
3. **备份机制** - 导入前自动创建时间戳备份，保留最近10个
4. **输入保护** - 配置期间API令牌输入被隐藏
5. **恢复保护** - 从备份恢复前自动备份当前配置

## 常见问题

### Q: 如何从旧版本配置迁移？

A: cc-switch-cli自动检测并将v1配置文件迁移到v2格式。

### Q: 配置文件损坏怎么办？

A: 您可以从以下备份恢复：
1. 使用 `ccs backup list` 查看所有自动备份
2. 使用 `ccs backup restore <backup-id>` 恢复到指定备份
3. 导入前的自动备份位于 `~/.cc-switch/backups/` 目录
4. 也可以从 `config.json.bak.cli` 手动恢复

### Q: 支持哪些Claude API提供商？

A: 支持所有与Anthropic API格式兼容的服务，包括：
- 官方Claude API
- 各种中继服务
- 本地代理服务

### Q: 如何验证配置是否有效？

A: 添加配置时会执行基本验证（名称、令牌、URL格式）。实际连接测试在使用时进行。

## 开发

### 项目结构

```
cc-switch-cli/
├── main.go                 # 入口点
├── cmd/                    # 命令行界面
│   ├── root.go            # 根命令 (集成TUI)
│   ├── ui.go              # TUI 子命令
│   ├── config.go          # 配置子命令
│   ├── add.go             # 添加配置
│   └── delete.go          # 删除配置
├── internal/              # 内部实现
│   ├── config/           # 配置管理
│   ├── tui/              # TUI 界面 (Bubble Tea)
│   ├── i18n/             # 国际化支持
│   └── utils/            # 实用函数
└── go.mod                # 依赖管理
```

### 技术栈

- **CLI 框架**: [Cobra](https://github.com/spf13/cobra) - 命令行接口
- **TUI 框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) - 终端用户界面
- **TUI 组件**: [Bubbles](https://github.com/charmbracelet/bubbles) - 可复用组件
- **样式美化**: [Lip Gloss](https://github.com/charmbracelet/lipgloss) - 终端样式

### 构建项目

```bash
# 为当前平台构建
go build -o ccs

# 为Windows构建
GOOS=windows GOARCH=amd64 go build -o ccs.exe

# 为macOS构建
GOOS=darwin GOARCH=amd64 go build -o ccs-darwin

# 为Linux构建
GOOS=linux GOARCH=amd64 go build -o ccs-linux
```

### 运行测试

本项目包含完整的单元测试和集成测试：

```bash
# 运行所有测试
go test ./...

# 运行单元测试（带覆盖率）
go test -cover ./internal/...

# 运行集成测试
go test -v ./test/integration/...

# 使用测试脚本
./test.bat           # Windows
./test.sh            # Linux/macOS

# 生成覆盖率报告
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```

#### 测试覆盖率

- **internal/utils**: 69.7% - 文件原子操作、JSON读写
- **internal/settings**: 82.4% - 设置管理、语言切换
- **internal/i18n**: 60.0% - 国际化支持（中英文）
- **internal/vscode**: 25.0% - VS Code/Cursor 集成
- **internal/config**: 32.1% - Provider CRUD、配置管理

#### 集成测试

集成测试验证了多个组件协同工作：

- ✅ Provider CRUD 操作
- ✅ 配置持久化（模拟重启）
- ✅ 多应用支持（Claude/Codex）
- ✅ 配置文件结构验证
- ✅ 并发访问保护
- ✅ 数据完整性验证

查看 [docs/testing.md](docs/testing.md) 了解详细的测试文档。

## 许可证

MIT许可证

## 贡献

欢迎提交问题和拉取请求！

## 相关项目

- [cc-switch](https://github.com/farion1231/cc-switch) - 带有图形界面的GUI版本，用于配置管理