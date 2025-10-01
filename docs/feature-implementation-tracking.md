# CC-Switch CLI 功能实现进度跟踪

> 基于原版 GUI (v3.3.1) 的功能对照表
> 最后更新: 2025-10-01

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

### 3. 配置文件操作

#### 原子写入
- [x] 实现原子文件写入（临时文件 + rename）
- [x] 跨平台兼容（Windows 先删除再 rename，Unix 直接覆盖）
- [x] 保留原文件权限（Unix系统）
- [x] 写入前创建备份（.bak文件）

#### 配置迁移 ✅
- [x] 副本文件合并（扫描 `settings.json`、`claude.json`）
- [x] 按名称+API Key去重
- [x] 迁移命令实现 (migrate)
- [x] 模拟运行模式支持 (--dry-run)

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

### 7. 导入导出功能

- [x] 从 live 配置导入（首次启动）
- [x] 导出配置到文件
- [x] 导入配置从文件
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

### 11. 备份恢复功能

- [x] 备份配置到文件（backup）
- [x] 从备份恢复配置（restore）
- [x] 列出所有备份
- [x] 自动清理旧备份
- [x] 备份验证

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

## 测试要求

### 单元测试
- [ ] 配置文件读写测试
- [ ] 原子操作测试
- [ ] 配置验证测试
- [ ] 路径解析测试

### 集成测试
- [ ] 完整切换流程测试
- [ ] 配置迁移测试
- [ ] 回滚机制测试
- [ ] 跨平台兼容性测试

## 注意事项

1. **向后兼容性**: 必须完全兼容原版 GUI 的配置文件格式
2. **数据安全**: 所有写操作必须使用原子操作，避免配置损坏
3. **错误处理**: 提供清晰的错误信息，便于用户理解和解决问题
4. **平台差异**: 注意处理 Windows/macOS/Linux 的路径差异
5. **性能考虑**: 避免频繁的文件 I/O 操作

## 开发进度跟踪

- 项目启动日期: 2025-10-01
- 当前版本: v0.2.0
- 目标版本: v1.0.0（与 GUI v3.3.1 功能对等）

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

### 待实现功能（P2）
- 🔲 便携版支持
- 🔲 多语言支持
- 🔲 自动更新检查
- 🔲 详细调试日志

### 项目里程碑
- 2025-10-01: 项目启动，基础架构搭建
- 2025-10-01 晚: P0 功能全部完成
- 2025-10-02: P1 功能全部完成，CLI 工具达到生产可用状态