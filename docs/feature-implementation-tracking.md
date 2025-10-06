# CC-Switch CLI 功能实现进度跟踪

> 基于原版 GUI (v3.3.1) 的功能对照表
> 最后更新: 2025-10-02

## 核心架构要求

### 配置文件兼容性
- [x] 使用相同的配置文件路径 `~/.cc-switch/config.json`
- [x] 兼容 v2 版本配置格式
- [ ] 支持从 v1 自动升级到 v2
- [x] 使用相同的数据结构（Provider、ProviderManager、MultiAppConfig）

### 单一数据源（SSOT）原则
- [x] 配置集中存储于 `~/.cc-switch/config.json`
- [x] 切换时直接写入目标工具的实际配置文件
- [x] 实现三步切换流程：回填(Backfill) → 切换(Switch) → 持久化(Persist)

## 功能模块对照表

### 1. 供应商管理 (Provider Management)

#### 基础 CRUD 操作
- [x] 列出所有供应商 (list)
- [x] 添加供应商 (add)
- [x] 更新供应商 (update)
- [x] 删除供应商 (delete)
- [x] 查看供应商详情 (show)

#### 切换功能
- [x] 基础切换功能 (switch)
- [x] 切换前回填当前 live 配置
- [x] 切换后更新 current 标记
- [x] 切换失败回滚机制

### 2. 多应用支持

#### Claude 支持
- [x] Claude 配置管理
- [x] 写入 `~/.claude/settings.json`
- [x] 兼容旧版 `~/.claude/claude.json`
- [x] Claude 配置验证（必须包含 env.ANTHROPIC_AUTH_TOKEN）

#### Codex 支持 ✅
- [x] Codex 配置管理
- [x] 原子写入 `~/.codex/config.yaml` + `~/.codex/api.json`
- [x] 双文件事务机制（失败自动回滚）
- [x] YAML 语法校验
- [x] Codex 配置验证（必须包含 api_key 字段）
- [x] Codex 专用命令组（codex add/list/switch/delete/update）
- [x] Codex 模型参数支持（model_name）
- [x] TUI 多应用支持（Claude/Codex 切换）

### 3. 配置文件操作

#### 原子写入 ✅
- [x] 实现原子文件写入（临时文件 + rename）
- [x] 跨平台兼容（Windows 先删除再 rename，Unix 直接覆盖）
- [x] 保留原文件权限（Unix系统）
- [x] 写入前创建备份（CLI 专用 .bak.cli 文件）

#### 配置迁移 ✅
- [x] 副本文件合并（扫描 `settings.json`、`claude.json`）
- [x] 按名称+API Key去重
- [x] 迁移命令实现 (migrate)
- [x] 模拟运行模式支持 (--dry-run)
- [x] 旧格式自动迁移（检测带"apps"键的v2旧格式）
- [x] 迁移时归档旧配置到 `archive/config.v2-old.backup.<timestamp>.json`

### 4. 配置目录管理

#### 路径解析 ✅
- [x] 支持自定义配置目录 (--dir 参数)
- [x] 波浪号（~）展开
- [x] 优先级：自定义目录 > 默认目录

#### 目录操作 ✅
- [x] 创建配置目录（如不存在）
- [x] 打开配置文件夹（系统文件管理器）
- [x] 显示当前配置路径

### 5. VS Code 集成 ✅

- [x] 自动检测 VS Code settings.json 路径
- [x] 检测运行中的 VS Code/Cursor 进程
- [x] 支持多个 VS Code 版本（VS Code、Cursor）
- [x] check 命令显示集成状态

### 6. 应用设置 ✅

- [x] 读取应用设置 `~/.cc-switch/settings.json`
- [x] 保存应用设置
- [x] 语言切换支持（en/zh）
- [x] 自定义配置目录设置

### 7. 导入导出功能 ✅

