# WSL/Windows 双环境行尾符与空白符问题完全指南

## 问题背景

在 WSL2（Ubuntu）和 Windows 双环境下同时开发时，由于两个系统对文本文件的行尾符（Line Ending）和空白字符处理方式不同，会导致以下问题：

1. **幽灵变更**：`git status` 显示大量文件被修改，但 `git diff` 看不到实质性改动
2. **构建失败**：行尾符不一致导致脚本无法执行（如 Shell 脚本包含 `\r`）
3. **测试异常**：字符串比较失败（如 `"hello\r\n"` vs `"hello\n"`）
4. **代码审查混乱**：PR 中显示大量无意义的空白符变更

## 根本原因分析

### 1. 行尾符差异

| 系统 | 行尾符 | 十六进制 | 别名 |
|------|--------|----------|------|
| Windows | CR+LF | `0D 0A` | `\r\n` |
| Linux/macOS | LF | `0A` | `\n` |
| 旧版 Mac | CR | `0D` | `\r` (已废弃) |

**典型场景**：
- 在 WSL2 中用 Vim/Nano 编辑文件 → 保存为 LF
- 在 Windows 中用 Notepad/VSCode 编辑同一文件 → 自动转换为 CRLF
- Git 检测到文件变化 → 提示 Modified

### 2. Git 的 `core.autocrlf` 配置陷阱

Git 提供 `core.autocrlf` 配置来自动转换行尾符，但在双环境下容易引发问题：

| 配置值 | 行为 | 适用场景 | WSL2 风险 |
|--------|------|----------|----------|
| `true` | Checkout: CRLF → LF<br>Commit: LF → CRLF | Windows 单环境 | ⚠️ 与 WSL2 冲突 |
| `input` | Checkout: 不转换<br>Commit: CRLF → LF | Linux/macOS | ✅ WSL2 推荐 |
| `false` | 完全不转换 | 仓库已有 `.gitattributes` | ✅ 严格模式 |

**问题根源**：
- Windows 侧设置 `core.autocrlf=true`
- WSL2 侧设置 `core.autocrlf=input`
- 同一文件在两侧提交时被反复转换行尾符

### 3. 文件系统边界问题

WSL2 的两种文件系统访问方式：

#### (1) WSL2 原生文件系统 (`/home/user`)
```bash
# Linux 路径
/home/lin/projects/cc-switch-cli/

# 从 Windows 访问（通过网络）
\\wsl$\Ubuntu\home\lin\projects\cc-switch-cli\
```

**特性**：
- ✅ 性能最佳（原生 ext4）
- ✅ 权限完整（Unix 权限位）
- ⚠️ Windows 侧编辑可能触发行尾符转换

#### (2) Windows 文件系统挂载 (`/mnt/d`)
```bash
# Linux 路径
/mnt/d/Projects/cc-switch-cli/

# Windows 路径
D:\Projects\cc-switch-cli\
```

**特性**：
- ⚠️ 性能较差（跨系统调用）
- ⚠️ 权限映射不完整
- ⚠️ 文件元数据（行尾符）由 Windows 主导

## 症状诊断

### 快速检测命令

```bash
# 1. 查看显示变更的文件数量
git status --short | wc -l

# 2. 忽略空白字符后的真实变更
git diff --ignore-all-space --ignore-blank-lines --stat

# 3. 检查特定文件的行尾符
file cmd/root.go
# 输出示例：
# cmd/root.go: UTF-8 Unicode text, with CRLF line terminators  ❌ Windows 格式
# cmd/root.go: UTF-8 Unicode text                              ✅ Unix 格式

# 4. 查看行尾符的十六进制
od -c cmd/root.go | head -n 5
# 如果看到 \r \n 连续出现，说明是 CRLF

# 5. 使用 Git 检查空白符变更
git diff --check
```

### 典型症状示例

