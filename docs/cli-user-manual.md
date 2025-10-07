# CC-Switch CLI 用户手册

> 版本: v0.6.0
> 最后更新: 2025-10-07

## 目录

- [简介](#简介)
- [安装](#安装)
- [快速开始](#快速开始)
- [核心概念](#核心概念)
- [命令详解](#命令详解)
- [配置文件](#配置文件)
- [高级功能](#高级功能)
- [故障排除](#故障排除)
- [最佳实践](#最佳实践)

---

## 简介

CC-Switch CLI 是一个轻量级命令行工具，用于管理 Claude Code 和 Codex CLI 的多个 API 配置并支持快速切换。

### 为什么使用 CC-Switch CLI?

- **快速切换**: 一键切换不同 API 配置
- **多应用支持**: 同时管理 Claude Code 和 Codex CLI
- **配置管理**: 集中存储和管理所有配置
- **GUI兼容**: 与 GUI 版本共享配置文件
- **便携模式**: 支持 USB 便携设备

### 系统要求

- Windows 10+, macOS 10.15+, 或 Linux (任何现代发行版)
- Claude Code 或 Codex CLI (取决于您的使用需求)

---

## 安装

### 方法1: 从 GitHub Releases 下载

1. 访问 [Releases 页面](https://github.com/YangQing-Lin/cc-switch-cli/releases)
2. 下载适合您系统的版本：
   - Windows: `ccs-windows-amd64.exe`
   - macOS: `ccs-darwin-amd64`
   - Linux: `ccs-linux-amd64`

3. 将可执行文件重命名为 `ccs` (或 Windows 上的 `ccs.exe`)

### 方法2: 从源码构建

```bash
# 克隆仓库
git clone https://github.com/YangQing-Lin/cc-switch-cli.git
cd cc-switch-cli

# 构建 (需要 Go 1.25.1+)
go build -o ccs         # macOS/Linux
go build -o ccs.exe     # Windows
```

### 配置环境变量 (推荐)

**Windows (PowerShell):**
```powershell
# 创建目录
mkdir -Force $env:USERPROFILE\bin

# 移动文件
move ccs.exe $env:USERPROFILE\bin\

# 添加到 PATH (永久)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\bin", "User")
```

**macOS:**
```bash
# 移动到系统目录
sudo mv ccs /usr/local/bin/
sudo chmod +x /usr/local/bin/ccs

# 验证安装
ccs version
```

**Linux:**
```bash
# 用户级安装
mkdir -p ~/.local/bin
mv ccs ~/.local/bin/
chmod +x ~/.local/bin/ccs

# 添加到 PATH (如果需要)
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

---

## 快速开始

### 第一次使用

1. **检查当前状态**
   ```bash
   ccs check
   ```

2. **添加您的第一个配置**
   ```bash
   # Claude Code 配置
   ccs config add my-claude \
     --apikey "sk-ant-xxxxx" \
     --base-url "https://api.anthropic.com"

   # Codex CLI 配置
   ccs codex add my-codex \
     --apikey "sk-ant-xxxxx" \
     --model "claude-3-5-sonnet-20241022"
   ```

3. **切换配置**
   ```bash
   ccs my-claude    # 切换到 Claude 配置
   ccs cx switch my-codex  # 切换到 Codex 配置
   ```

### 使用 TUI 界面 (推荐新手)

```bash
# 启动交互式界面
ccs
```

**TUI 快捷键**:
- `↑`/`↓` 或 `k`/`j`: 移动光标
- `Enter`: 切换到选中的配置
- `a`: 添加新配置
- `e`: 编辑配置
- `d`: 删除配置
- `t`: 切换应用 (Claude/Codex)
- `q`: 退出

---

## 核心概念

### 配置文件结构

CC-Switch CLI 使用单一配置文件存储所有配置（SSOT 原则）：

```
~/.cc-switch/
├── config-cli.json     # 主配置文件
├── settings.json       # 应用设置
└── backups/            # 自动备份目录
    ├── backup_20251007_120000.json
    └── backup_20251007_123000.json
```

### 三步切换流程 (SSOT)

1. **回填 (Backfill)**: 从当前 live 配置读取最新修改
2. **切换 (Switch)**: 更新当前激活的配置 ID
3. **持久化 (Persist)**: 将新配置写入目标应用

### Provider 数据结构

```json
{
  "id": "uuid-xxx",
  "name": "配置名称",
  "settingsConfig": {
    "env": {
      "ANTHROPIC_AUTH_TOKEN": "sk-ant-xxx",
      "ANTHROPIC_BASE_URL": "https://api.example.com"
    }
  },
  "category": "third_party",
  "createdAt": 1728300000000,
  "meta": {
    "custom_endpoints": {
      "https://api.example.com": {
        "url": "https://api.example.com",
        "addedAt": 1728300000000,
        "lastUsed": 1728310000000
      }
    }
  }
}
```

**字段说明**:
- `id`: 唯一标识符 (UUID)
- `name`: 用户可读名称
- `settingsConfig`: 完整的配置 JSON
- `category`: 分类 (official/third_party/custom)
- `createdAt`: 创建时间戳 (毫秒)
- `meta`: 元数据 (v0.6.0+, 可选)
  - `custom_endpoints`: 自定义端点列表 (与 GUI v3.5.0 兼容)

---

## 命令详解

### 1. 配置管理 (config)

#### config add - 添加新配置

```bash
# 交互式添加
ccs config add my-config

# 命令行参数
ccs config add my-config \
  --apikey "sk-ant-xxxxx" \
  --base-url "https://api.anthropic.com" \
  --category "third_party"
```

**支持的分类**:
- `official` - Anthropic 官方 API
- `cn_official` - 官方中国区
- `aggregator` - API 聚合服务
- `third_party` - 第三方中继服务
- `custom` - 自定义 (默认)

#### config delete - 删除配置

```bash
# 交互式确认
ccs config delete my-config

# 跳过确认
ccs config delete my-config --force
```

**注意**: 无法删除当前激活的配置

#### config show - 查看配置详情

```bash
# 查看特定配置
ccs config show my-config

# 查看所有配置
ccs config show
```

#### config update - 更新配置

```bash
# 更新 API Key
ccs config update my-config --apikey "new-key"

# 更新 Base URL
ccs config update my-config --base-url "https://new-api.com"

# 更新多个字段
ccs config update my-config \
  --name "new-name" \
  --apikey "new-key" \
  --base-url "https://new-api.com"
```

### 2. Codex 管理 (codex/cx)

#### codex add - 添加 Codex 配置

```bash
# 完整命令
ccs codex add my-codex \
  --apikey "sk-ant-xxxxx" \
  --base-url "https://api.anthropic.com" \
  --model "claude-3-5-sonnet-20241022"

# 简化命令 (推荐)
ccs cx add my-codex \
  --apikey "sk-ant-xxxxx" \
  --model "claude-3-5-sonnet-20241022"
```

**默认值**:
- `base-url`: `https://api.anthropic.com`
- `model`: `claude-3-5-sonnet-20241022`

#### codex list - 列出 Codex 配置

```bash
ccs cx list
```

输出示例:
```
Codex CLI 配置列表
==================

名称: my-codex [当前使用]
  UUID: d1234567-8901-2345-6789-012345678901
  Base URL: https://api.anthropic.com
  API Key: sk-a...***
  Model: claude-3-5-sonnet-20241022

名称: backup-codex
  UUID: e2345678-9012-3456-7890-123456789012
  Base URL: https://api.example.com
  API Key: sk-b...***
  Model: claude-opus-4-20250514
```

#### codex switch - 切换 Codex 配置

```bash
ccs cx switch my-codex
```

**自动更新的文件**:
- `~/.codex/config.yaml`
- `~/.codex/api.json`

#### codex update - 更新 Codex 配置

```bash
# 更新模型
ccs cx update my-codex --model "claude-opus-4-20250514"

# 更新 API Key
ccs cx update my-codex --apikey "new-key"

# 更新多个字段
ccs cx update my-codex \
  --model "claude-opus-4-20250514" \
  --apikey "new-key" \
  --base-url "https://new-api.com"
```

#### codex delete - 删除 Codex 配置

```bash
ccs cx delete my-codex -f
```

### 3. 备份恢复 (backup)

#### backup - 手动创建备份

```bash
ccs backup
```

输出示例:
```
✓ 配置已备份到: C:\Users\username\.cc-switch\backups\backup_20251007_143528.json
```

#### backup list - 列出所有备份

```bash
ccs backup list
```

输出示例:
```
配置备份列表
============

时间: 2025-10-07 14:35:28
文件: backup_20251007_143528.json
大小: 2.45 KB
路径: C:\Users\username\.cc-switch\backups\backup_20251007_143528.json

时间: 2025-10-06 09:12:45
文件: backup_20251006_091245.json
大小: 2.41 KB
路径: C:\Users\username\.cc-switch\backups\backup_20251006_091245.json

总计: 2 个备份文件
```

#### backup restore - 从备份恢复

```bash
ccs backup restore backup_20251007_143528
```

输出示例:
```
✓ 已创建恢复前备份: backup_20251007_143639_pre-restore.json
✓ 配置已从备份恢复: backup_20251007_143528.json
```

**注意**: 恢复前会自动备份当前配置

### 4. 导入导出 (import/export)

#### export - 导出配置

```bash
# 导出到默认文件
ccs export

# 导出到指定文件
ccs export --output my-config.json

# 导出特定应用的配置
ccs export --app claude --output claude-only.json

# 美化输出 (便于阅读)
ccs export --pretty
```

#### import - 导入配置

```bash
# 从文件导入 (自动创建备份)
ccs import --from-file my-config.json
```

输出示例:
```
✓ 已创建备份: backup_20251007_143528
✓ 导入配置: Claude官方-1
✓ 导入配置: PackyCode
导入完成: 2 个配置已导入, 0 个配置已跳过
```

**重要特性**:
- 导入前自动备份当前配置
- 备份格式: `backup_YYYYMMDD_HHMMSS.json`
- 自动清理旧备份 (保留最近10个)

### 5. Claude 插件集成 (claude-plugin)

#### claude-plugin status - 查看插件状态

```bash
ccs claude-plugin status
```

输出示例:
```
Claude 插件配置状态
===================

配置文件: C:\Users\username\.claude\config.json
文件状态: ✓ 存在
配置状态: ✓ 已应用（由 cc-switch 管理）
```

#### claude-plugin apply - 应用插件配置

```bash
ccs claude-plugin apply
```

**用途**: 启用第三方 API 支持

输出示例:
```
✓ Claude 插件配置已应用
  配置文件: C:\Users\username\.claude\config.json

说明:
  - 已写入 primaryApiKey 字段
  - 第三方 API 服务现在可以正常工作
```

#### claude-plugin remove - 移除插件配置

```bash
ccs claude-plugin remove
```

**用途**: 恢复使用官方 API

输出示例:
```
✓ Claude 插件配置已移除
  配置文件: C:\Users\username\.claude\config.json

说明:
  - 已删除 primaryApiKey 字段
  - 其他配置字段已保留
  - 官方 API 服务现在可以正常工作
```

#### claude-plugin check - 检查配置状态

```bash
ccs claude-plugin check
```

### 6. 便携版管理 (portable/port/p)

#### portable status - 查看便携版状态

```bash
# 以下命令等效
ccs portable
ccs portable status
ccs port
ccs p
```

输出示例:
```
便携版模式状态
==============

✓ 便携版模式：已启用

程序目录: D:\Programs\cc-switch
标记文件: D:\Programs\cc-switch\portable.ini (存在)

配置文件: D:\Programs\cc-switch\.cc-switch\config-cli.json
配置状态: 已存在
```

#### portable enable - 启用便携版

```bash
# 以下命令等效
ccs portable enable
ccs portable on
ccs port on
ccs p on
```

**效果**:
- 在程序目录创建 `portable.ini`
- 配置文件位于 `<程序目录>/.cc-switch/config-cli.json`

#### portable disable - 禁用便携版

```bash
# 以下命令等效
ccs portable disable
ccs portable off
ccs port off
ccs p off
```

**效果**:
- 删除 `portable.ini`
- 配置文件恢复到 `~/.cc-switch/config-cli.json`

### 7. 设置管理 (settings)

#### settings - 查看所有设置

```bash
ccs settings
```

输出示例:
```
应用设置
========

语言: zh (中文)
配置目录: 默认 (C:\Users\username\.cc-switch)
```

#### settings --get - 获取特定设置

```bash
ccs settings --get language
```

#### settings --set - 修改设置

```bash
# 设置语言
ccs settings --set language=en

# 设置自定义配置目录
ccs settings --set configDir=D:\MyConfigs\.cc-switch
```

**支持的设置项**:
- `language`: 界面语言 (`en` 或 `zh`)
- `configDir`: 自定义配置目录路径

### 8. 系统命令

#### version - 查看版本信息

```bash
ccs version
```

输出示例:
```
cc-switch-cli v0.6.0
构建日期: 2025-10-07
Git提交: a1b2c3d

项目地址: https://github.com/YangQing-Lin/cc-switch-cli
```

#### check - 检查系统状态

```bash
ccs check
```

输出示例:
```
系统状态检查
============

✓ 配置文件: 已找到
  路径: C:\Users\username\.cc-switch\config-cli.json

✓ Claude Code: 已安装
  配置: C:\Users\username\.claude\settings.json

✓ Codex CLI: 已安装
  配置: C:\Users\username\.codex\config.yaml

✓ 当前激活:
  Claude: official-claude (官方配置)
  Codex: my-codex (自定义配置)
```

#### check-updates - 检查更新

```bash
ccs check-updates
```

**功能**: 打开 GitHub Releases 页面

#### validate - 验证配置文件

```bash
ccs validate
```

输出示例:
```
配置验证
========

✓ 配置文件格式正确
✓ 版本: 2
✓ Claude 配置: 5 个供应商
✓ Codex 配置: 2 个供应商
✓ 所有配置验证通过
```

#### config-dir - 管理配置目录

```bash
# 显示当前配置目录
ccs config-dir
```

输出示例:
```
配置目录信息
============

当前配置目录: C:\Users\username\.cc-switch
配置文件: C:\Users\username\.cc-switch\config-cli.json
设置文件: C:\Users\username\.cc-switch\settings.json

提示:
  要更改配置目录，请使用：
  ccs settings --set configDir=<新路径>
```

#### open-config - 打开配置文件夹

```bash
ccs open-config
```

**功能**: 在系统文件管理器中打开配置目录

---

## 配置文件

### 主配置文件 (config-cli.json)

**位置**:
- 普通模式: `~/.cc-switch/config-cli.json`
- 便携版: `<程序目录>/.cc-switch/config-cli.json`

**格式**:
```json
{
  "version": 2,
  "claude": {
    "providers": {
      "provider-id-1": {
        "id": "provider-id-1",
        "name": "Claude官方",
        "settingsConfig": {
          "env": {
            "ANTHROPIC_AUTH_TOKEN": "sk-ant-xxx",
            "ANTHROPIC_BASE_URL": "https://api.anthropic.com"
          }
        },
        "category": "official",
        "createdAt": 1728300000000
      }
    },
    "current": "provider-id-1"
  },
  "codex": {
    "providers": {
      "codex-id-1": {
        "id": "codex-id-1",
        "name": "Codex配置",
        "settingsConfig": {
          "config": {
            "base_url": "https://api.anthropic.com",
            "api_key": "sk-ant-xxx",
            "model_name": "claude-3-5-sonnet-20241022"
          }
        },
        "category": "custom",
        "createdAt": 1728310000000,
        "meta": {
          "custom_endpoints": {
            "https://api.example.com": {
              "url": "https://api.example.com",
              "addedAt": 1728310000000,
              "lastUsed": 1728320000000
            }
          }
        }
      }
    },
    "current": "codex-id-1"
  }
}
```

### 应用设置文件 (settings.json)

**位置**: `~/.cc-switch/settings.json`

**格式**:
```json
{
  "language": "zh",
  "configDir": ""
}
```

**字段说明**:
- `language`: 界面语言 (`en` 或 `zh`)
- `configDir`: 自定义配置目录 (空字符串表示使用默认)

### Meta 字段 (v0.6.0+)

**用途**: 存储与 GUI v3.5.0+ 兼容的元数据

**示例**:
```json
{
  "meta": {
    "custom_endpoints": {
      "https://api.packycode.com": {
        "url": "https://api.packycode.com",
        "addedAt": 1728307200000,
        "lastUsed": 1728310800000
      },
      "https://api-hk-cn2.packycode.com": {
        "url": "https://api-hk-cn2.packycode.com",
        "addedAt": 1728307300000
      }
    }
  }
}
```

**注意**:
- CLI 不提供端点管理命令
- 用户可手动编辑 `config-cli.json` 添加自定义端点
- 端点测速功能请使用 GUI 版本

---

## 高级功能

### 1. 自定义配置目录

```bash
# 设置自定义目录
ccs settings --set configDir=/path/to/custom/dir

# 验证
ccs config-dir

# 恢复默认
ccs settings --set configDir=
```

**应用场景**:
- 多用户共享配置
- 网络驱动器同步
- 云同步配置

### 2. GUI 与 CLI 协同使用

CC-Switch CLI 与 GUI 完全兼容，可以同时使用：

```bash
# CLI 创建配置
ccs config add test-config --apikey "xxx"

# GUI 可以立即看到并使用该配置
# GUI 修改配置后，CLI 也能立即看到变化
```

**注意事项**:
- GUI 使用 `config.json`，CLI 使用 `config-cli.json`
- 两者数据结构完全相同
- 备份可以互相恢复

### 3. 配置迁移

**从旧版本迁移**:
```bash
# 自动检测并升级 v1 配置
ccs migrate
```

**在设备间迁移**:
```bash
# 设备 A: 导出配置
ccs export --output my-config.json

# 将 my-config.json 复制到设备 B

# 设备 B: 导入配置
ccs import --from-file my-config.json
```

### 4. 批量操作

**批量添加配置**:
```bash
#!/bin/bash
# add-multiple.sh

configs=(
  "config1:sk-ant-key1:https://api1.com"
  "config2:sk-ant-key2:https://api2.com"
  "config3:sk-ant-key3:https://api3.com"
)

for config in "${configs[@]}"; do
  IFS=':' read -r name key url <<< "$config"
  ccs config add "$name" --apikey "$key" --base-url "$url"
done
```

### 5. 脚本集成

**在 CI/CD 中使用**:
```bash
#!/bin/bash
# ci-switch.sh

# 切换到测试环境配置
ccs test-env

# 运行测试
npm test

# 切换回生产环境
ccs prod-env
```

**在 NPM Scripts 中使用**:
```json
{
  "scripts": {
    "dev": "ccs dev-config && npm run start",
    "prod": "ccs prod-config && npm run build"
  }
}
```

---

## 故障排除

### 常见问题

#### Q1: 配置文件不存在

**症状**: 运行命令时提示 "配置文件不存在"

**解决方法**:
```bash
# 检查配置目录
ccs config-dir

# 如果目录不存在，添加第一个配置会自动创建
ccs config add first-config
```

#### Q2: 无法切换配置

**症状**: 切换配置后应用未生效

**解决方法**:
```bash
# 1. 检查当前状态
ccs check

# 2. 验证配置格式
ccs validate

# 3. 查看详细日志
ccs --verbose switch my-config

# 4. 手动验证 live 配置文件
# Claude: ~/.claude/settings.json
# Codex: ~/.codex/config.yaml
```

#### Q3: 备份恢复失败

**症状**: 恢复备份时提示格式错误

**解决方法**:
```bash
# 1. 列出所有备份
ccs backup list

# 2. 验证备份文件
cat ~/.cc-switch/backups/backup_xxx.json | jq .

# 3. 如果备份损坏，使用 .bak.cli 文件
cp ~/.cc-switch/config-cli.json.bak.cli ~/.cc-switch/config-cli.json
```

#### Q4: Claude 插件配置冲突

**症状**: 第三方 API 无法工作

**解决方法**:
```bash
# 1. 检查插件状态
ccs claude-plugin status

# 2. 如果未应用，手动应用
ccs claude-plugin apply

# 3. 如果已应用但不工作，尝试移除后重新应用
ccs claude-plugin remove
ccs claude-plugin apply
```

#### Q5: Codex 双文件不同步

**症状**: `config.yaml` 和 `api.json` 内容不一致

**解决方法**:
```bash
# 切换配置会自动同步两个文件
ccs cx switch my-codex

# 如果仍有问题，手动验证文件
cat ~/.codex/config.yaml
cat ~/.codex/api.json
```

### 调试技巧

#### 启用详细日志

```bash
# 大部分命令支持 --verbose 参数
ccs --verbose switch my-config
ccs --verbose check
ccs --verbose validate
```

#### 查看原始配置文件

```bash
# macOS/Linux
cat ~/.cc-switch/config-cli.json | jq .

# Windows (PowerShell)
Get-Content $env:USERPROFILE\.cc-switch\config-cli.json | ConvertFrom-Json
```

#### 重置配置

```bash
# 备份当前配置
ccs backup

# 删除配置文件（小心！）
rm ~/.cc-switch/config-cli.json

# 重新添加配置
ccs config add first-config
```

---

## 最佳实践

### 1. 配置命名规范

```bash
# 推荐：语义化命名
ccs config add prod-claude-official
ccs config add dev-packycode
ccs codex add test-codex

# 不推荐：无意义命名
ccs config add config1
ccs config add test
```

### 2. 定期备份

```bash
# 在重要操作前手动备份
ccs backup

# 或使用定时任务自动备份 (Linux/macOS)
# 添加到 crontab
0 0 * * * /usr/local/bin/ccs backup
```

### 3. 分类管理

```bash
# 使用 category 字段分类
ccs config add official-api --category official
ccs config add relay-api --category third_party
ccs config add my-proxy --category custom
```

### 4. 环境隔离

```bash
# 开发环境
ccs config add dev-config \
  --apikey "dev-key" \
  --base-url "https://dev-api.com"

# 测试环境
ccs config add test-config \
  --apikey "test-key" \
  --base-url "https://test-api.com"

# 生产环境
ccs config add prod-config \
  --apikey "prod-key" \
  --base-url "https://api.anthropic.com"
```

### 5. 安全建议

- ✅ **使用环境变量**: 不要在脚本中硬编码 API Key
- ✅ **定期备份**: 使用 `ccs backup` 定期备份配置
- ✅ **权限控制**: 确保配置文件权限为 600 (仅所有者可读写)
- ✅ **版本控制**: 不要将 `config-cli.json` 提交到 git (已在 `.gitignore` 中)
- ✅ **审计日志**: 使用 `--verbose` 参数记录操作日志

### 6. 性能优化

```bash
# 使用 TUI 而非反复调用命令行
ccs  # 启动 TUI，在界面中操作

# 批量操作使用脚本
# 而非逐个执行命令
```

### 7. 团队协作

```bash
# 1. 导出团队模板（不含敏感信息）
ccs export --output team-template.json

# 2. 团队成员导入模板
ccs import --from-file team-template.json

# 3. 各自修改 API Key
ccs config update official-config --apikey "your-key"
```

---

## 附录

### A. 完整命令列表

```bash
# 核心命令
ccs                          # 启动 TUI
ccs <config-name>            # 切换配置
ccs ui                       # 启动 TUI (显式)

# 配置管理
ccs config add <name>        # 添加配置
ccs config delete <name>     # 删除配置
ccs config show [name]       # 查看配置
ccs config update <name>     # 更新配置

# Codex 管理
ccs codex add <name>         # 添加 Codex 配置
ccs codex list               # 列出 Codex 配置
ccs codex switch <name>      # 切换 Codex 配置
ccs codex update <name>      # 更新 Codex 配置
ccs codex delete <name>      # 删除 Codex 配置
ccs cx <subcommand>          # codex 简化命令

# 备份恢复
ccs backup                   # 手动备份
ccs backup list              # 列出备份
ccs backup restore <id>      # 恢复备份

# 导入导出
ccs export                   # 导出配置
ccs import --from-file <f>   # 导入配置

# Claude 插件
ccs claude-plugin status     # 查看状态
ccs claude-plugin apply      # 应用配置
ccs claude-plugin remove     # 移除配置
ccs claude-plugin check      # 检查状态
ccs plugin <subcommand>      # 简化命令

# 便携版
ccs portable                 # 查看状态
ccs portable enable          # 启用便携版
ccs portable disable         # 禁用便携版
ccs port <subcommand>        # 简化命令
ccs p <subcommand>           # 最短命令

# 设置管理
ccs settings                 # 查看所有设置
ccs settings --get <key>     # 获取设置
ccs settings --set key=val   # 修改设置

# 系统命令
ccs version                  # 版本信息
ccs check                    # 系统检查
ccs check-updates            # 检查更新
ccs validate                 # 验证配置
ccs config-dir               # 配置目录信息
ccs open-config              # 打开配置文件夹

# 全局参数
--dir <path>                 # 指定配置目录
--verbose                    # 详细输出
--help, -h                   # 帮助信息
```

### B. 配置文件路径

| 文件 | 普通模式 | 便携版模式 |
|------|---------|-----------|
| 主配置 | `~/.cc-switch/config-cli.json` | `<程序目录>/.cc-switch/config-cli.json` |
| 应用设置 | `~/.cc-switch/settings.json` | `<程序目录>/.cc-switch/settings.json` |
| 备份目录 | `~/.cc-switch/backups/` | `<程序目录>/.cc-switch/backups/` |
| Claude 配置 | `~/.claude/settings.json` | 同左 |
| Codex 配置 | `~/.codex/config.yaml` | 同左 |
| 便携版标记 | 不存在 | `<程序目录>/portable.ini` |

### C. 错误代码

| 代码 | 含义 | 解决方法 |
|-----|------|---------|
| 0 | 成功 | - |
| 1 | 一般错误 | 查看错误信息 |
| 2 | 配置文件不存在 | 使用 `ccs config add` 创建 |
| 3 | 配置格式错误 | 使用 `ccs validate` 检查 |
| 4 | 权限错误 | 检查文件权限 |
| 5 | 配置不存在 | 使用 `ccs config show` 查看可用配置 |

### D. 版本历史

| 版本 | 日期 | 主要变更 |
|------|------|---------|
| v0.6.0 | 2025-10-07 | GUI v3.5.0 配置兼容性，添加 Provider.Meta 字段 |
| v0.5.0 | 2025-10-06 | 导入前自动备份，与 GUI v3.4.0 对齐 |
| v0.4.0 | 2025-10-06 | Codex CLI 完整支持，TUI 多应用切换 |
| v0.3.1 | 2025-10-02 | 配置格式兼容性改进 |
| v0.3.0 | 2025-10-02 | 国际化支持，设置管理，版本控制 |
| v0.2.0 | 2025-10-01 | VS Code 集成，配置迁移 |
| v0.1.0 | 2025-10-01 | 初始版本，基础 CRUD 功能 |

---

## 获取帮助

- **GitHub Issues**: https://github.com/YangQing-Lin/cc-switch-cli/issues
- **GUI 版本**: https://github.com/farion1231/cc-switch
- **文档**: `docs/` 目录

---

**感谢使用 CC-Switch CLI！**