- [x] 从 live 配置导入（首次启动）
- [x] 导出配置到文件
- [x] 导入配置从文件
- [x] 导入前自动备份（匹配 GUI v3.4.0）
- [x] 配置验证

### 8. 安全性功能 ✅

- [x] API Token 掩码显示
- [x] 配置文件权限设置（600）
- [x] 敏感信息保护

### 9. 便携版支持

- [ ] 检测 portable.ini 文件
- [ ] 便携版模式（配置存储在程序目录）

### 10. 其他功能

- [x] 版本信息显示
- [x] 检查更新（打开 GitHub Releases）
- [x] 配置状态检查
- [x] 详细错误信息输出
- [ ] 调试模式日志

### 11. 备份恢复功能 ✅

#### 备份机制
- [x] 常规备份（每次保存前自动创建 .bak.cli）
- [x] 手动备份（backup 命令）
- [x] 导入前自动备份（`backup_YYYYMMDD_HHMMSS.json` 格式，匹配 GUI v3.4.0）
- [x] 列出所有备份（backup list 命令）
- [x] 自动清理旧备份（保留最近10个，匹配 GUI）
- [x] 备份验证

#### 归档机制
- [x] v2旧格式迁移归档（`archive/config.v2-old.backup.<timestamp>.json`）
- [x] 归档目录结构：`~/.cc-switch/archive/`
- [x] 带时间戳的归档文件名

#### 恢复机制
- [x] 从备份恢复配置（backup restore 命令）
- [x] 从指定文件恢复
- [x] 恢复前验证配置格式
- [x] 恢复前自动备份当前配置（pre-restore 备份）

#### 与 GUI v3.4.0 的一致性
- ✅ 导入前自动备份（backup_YYYYMMDD_HHMMSS.json）
- ✅ 备份目录：`~/.cc-switch/backups/`
- ✅ 自动清理旧备份（保留最近10个）
- ✅ 备份格式完全兼容
- CLI 额外使用 `.bak.cli` 后缀（用于常规保存时的备份）
- CLI 归档到 `archive/config.v2-old.backup.<timestamp>.json`
- GUI 归档到 `archive/<timestamp>/<category>/` 结构

### 12. Claude 插件集成 🆕

> 基于 GUI v3.3.1+ 新增功能

- [ ] Claude 插件配置管理（`~/.claude/config.json`）
- [ ] 检测 Claude 插件配置状态
- [ ] 读取 Claude 插件配置内容
- [ ] 应用配置到 Claude 插件（写入固定 JSON）
- [ ] 移除 Claude 插件配置（清除特定字段）
- [ ] 检测配置是否已应用（验证 `primaryApiKey: "any"`）
- [ ] 切换供应商时自动同步 Claude 插件
- [ ] 第三方供应商自动应用配置
- [ ] 官方供应商自动移除配置

## 命令行接口设计

### 主命令
```bash
cc-switch [command] [flags]
```

### 子命令规划
```bash
# 供应商管理
cc-switch list [--app claude|codex]          # 列出供应商
cc-switch add <name> [--app] [--api-key]     # 添加供应商
cc-switch update <id|name> [--name] [--api-key] # 更新供应商
cc-switch delete <id|name> [--force]         # 删除供应商
cc-switch show <id|name>                     # 显示详情
cc-switch switch <id|name>                   # 切换供应商

# 配置管理
cc-switch import [--from-live]               # 导入配置
cc-switch export [--to file]                 # 导出配置
cc-switch backup                             # 备份配置
cc-switch restore [--from file]              # 恢复配置

# 目录管理
cc-switch config-dir [--set path]            # 管理配置目录
cc-switch open-config                        # 打开配置文件夹

# VS Code 集成
cc-switch vscode [--sync]                    # VS Code 同步

# 系统命令
cc-switch settings [--get|--set key=value]   # 管理设置
cc-switch migrate                            # 执行迁移
cc-switch validate                           # 验证配置
cc-switch version                            # 版本信息
cc-switch update                             # 检查更新

# Claude 插件集成 (新增)
cc-switch claude-plugin status               # 检查 Claude 插件配置状态
cc-switch claude-plugin apply                # 应用配置到 Claude 插件
cc-switch claude-plugin remove               # 移除 Claude 插件配置
cc-switch claude-plugin check                # 检测配置是否已应用

# Codex CLI 管理 (v0.4.0 新增) ✅
cc-switch codex add <name> [--apikey] [--base-url] [--model]  # 添加 Codex 配置
cc-switch codex list                         # 列出所有 Codex 配置
cc-switch codex switch <name>                # 切换 Codex 配置
cc-switch codex update <name> [--apikey] [--base-url] [--model] # 更新 Codex 配置
cc-switch codex delete <name> [-f]           # 删除 Codex 配置
```