**症状 A：幽灵变更**
```bash
$ git status --short
M  cmd/gemini_env.go
M  internal/config/gemini.go
M  main.go
... (共 30 个文件)

$ git diff --ignore-all-space --stat
# 无输出或只有个位数文件

✅ 诊断：纯行尾符差异，无实质变更
```

**症状 B：Shell 脚本执行失败**
```bash
$ ./build.sh
bash: ./build.sh: /bin/bash^M: bad interpreter: No such file or directory

✅ 诊断：脚本包含 CRLF，`^M` 是 `\r` 的可见符号
```

**症状 C：Go 测试失败**
```bash
$ go test ./internal/config
--- FAIL: TestGenerateGeminiEnvExport (0.00s)
    gemini_test.go:25: expected "export FOO=bar\n", got "export FOO=bar\r\n"

✅ 诊断：代码中硬编码了 `\n`，但文件实际包含 `\r\n`
```

## 永久解决方案

### 阶段 1：统一 Git 配置

在 **WSL2** 和 **Windows Git Bash** 中分别执行：

```bash
# WSL2 环境
git config --global core.autocrlf input
git config --global core.eol lf

# Windows Git Bash 环境
git config --global core.autocrlf input
git config --global core.eol lf

# 验证配置
git config --global --list | grep -E 'core\.(autocrlf|eol)'
```

**关键理由**：
- 两侧都用 `input`，提交时统一转换为 LF
- 设置 `core.eol=lf` 确保 checkout 时也用 LF
- 避免双环境配置不一致

### 阶段 2：强制项目规范（`.gitattributes`）

项目根目录的 `.gitattributes` 文件（本项目已配置）：

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

# Markdown 和配置文件使用 LF
*.md text eol=lf
*.json text eol=lf
*.toml text eol=lf
*.yaml text eol=lf
*.yml text eol=lf

# 二进制文件
*.exe binary
*.dll binary
*.so binary
*.png binary
*.jpg binary
*.ico binary
```

**工作原理**：
- `.gitattributes` 的规则优先级 **高于** `core.autocrlf`
- `text eol=lf` 强制该类型文件在工作区使用 LF
- `binary` 标记确保二进制文件不被转换

### 阶段 3：重新规范化现有文件

```bash
# 1. 确保 .gitattributes 已提交
git add .gitattributes
git commit -m "chore: 添加 .gitattributes 统一行尾符"

# 2. 重新规范化所有文件（关键步骤）
git add --renormalize .

# 3. 检查状态（应该只显示真正有变更的文件）
git status --short

# 4. 查看实际变更内容
git diff --cached --stat

# 5. 提交规范化变更
git commit -m "chore: 重新规范化文件行尾符

根据 .gitattributes 规则统一所有文件为 LF 格式"
```

**`git add --renormalize` 详解**：
- 读取 `.gitattributes` 规则
- 重新转换工作区所有文件的行尾符
- 更新 Git 索引（staging area）
- **不会修改工作区文件本身**，只修改 Git 记录的状态

### 阶段 4：配置编辑器/IDE

#### VS Code (`settings.json`)

```json
{
  // 全局默认使用 LF
  "files.eol": "\n",

  // 保存时移除行尾空白符
  "files.trimTrailingWhitespace": true,

  // 文件末尾保留空行
  "files.insertFinalNewline": true,

  // 显示空白字符（调试用）
  "editor.renderWhitespace": "boundary"
}
```

**Remote-WSL 扩展注意事项**：
- 使用 Remote-WSL 连接到 WSL2 时，VS Code 自动使用 WSL2 的文件系统
- 此时编辑器行为由 WSL2 环境决定，不受 Windows 侧配置影响

#### GoLand/IntelliJ IDEA

1. **Settings** → **Editor** → **Code Style**
2. **Line separator**: 选择 `Unix and macOS (\n)`
3. **勾选** "Ensure every saved file ends with a line break"

#### Vim/Neovim

```vim
" ~/.vimrc 或 ~/.config/nvim/init.vim
set fileformat=unix
set fileformats=unix,dos

