# 发布流程文档

本文档描述如何使用 GitHub Actions 自动发布 cc-switch-cli 的新版本。

## 概述

项目已配置 GitHub Actions 自动化发布流程，当推送版本 tag 时会自动：
- 运行测试
- 编译 6 个平台的二进制文件（Linux/macOS/Windows × amd64/arm64）
- 创建 GitHub Release
- 上传构建产物和 checksums

## 首次发布 (v1.0.0)

### 第一步：提交当前修改

```bash
# 1. 查看当前修改
git status

# 2. 添加所有新文件和修改
git add .

# 3. 提交（包含格式化后的代码 + 新增的 GitHub Actions 配置）
git commit -m "chore: Add GitHub Actions release workflow and prepare v1.0.0

- Add .goreleaser.yml for multi-platform builds
- Add .github/workflows/release.yml for automated releases
- Update version to 1.0.0
- Format code with go fmt"

# 4. 推送到 GitHub
git push origin master
```

### 第二步：创建并推送 Tag

```bash
# 1. 创建 v1.0.0 tag（可选添加注释）
git tag -a v1.0.0 -m "Release version 1.0.0

First stable release with multi-app configuration support"

# 或者简单版本（不带注释）
git tag v1.0.0

# 2. 推送 tag 到 GitHub（这会触发 GitHub Actions）
git push origin v1.0.0
```

### 第三步：观察 GitHub Actions 运行

1. 推送 tag 后，访问仓库 Actions 页面：
   ```
   https://github.com/YangQing-Lin/cc-switch-cli/actions
   ```

2. 查看名为 "Release" 的 workflow 运行状态

3. 点击进入查看详细日志：
   - ✅ Checkout code
   - ✅ Set up Go
   - ✅ Run tests
   - ✅ Run GoReleaser (构建 6 个平台)
   - ✅ Upload to GitHub Release

4. 大约 2-5 分钟后，访问 Release 页面：
   ```
   https://github.com/YangQing-Lin/cc-switch-cli/releases
   ```

### 第四步：验证发布结果

检查 Release 页面应该包含：
- ✅ Release 标题：v1.0.0
- ✅ 自动生成的 Release Notes
- ✅ 6 个二进制压缩包（Assets）:
  - `ccs-1.0.0-linux-amd64.tar.gz`
  - `ccs-1.0.0-linux-arm64.tar.gz`
  - `ccs-1.0.0-darwin-amd64.tar.gz`
  - `ccs-1.0.0-darwin-arm64.tar.gz`
  - `ccs-1.0.0-windows-amd64.zip`
  - `ccs-1.0.0-windows-arm64.zip`
- ✅ `checksums.txt`
- ✅ Source code (zip/tar.gz)

## 后续版本发布流程

### 准备新版本

```bash
# 1. 修改版本号
# 编辑 internal/version/version.go
# 将 const Version = "1.0.0" 改为新版本号，例如 "1.1.0"

# 2. 开发新功能/修复 bug
# ... 进行代码修改 ...

# 3. 运行测试和格式化
go test ./...
go fmt ./...

# 4. 提交修改
git add .
git commit -m "feat: Add new awesome feature"
git push origin master
```

### 发布新版本

```bash
# 1. 创建新版本 tag
git tag -a v1.1.0 -m "Release version 1.1.0

- Feature description
- Bug fix description"

# 2. 推送 tag 触发 GitHub Actions
git push origin v1.1.0

# 3. 等待 GitHub Actions 完成构建和发布
```

## 故障排除

### 常见问题

#### 1. 权限问题

**症状**: GitHub Actions 运行失败，提示权限错误

**解决方案**:
1. 进入仓库设置：Settings → Actions → General
2. 找到 "Workflow permissions"
3. 选择 "Read and write permissions"
4. 保存设置

#### 2. 测试失败

**症状**: Actions 在 "Run tests" 步骤失败

**解决方案**:
```bash
# 在本地运行测试查看错误
go test ./... -v

# 修复问题后重新提交
git add .
git commit -m "fix: Fix failing tests"
git push origin master

# 删除失败的 tag 并重建
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0
git tag v1.0.0
git push origin v1.0.0
```

#### 3. GoReleaser 配置错误

**症状**: Actions 在 "Run GoReleaser" 步骤失败

**解决方案**:
1. 查看 Actions 日志中的具体错误信息
2. 修改 `.goreleaser.yml` 配置文件
3. 提交修复并重建 tag：

```bash
# 修复配置
git add .goreleaser.yml
git commit -m "fix: Fix GoReleaser configuration"
git push origin master

# 删除旧 tag
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0

# 重新创建 tag
git tag v1.0.0
git push origin v1.0.0
```

### 删除已发布的 Release

如果需要删除错误的 Release：

```bash
# 1. 在 GitHub Release 页面手动删除 Release

# 2. 删除本地 tag
git tag -d v1.0.0

# 3. 删除远程 tag
git push origin :refs/tags/v1.0.0
```

## Commit Message 规范

为了生成更好的 Release Notes，建议使用以下 commit message 前缀：

- `feat:` - 新功能（会出现在 "Features" 分组）
- `fix:` - Bug 修复（会出现在 "Bug Fixes" 分组）
- `docs:` - 文档修改（不会出现在 Release Notes）
- `test:` - 测试相关（不会出现在 Release Notes）
- `chore:` - 构建/工具相关（不会出现在 Release Notes）
- `refactor:` - 代码重构
- `perf:` - 性能优化

示例：
```bash
git commit -m "feat: Add backup management features"
git commit -m "fix: Fix Windows path handling issue"
git commit -m "docs: Update README with installation instructions"
```

## 版本号规范

遵循语义化版本 (Semantic Versioning)：

- **主版本号 (MAJOR)**: 不兼容的 API 变更 (`v2.0.0`)
- **次版本号 (MINOR)**: 向后兼容的新功能 (`v1.1.0`)
- **修订号 (PATCH)**: 向后兼容的 bug 修复 (`v1.0.1`)

示例：
- `v1.0.0` → `v1.0.1`: Bug 修复
- `v1.0.1` → `v1.1.0`: 新增功能
- `v1.1.0` → `v2.0.0`: 破坏性变更

## 相关文件

- `.goreleaser.yml` - GoReleaser 构建配置
- `.github/workflows/release.yml` - GitHub Actions workflow
- `internal/version/version.go` - 版本号定义
- `CLAUDE.md` - 项目开发规范（包含构建要求）

## 参考资源

- [GoReleaser 文档](https://goreleaser.com/intro/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [语义化版本规范](https://semver.org/lang/zh-CN/)