## 数据结构对照

### Provider 结构
```go
type Provider struct {
    ID             string          `json:"id"`
    Name           string          `json:"name"`
    SettingsConfig json.RawMessage `json:"settingsConfig"`
    WebsiteURL     *string         `json:"websiteUrl,omitempty"`
    Category       *string         `json:"category,omitempty"`
    CreatedAt      *int64          `json:"createdAt,omitempty"`
}
```

### ProviderManager 结构
```go
type ProviderManager struct {
    Providers map[string]Provider `json:"providers"`
    Current   string              `json:"current"`
}
```

### MultiAppConfig 结构
```go
type MultiAppConfig struct {
    Version int                        `json:"version"`
    Apps    map[string]ProviderManager `json:"apps"`
}
```

## 实现优先级

### P0 - 核心功能（必须实现）
1. 使用原版配置文件路径和格式
2. 基础供应商切换功能
3. Claude 支持
4. 原子文件写入
5. 配置验证

### P1 - 重要功能（应该实现）
1. Codex 支持
2. 配置迁移
3. VS Code 集成
4. 自定义配置目录
5. 导入导出功能

### P2 - 增强功能（可以实现）
1. 便携版支持
2. 多语言支持
3. 自动更新检查
4. 详细调试日志

### P3 - 扩展功能（GUI 新增）
1. Claude 插件集成（基于 GUI v3.3.1+ 新功能）
2. 供应商预设支持 `apiKeyUrl` 字段

## 测试要求

### 单元测试
- [x] 配置文件读写测试 (internal/utils)
- [x] 原子操作测试 (internal/utils)
- [x] 配置验证测试 (internal/settings, internal/i18n)
- [x] 路径解析测试 (internal/vscode)
- [x] 文件工具函数测试 (internal/utils)
- [x] 设置管理测试 (internal/settings)
- [x] 国际化测试 (internal/i18n)
- [x] VS Code集成测试 (internal/vscode)
- [x] Provider CRUD 测试 (internal/config)

### 测试覆盖率
- internal/utils: 69.7%
- internal/settings: 82.4%
- internal/i18n: 60.0%
- internal/config: 32.1% ⭐ 新增
- internal/vscode: 25.0%
- **平均覆盖率**: 53.8%

### 集成测试
- [x] 基本 Provider 操作测试 (test/integration/basic_test.go)
- [x] Provider 持久化测试
- [x] 多应用支持测试
- [x] 配置文件结构测试
- [x] Provider 验证测试
- [x] 并发访问测试

## 注意事项

1. **向后兼容性**: 必须完全兼容原版 GUI 的配置文件格式
2. **数据安全**: 所有写操作必须使用原子操作，避免配置损坏
3. **错误处理**: 提供清晰的错误信息，便于用户理解和解决问题
4. **平台差异**: 注意处理 Windows/macOS/Linux 的路径差异
5. **性能考虑**: 避免频繁的文件 I/O 操作

## 开发进度跟踪

- 项目启动日期: 2025-10-01
- 当前版本: v0.3.0
- 目标版本: v1.0.0（与 GUI v3.3.1+ 功能对等）
- GUI 参考版本: v3.3.1+ (含 Claude 插件同步功能)

