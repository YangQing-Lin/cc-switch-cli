# Git 行尾符问题排查与修复

## 问题现象

在 WSL2 开发环境中执行 `git status` 时，显示大量文件处于 Modified 状态（例如 51 个文件），但执行 `git diff` 查看具体变更内容时，却发现没有实质性的代码改动。

```bash
$ git status --short | wc -l
51

$ git diff --stat
 go.mod                       |  5 ++---
 go.sum                       | 13 -------------
 internal/template/manager.go |  3 ++-
 3 files changed, 4 insertions(+), 17 deletions(-)
```

## 问题原因

这是典型的 **行尾符（Line Ending）不一致问题**，常见于跨平台开发场景：

### 行尾符差异

- **Windows**: 使用 CRLF (`\r\n`) 作为行尾符
- **Linux/macOS**: 使用 LF (`\n`) 作为行尾符

### 触发场景

1. 在 WSL2（Linux 环境）中开发，但从 Windows 侧编辑过文件
2. Git 配置 `core.autocrlf` 在不同环境下设置不一致
3. 项目中同时存在 LF 和 CRLF 的文件
4. 没有 `.gitattributes` 文件来统一规范行尾符

### 验证方法

```bash
# 检查 Git 配置
$ git config core.autocrlf
input

# 忽略空白字符差异后的 diff（应该只显示真正的代码变更）
$ git diff --ignore-all-space --ignore-blank-lines --stat
 go.mod                       |  5 ++---
 go.sum                       | 13 -------------
 internal/template/manager.go |  3 ++-
 3 files changed, 4 insertions(+), 17 deletions(-)
```

如果忽略空白字符后的变更数量大幅减少，证明是行尾符问题。

## 解决方案

### 步骤 1: 创建 `.gitattributes` 文件

在项目根目录创建 `.gitattributes` 文件，强制规范化行尾符：

```gitattributes
# 自动检测文本文件并规范化
* text=auto

# Go 源文件强制使用 LF
*.go text eol=lf
*.mod text eol=lf
*.sum text eol=lf

# Shell 脚本使用 LF
*.sh text eol=lf

# Windows 脚本使用 CRLF
*.bat text eol=crlf
*.ps1 text eol=crlf

# 二进制文件
*.exe binary
*.dll binary
*.so binary
```

### 步骤 2: 重新规范化所有文件

```bash
# 根据新的 .gitattributes 规则重新规范化所有文件
git add --renormalize .

# 检查状态（现在应该只显示真正有变更的文件）
git status --short
```

### 步骤 3: 提交变更

```bash
git add .gitattributes
git commit -m "chore: 添加 .gitattributes 规范化行尾符

- 统一 Go 源文件使用 LF
- 修复行尾符不一致导致的幽灵变更"
```

## 预防措施

### 1. Git 配置建议

```bash
# Linux/WSL2 环境（推荐）
git config --global core.autocrlf input

# Windows 环境（推荐）
git config --global core.autocrlf true

# macOS 环境（推荐）
git config --global core.autocrlf input
```

### 2. 编辑器配置

确保你的编辑器/IDE 使用统一的行尾符：

**VS Code** (`settings.json`):
```json
{
  "files.eol": "\n"
}
```

**GoLand/IntelliJ IDEA**:
- Settings → Editor → Code Style → Line separator → Unix and macOS (\n)

### 3. 在 WSL2 中开发的注意事项

- **避免从 Windows 侧直接编辑 WSL2 文件系统中的文件**（性能差且容易引起行尾符问题）
- 推荐使用 Remote-WSL 扩展（VS Code）或在 WSL2 内直接运行编辑器
- 如果必须从 Windows 访问，使用 `\\wsl$\Ubuntu\` 路径

## 常见问题

### Q: 为什么 `git diff` 显示空，但 `git status` 显示已修改？

A: Git 检测到文件的元数据（如行尾符）发生变化，但内容本质没变。使用 `git diff --ignore-all-space` 可以验证。

### Q: 已经提交了混乱的行尾符，如何修复历史记录？

A: 不推荐修改历史记录（会破坏协作）。建议：
1. 添加 `.gitattributes`
2. 执行 `git add --renormalize .`
3. 创建一个专门的 commit 说明这是行尾符规范化

### Q: `git add --renormalize` 做了什么？

A: 它会：
1. 读取 `.gitattributes` 的规则
2. 将工作区的所有文件按照规则重新转换行尾符
3. 更新 Git 索引（staging area）
4. 不会修改工作区文件，只修改 Git 记录的状态

## 本次修复的实际变更

经过修复后，本项目的真实变更只有 3 个文件：

1. **go.mod** - 依赖项顺序调整（从 indirect 移到 direct）
2. **go.sum** - 移除未使用的依赖校验和
3. **internal/template/manager.go** - 修复 `embed.FS` 路径分隔符问题（必须使用 `/` 而非 `filepath.Join`）

原先显示的 51 个文件变更是行尾符不一致导致的"幽灵变更"。

## 参考资料

- [Git 官方文档 - gitattributes](https://git-scm.com/docs/gitattributes)
- [GitHub 帮助 - Configuring Git to handle line endings](https://docs.github.com/en/get-started/getting-started-with-git/configuring-git-to-handle-line-endings)
- [Go embed.FS 路径要求](https://pkg.go.dev/embed#hdr-Directives) - 必须使用正斜杠 `/`