" 自动移除行尾空白符
autocmd BufWritePre * :%s/\s\+$//e
```

### 阶段 5：批量修复现有文件（可选）

如果发现仓库中仍有混乱的行尾符，使用以下脚本批量修复：

```bash
#!/bin/bash
# 文件名：fix-line-endings.sh

# 查找所有 Go 文件并转换为 LF
find . -type f -name "*.go" -exec dos2unix {} \;

# 查找所有 Shell 脚本并转换为 LF
find . -type f -name "*.sh" -exec dos2unix {} \;

# 查找所有 Markdown 文件并转换为 LF
find . -type f -name "*.md" -exec dos2unix {} \;

echo "行尾符修复完成"
```

**安装 `dos2unix` 工具**：
```bash
# WSL2/Ubuntu
sudo apt-get install dos2unix

# macOS
brew install dos2unix

# Windows (Git Bash)
# 通常已包含在 Git for Windows 中
```

## 双环境开发最佳实践

### ✅ 推荐做法

1. **在 WSL2 原生文件系统中开发**
   ```bash
   # 项目路径
   /home/lin/projects/cc-switch-cli/

   # 从 Windows 访问（仅用于文件管理，不编辑）
   \\wsl$\Ubuntu\home\lin\projects\cc-switch-cli\
   ```

2. **使用 VS Code Remote-WSL 扩展**
   - 在 Windows 中打开 VS Code
   - 连接到 WSL2：`Ctrl+Shift+P` → `Remote-WSL: New Window`
   - 编辑器自动使用 WSL2 环境的配置

3. **所有 Git 操作在 WSL2 中执行**
   ```bash
   # 在 WSL2 终端中
   cd /home/lin/projects/cc-switch-cli/
   git add .
   git commit -m "..."
   git push
   ```

4. **定期验证行尾符一致性**
   ```bash
   # 添加到 Git pre-commit hook
   git diff --check
   ```

### ❌ 避免做法

1. **不要在 Windows 侧直接编辑 WSL2 文件系统的文件**
   - 性能极差（每次操作都跨系统调用）
   - 容易触发行尾符自动转换

2. **不要在两侧使用不同的 `core.autocrlf` 设置**
   ```bash
   # ❌ 错误配置
   # Windows 侧
   git config core.autocrlf true

   # WSL2 侧
   git config core.autocrlf input

   # 结果：文件在两侧反复转换行尾符
   ```

3. **不要依赖编辑器的自动检测**
   - 明确配置 `files.eol="\n"`，不要用 `auto`
   - 避免编辑器根据文件现有行尾符"猜测"

4. **不要在 `/mnt/d` 下进行高频 Git 操作**
   - 性能差（NTFS → WSL2 转换层）
   - 文件元数据由 Windows 主导，容易触发行尾符问题

## 应急修复手册

### 场景 1：突然出现大量幽灵变更

```bash
# 1. 验证是行尾符问题
git diff --ignore-all-space --stat
# 如果输出为空或极少文件，确认是行尾符问题

# 2. 丢弃工作区变更（谨慎操作）
git restore .

# 3. 重新规范化
git add --renormalize .
git commit -m "chore: 重新规范化行尾符"
```

### 场景 2：Shell 脚本无法执行（`^M` 错误）

```bash
# 快速修复单个文件
dos2unix build.sh

# 或使用 sed
sed -i 's/\r$//' build.sh

# 验证修复
file build.sh
# 应输出：build.sh: Bourne-Again shell script, ASCII text executable
```

### 场景 3：Go 测试中字符串比较失败

```go
// 测试代码
expected := "export FOO=bar\n"
actual := generateScript()
if expected != actual {
    t.Fatalf("expected %q, got %q", expected, actual)
}
```

**修复方案**：
```go
// 方案 A：使用 strings.TrimSpace 忽略尾部空白符
expected := "export FOO=bar"
actual := strings.TrimSpace(generateScript())