### 已完成功能

#### P0 核心功能（100% 完成）✅
- ✅ 完整的 SSOT 架构实现
- ✅ Claude 配置管理和切换
- ✅ 供应商 CRUD 操作
- ✅ 导入导出功能
- ✅ 配置验证 (validate)
- ✅ 备份恢复功能
- ✅ 原子文件操作和回滚机制

#### P1 重要功能（100% 完成）✅
- ✅ Codex 支持（双文件事务）
- ✅ VS Code/Cursor 集成检测
- ✅ 配置迁移和去重
- ✅ 自定义配置目录支持

#### P2 增强功能（部分完成）
- ✅ 多语言支持（i18n 框架和中英文翻译）
- ✅ 自动更新检查（打开 GitHub Releases）
- ✅ 版本信息显示（支持构建时注入）
- ✅ 应用设置管理（settings 命令）
- ✅ 配置目录管理（config-dir 和 open-config 命令）
- ✅ 配置文件权限管理（敏感文件 0600）
- 🔲 便携版支持（portable.ini 检测）
- 🔲 详细调试日志（--debug 模式）

#### P3 扩展功能（GUI v3.3.1+ 新增）
- 🔲 Claude 插件配置管理（`~/.claude/config.json`）
- 🔲 Claude 插件状态检测
- 🔲 自动同步 Claude 插件配置
- 🔲 供应商预设 `apiKeyUrl` 字段支持

### 项目里程碑
- 2025-10-01: 项目启动，基础架构搭建
- 2025-10-01 晚: P0 功能全部完成
- 2025-10-02 早: P1 功能全部完成
- 2025-10-02 午: P2 大部分功能完成（多语言、设置管理、版本控制等）
- 2025-10-02: GUI 更新至 v3.3.1+，新增 Claude 插件同步功能（待实现）
- 2025-10-06: Codex CLI 完整支持实现（v0.4.0）
- 2025-10-06: 导入前自动备份功能实现，与 GUI v3.4.0 备份机制对齐（v0.5.0）

### 最新完成的功能（v0.3.0）

#### 配置目录管理
- ✅ `config-dir` 命令：显示当前配置目录路径，提供设置自定义目录的指导
- ✅ `open-config` 命令：在系统文件管理器中打开配置文件夹（跨平台支持）
- ✅ 自动创建配置目录（如不存在）

#### 应用设置系统
- ✅ `settings` 命令：管理应用设置（读取/显示/设置）
- ✅ 支持的设置项：
  - `language`: 界面语言（en/zh）
  - `configDir`: 自定义配置目录路径
- ✅ 设置持久化到 `~/.cc-switch/settings.json`（权限 0600）
- ✅ 设置验证和错误处理

#### 国际化支持
- ✅ i18n 框架实现（`internal/i18n` 包）
- ✅ 支持中文（zh）和英文（en）双语
- ✅ 从应用设置自动加载语言偏好
- ✅ 消息翻译函数 `T(key, args...)`

#### 版本管理
- ✅ `version` 命令：显示版本信息
- ✅ 支持构建时注入：
  - 构建日期（BuildDate）
  - Git 提交哈希（GitCommit）
- ✅ 显示项目地址链接

#### 更新检查
- ✅ `check-updates` 命令：打开 GitHub Releases 页面检查更新
- ✅ 跨平台浏览器启动支持（Windows/macOS/Linux）
- ✅ 显示当前版本号对比

#### 安全增强
- ✅ 配置文件权限管理：
  - 敏感文件（config.json, settings.json）使用 0600 权限
  - 一般文件使用 0644 权限
- ✅ 修复原子文件写入权限逻辑
- ✅ 保留原文件权限（Unix 系统）

#### 代码优化
- ✅ 创建 `cmd/common.go` 提供 `getManager()` 辅助函数
- ✅ 统一处理 `--dir` 全局参数
- ✅ 改进错误处理和用户提示

#### 配置格式兼容性改进（v0.3.1）
- ✅ 实现与 GUI 完全兼容的 v2 配置格式（扁平化结构）
- ✅ 自定义 JSON 序列化/反序列化（MarshalJSON/UnmarshalJSON）
- ✅ 自动检测并迁移旧格式（带"apps"键）到新格式
- ✅ 迁移时创建归档备份到 `archive/config.v2-old.backup.<timestamp>.json`
- ✅ CLI 专用备份文件使用 `.bak.cli` 后缀（避免与 GUI 冲突）
- ✅ 完整测试覆盖（迁移测试、格式兼容性测试、备份测试）

### 最新完成的功能（v0.4.0）- Codex CLI 完整支持 🎉

#### Codex 命令组实现
- ✅ `codex add` 命令：添加 Codex CLI 配置
  - 支持 API Key、Base URL、Model 参数
  - 交互式输入缺失参数
  - 自动生成双文件配置（config.yaml + api.json）
- ✅ `codex list` 命令：列出所有 Codex 配置
  - 显示配置详情（API Key、Base URL、Model）
  - 当前激活配置标记
- ✅ `codex switch` 命令：切换 Codex 配置
  - SSOT 三步流程（回填 → 切换 → 持久化）
  - 双文件原子写入（config.yaml + api.json）
  - 失败自动回滚机制
- ✅ `codex update` 命令：更新 Codex 配置
  - 支持更新 API Key、Base URL、Model
  - 交互式输入保留原值选项
  - 立即应用到 live 配置（如果是当前激活）
- ✅ `codex delete` 命令：删除 Codex 配置
  - 确认提示（可用 -f 跳过）
  - 防止删除当前激活配置

#### TUI 多应用支持
- ✅ 应用类型切换：
  - 按 `t` 键：在 Claude 和 Codex 之间切换
  - 按 `c` 键：直接切换到 Claude
  - 按 `x` 键：直接切换到 Codex
- ✅ 动态标题显示：
  - "CC Switch - Claude Code 配置管理"
  - "CC Switch - Codex CLI 配置管理"
- ✅ 应用选择界面：
  - 可视化应用选择（app_select 模式）
  - 键盘导航选择应用
- ✅ 统一操作体验：
  - 所有 CRUD 操作支持 Claude/Codex
  - 添加、编辑、删除、切换自动适配当前应用
  - 帮助文本更新显示新快捷键

#### Codex 配置结构
```json
{
  "config": {
    "base_url": "https://api.anthropic.com",
    "api_key": "sk-xxx",
    "model_name": "claude-3-5-sonnet-20241022"
  },
  "api": {
    "baseURL": "https://api.anthropic.com",
    "apiKey": "sk-xxx"
  }
}
```

#### 技术实现亮点
- ✅ 双文件事务机制：config.yaml + api.json 原子写入
- ✅ 失败自动回滚：任一文件写入失败则恢复原状态
- ✅ Model 参数支持：可自定义模型名称
- ✅ 完整 SSOT 模式：与 Rust 后端架构保持一致
- ✅ TUI 状态管理：currentApp 字段追踪当前应用类型

### 最新完成的功能（v0.5.0）- 导入前自动备份 🎉

#### 与 GUI v3.4.0 备份功能对齐

根据 GUI 项目 v3.4.0 新增的配置备份、导入与导出功能，CLI 已实现对应功能：

1. **导入前自动备份**
   - ✅ 导入配置时自动创建备份
   - ✅ 备份命名格式：`backup_YYYYMMDD_HHMMSS.json`（与 GUI 一致）
   - ✅ 备份位置：`~/.cc-switch/backups/`（与 GUI 一致）
   - ✅ 自动清理旧备份（保留最近10个，与 GUI 一致）

2. **备份列表命令**
   - ✅ `backup list` - 列出所有备份文件
   - ✅ 显示备份时间、大小、路径
   - ✅ 按时间倒序排列（最新的在前）