// 方案 B：跨平台行尾符常量
const eol = "\n"  // 在 Windows 下也使用 LF
expected := fmt.Sprintf("export FOO=bar%s", eol)

// 方案 C：显式检查（推荐用于调试）
if strings.Contains(actual, "\r\n") {
    t.Fatalf("unexpected CRLF in output: %q", actual)
}
```

### 场景 4：PR 中显示大量空白符变更

```bash
# 在提交前检查
git diff --check

# 查看忽略空白符后的 diff
git diff --ignore-all-space

# 如果确认只是空白符问题，修复后重新提交
git add --renormalize .
git commit --amend --no-edit
git push --force-with-lease
```

## 验证配置正确性

运行以下检查清单：

```bash
# ✅ 检查 1：Git 配置统一
echo "=== Git 配置 ==="
git config core.autocrlf
git config core.eol
# 期望输出：
# input
# lf

# ✅ 检查 2：.gitattributes 存在
echo "=== .gitattributes ==="
cat .gitattributes | grep "*.go"
# 期望输出：*.go text eol=lf

# ✅ 检查 3：文件行尾符一致
echo "=== 文件行尾符 ==="
file cmd/*.go | grep CRLF
# 期望输出：空（没有 CRLF）

# ✅ 检查 4：无幽灵变更
echo "=== Git 状态 ==="
git status --short | wc -l
git diff --ignore-all-space --stat
# 期望：两个命令输出的文件数量一致

# ✅ 检查 5：编辑器配置
echo "=== VS Code 配置 ==="
cat ~/.config/Code/User/settings.json | grep "files.eol"
# 期望输出："files.eol": "\n"
```

## 常见问题 FAQ

### Q1: 为什么 Windows 也要用 `core.autocrlf=input`？

A: 虽然 Windows 原生使用 CRLF，但在双环境下：
- `.gitattributes` 已经强制规范为 LF
- `input` 模式确保提交时统一转换
- 避免与 WSL2 侧配置冲突

### Q2: `git add --renormalize` 会修改我的工作区文件吗？

A: **不会**。它只修改 Git 索引（staging area），工作区文件保持不变。你可以通过 `git diff` 查看将要提交的变更。

### Q3: 已经推送到远程仓库的 CRLF 文件如何修复？

A:
1. 本地执行 `git add --renormalize .`
2. 提交并推送
3. 团队成员执行 `git pull` 后自动同步正确的行尾符

### Q4: 为什么 `.gitattributes` 中 `*.ps1` 使用 CRLF？

A: PowerShell 脚本在 Windows 中执行，必须使用 CRLF。但 Go 源文件等应统一用 LF。

### Q5: 在 Windows 下能直接打开 `/mnt/d` 的仓库吗？

A: 可以，但 **不推荐**：
- 性能差（跨文件系统）
- 行尾符容易混乱
- 推荐使用 WSL2 原生文件系统 + Remote-WSL

## 参考资料

- [Git 官方文档 - gitattributes](https://git-scm.com/docs/gitattributes)
- [GitHub - Configuring Git to handle line endings](https://docs.github.com/en/get-started/getting-started-with-git/configuring-git-to-handle-line-endings)
- [Microsoft - Working across Windows and Linux file systems](https://learn.microsoft.com/en-us/windows/wsl/filesystems)
- [dos2unix 工具手册](https://linux.die.net/man/1/dos2unix)

## 总结

WSL/Windows 双环境下的行尾符问题根源在于 **两个系统的默认行为差异** 和 **文件系统边界**。通过以下四步永久解决：

1. **统一 Git 配置**：两侧都用 `core.autocrlf=input` + `core.eol=lf`
2. **强制项目规范**：`.gitattributes` 明确每种文件类型的行尾符
3. **重新规范化**：`git add --renormalize .` 清理历史混乱
4. **配置编辑器**：明确设置 `files.eol="\n"`，不依赖自动检测

遵循这些规范，可以彻底避免幽灵变更、脚本执行失败等问题，确保团队协作的代码一致性。