3. **备份恢复命令**
   - ✅ `backup restore <backup-id>` - 从备份恢复配置
   - ✅ 恢复前自动备份当前配置（pre-restore）
   - ✅ 验证备份文件格式

4. **导入命令增强**
   - ✅ `import --from-file` 导入前自动创建备份
   - ✅ 显示备份ID给用户
   - ✅ 备份失败时给出警告但不中断导入

#### 命令示例

```bash
# 导出配置
cc-switch export --output my-config.json

# 导入配置（自动创建备份）
cc-switch import --from-file my-config.json
# 输出: ✓ 已创建备份: backup_20251006_143528

# 列出所有备份
cc-switch backup list

# 从备份恢复
cc-switch backup restore backup_20251006_143528
```

#### 技术实现细节
- ✅ 使用UTC时间戳确保跨时区一致性
- ✅ 备份清理使用文件修改时间排序
- ✅ 恢复前验证备份文件JSON格式
- ✅ 所有备份操作具有原子性（失败不影响现有配置）

### 代码质量
- ✅ 所有新功能已通过编译测试
- ✅ 所有新功能已通过功能测试
- ✅ 跨平台兼容性验证（Windows/macOS/Linux）
- ✅ 文件权限正确设置和验证
- ✅ 配置文件与 GUI 项目完全兼容

## GUI 新增功能追踪（v3.3.1+）

### Claude 插件同步功能详解

GUI 项目在最新更新中新增了 Claude 插件配置同步功能，具体实现如下：

#### 核心功能
1. **配置文件管理**: 管理 `~/.claude/config.json` 文件
2. **固定配置写入**: 第三方供应商时写入 `{"primaryApiKey": "any"}`
3. **配置移除**: 官方供应商时移除 `primaryApiKey` 字段
4. **状态检测**: 检测配置是否已应用（验证 `primaryApiKey: "any"`）
5. **自动同步**: 切换供应商时自动同步配置

#### 文件变更
- `src-tauri/src/claude_plugin.rs`: 新增 Rust 模块，103 行代码
- `src-tauri/src/commands.rs`: 新增 4 个 Tauri 命令
- `src-tauri/src/lib.rs`: 注册新命令
- `src/App.tsx`: 实现自动同步逻辑
- `src/components/ProviderList.tsx`: 新增 UI 按钮和交互
- `src/lib/tauri-api.ts`: 新增 API 封装
- `src/i18n/locales/{en,zh}.json`: 新增国际化文本

#### CLI 实现建议
为了在 CLI 中实现相同功能，需要：

1. **新建模块**: `internal/claude/plugin.go`
   - `GetClaudeConfigPath()` - 获取配置文件路径
   - `ReadClaudeConfig()` - 读取配置内容
   - `WriteClaudeConfig()` - 写入固定配置
   - `ClearClaudeConfig()` - 移除特定字段
   - `IsClaudeConfigApplied()` - 检测配置状态

2. **新增命令**: `cmd/claude_plugin.go`
   - `claude-plugin status` - 显示配置状态
   - `claude-plugin apply` - 应用配置
   - `claude-plugin remove` - 移除配置
   - `claude-plugin check` - 检测是否已应用

3. **集成到切换流程**: 在 `switch` 命令中自动调用
   - 第三方供应商 → 自动应用配置
   - 官方供应商 → 自动移除配置

4. **配置结构**:
   ```json
   {
     "primaryApiKey": "any"
   }
   ```

#### 实现优先级
- 优先级: P3（扩展功能）
- 依赖: 需要 JSON 文件读写和合并能力
- 难度: 中等（类似 VS Code 集成）
- 预计工作量: 2-3 小时

#### 注意事项
1. 需要保留配置文件中的其他字段
2. 移除配置时只删除 `primaryApiKey` 字段，不是删除整个文件
3. 配置文件可能不存在，需要先创建目录
4. 权限设置建议 0600（与其他配置文件一致）