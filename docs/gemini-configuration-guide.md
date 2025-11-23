# Gemini 配置指南 / Gemini Configuration Guide

[中文](#中文) | [English](#english)

---

## 中文

### 概述

本文档详细介绍了 CC-Switch 项目中 Gemini CLI 的完整配置方法，包括配置文件结构、认证模式、MCP 服务器管理以及供应商切换机制。

### 目录

1. [配置文件结构](#1-配置文件结构)
2. [认证模式](#2-认证模式)
3. [环境变量配置](#3-环境变量配置)
4. [预设供应商](#4-预设供应商)
5. [MCP 服务器配置](#5-mcp-服务器配置)
6. [配置流程](#6-配置流程)
7. [代码架构](#7-代码架构)
8. [常见问题](#8-常见问题)

---

### 1. 配置文件结构

Gemini CLI 使用双文件配置系统：

#### 1.1 配置文件位置

```
~/.gemini/
├── .env                    # API 密钥和环境变量
└── settings.json           # 认证模式和 MCP 服务器配置
```

#### 1.2 `.env` 文件

用于存储 API 密钥和自定义环境变量。

**重要说明**：
- 📍 **配置位置**：`~/.gemini/.env` 文件（**不是系统环境变量**）
- 🔄 **自动加载**：Gemini CLI 会自动读取此文件，无需手动 export
- 🔒 **安全隔离**：配置仅作用于 Gemini CLI，不影响其他程序

**Gemini CLI 的 .env 文件加载顺序**：
1. 当前工作目录（从执行命令的位置向上查找）
2. `~/.gemini/.env`（**推荐位置，CC-Switch 使用此位置**）
3. `~/.env`（备选位置）

**格式**：
```bash
# API 密钥（必需，除非使用 OAuth）
GEMINI_API_KEY=your-api-key-here

# 自定义 API 端点（可选）
GOOGLE_GEMINI_BASE_URL=https://api.example.com

# 模型名称（可选）
GEMINI_MODEL=gemini-3-pro-preview
```

**解析规则**：
- 支持空行和 `#` 注释
- 格式：`KEY=VALUE`
- Key 只能包含字母、数字和下划线
- 文件权限自动设置为 `600`（仅所有者可读写）

**与系统环境变量的区别**：
- ❌ **不是** 在 `~/.bashrc` 或 `~/.zshrc` 中使用 `export GEMINI_API_KEY=...`
- ✅ **而是** 在 `~/.gemini/.env` 文件中直接写入 `GEMINI_API_KEY=...`
- ✅ 优势：不污染系统环境变量，配置独立于 shell 会话

#### 1.3 `settings.json` 文件

存储认证模式和 MCP 服务器配置。

**基本结构**：
```json
{
  "security": {
    "auth": {
      "selectedType": "gemini-api-key"  // 或 "oauth-personal"
    }
  },
  "mcpServers": {
    "server-id": {
      "command": "node",
      "args": ["/path/to/server.js"],
      "env": {}
    }
  }
}
```

---

### 2. 认证模式

Gemini CLI 支持两种认证模式：

#### 2.1 API Key 认证

**适用场景**：PackyCode 等第三方供应商、自定义端点

**配置**：
- `.env` 文件：包含 `GEMINI_API_KEY` 和其他环境变量
- `settings.json`：`security.auth.selectedType = "gemini-api-key"`

**示例配置**：
```json
{
  "env": {
    "GEMINI_API_KEY": "sk-xxxxx",
    "GOOGLE_GEMINI_BASE_URL": "https://www.packyapi.com",
    "GEMINI_MODEL": "gemini-3-pro-preview"
  }
}
```

#### 2.2 OAuth 认证

**适用场景**：Google 官方 Gemini API

**配置**：
- `.env` 文件：为空（不需要 API Key）
- `settings.json`：`security.auth.selectedType = "oauth-personal"`

**示例配置**：
```json
{
  "env": {}
}
```

---

### 3. 环境变量配置

#### 3.1 标准环境变量

| 变量名 | 必需 | 说明 | 示例 |
|--------|------|------|------|
| `GEMINI_API_KEY` | 是* | API 密钥 | `sk-xxxxx` |
| `GOOGLE_GEMINI_API_KEY` | 是* | API 密钥（备选名称） | `sk-xxxxx` |
| `GOOGLE_GEMINI_BASE_URL` | 否 | 自定义 API 端点 | `https://api.example.com` |
| `GEMINI_MODEL` | 否 | 默认模型名称 | `gemini-3-pro-preview` |

*OAuth 模式下不需要 API Key

#### 3.2 配置验证

**基本验证**（创建供应商时）：
- 验证 `env` 字段是否为对象
- 验证 `config` 字段是否为对象或 null
- 不强制要求 `GEMINI_API_KEY`（允许稍后填写）

**严格验证**（切换供应商时）：
- 对于 API Key 模式，必须包含 `GEMINI_API_KEY`
- 对于 OAuth 模式，允许空的 `env` 对象

---

### 4. 预设供应商

CC-Switch 提供三个预设供应商配置：

#### 4.1 Google Official

**认证方式**：OAuth
**特点**：
- 使用 Google 官方 OAuth 认证
- 无需 API Key
- 直接连接到 Google AI Studio

**配置**：
```typescript
{
  name: "Google Official",
  websiteUrl: "https://ai.google.dev/",
  apiKeyUrl: "https://aistudio.google.com/apikey",
  settingsConfig: {
    env: {}
  },
  category: "official",
  partnerPromotionKey: "google-official"
}
```

#### 4.2 PackyCode

**认证方式**：API Key
**特点**：
- 官方合作伙伴
- 提供 API 中继服务
- 专属优惠码：`cc-switch`

**配置**：
```typescript
{
  name: "PackyCode",
  websiteUrl: "https://www.packyapi.com",
  settingsConfig: {
    env: {
      GOOGLE_GEMINI_BASE_URL: "https://www.packyapi.com",
      GEMINI_MODEL: "gemini-3-pro-preview"
    }
  },
  category: "third_party",
  isPartner: true,
  partnerPromotionKey: "packycode"
}
```

#### 4.3 自定义

**认证方式**：API Key
**特点**：
- 允许用户自定义 API 端点
- 完全控制配置

**配置**：
```typescript
{
  name: "自定义",
  websiteUrl: "",
  settingsConfig: {
    env: {
      GOOGLE_GEMINI_BASE_URL: "",
      GEMINI_MODEL: "gemini-3-pro-preview"
    }
  },
  category: "custom"
}
```

---

### 5. MCP 服务器配置

#### 5.1 配置位置

MCP 服务器配置存储在 `~/.gemini/settings.json` 的 `mcpServers` 字段中。

#### 5.2 支持的传输类型

**stdio**（标准输入输出）：
```json
{
  "server-name": {
    "command": "node",
    "args": ["/path/to/server.js"],
    "env": {
      "ENV_VAR": "value"
    }
  }
}
```

**HTTP**（HTTP Streaming）：
```json
{
  "server-name": {
    "httpUrl": "http://localhost:8080/mcp"
  }
}
```

**SSE**（Server-Sent Events）：
```json
{
  "server-name": {
    "url": "http://localhost:8080/sse"
  }
}
```

#### 5.3 格式转换规则

CC-Switch 内部使用统一的 MCP 配置格式，在写入 Gemini CLI 时会进行以下转换：

1. **移除 `type` 字段**：Gemini CLI 从字段名推断传输类型
2. **HTTP 传输转换**：`url` → `httpUrl`
3. **移除 UI 辅助字段**：`enabled`、`source`、`id`、`name`、`description`、`tags` 等

#### 5.4 导入/导出

- **导入**：从 `~/.gemini/settings.json` 读取 `mcpServers`
- **导出**：将启用的 MCP 服务器写入 `~/.gemini/settings.json`
- **同步**：切换供应商时自动同步启用的 MCP 服务器

---

### 6. 配置流程

#### 6.1 添加供应商

1. **选择预设或自定义**：
   - Google Official（OAuth）
   - PackyCode（API Key）
   - 自定义（API Key）

2. **填写配置**：
   - 供应商名称
   - 网站 URL
   - API Key（如需要）
   - 自定义环境变量

3. **验证配置**：
   - 基本格式验证（不强制 API Key）
   - 保存到 `~/.cc-switch/config.json`

#### 6.2 切换供应商

1. **验证配置**：
   - 严格验证（API Key 模式必须有密钥）
   - 检查配置完整性

2. **写入配置文件**：
   ```
   write_gemini_live()
   ├── 检测认证类型（OAuth/PackyCode/Generic）
   ├── 写入 ~/.gemini/.env（环境变量）
   ├── 写入 ~/.gemini/settings.json（认证模式）
   └── 同步 MCP 服务器
   ```

3. **设置认证标志**：
   - **Google Official**：`selectedType = "oauth-personal"`
   - **PackyCode**：`selectedType = "gemini-api-key"`
   - **其他**：保持默认

4. **无需重启**：
   - Gemini CLI 自动检测 `.env` 变化
   - 立即生效

#### 6.3 配置回滚

在切换供应商前，系统会：
1. 创建当前配置的快照（`LiveSnapshot::Gemini`）
2. 保存 `.env` 和 `settings.json` 内容
3. 失败时自动恢复原配置

---

### 7. 代码架构

#### 7.1 核心模块

| 模块 | 文件 | 功能 |
|------|------|------|
| 配置管理 | `gemini_config.rs` | `.env` 文件的读写、解析、验证 |
| MCP 管理 | `gemini_mcp.rs` | MCP 服务器的导入/导出/同步 |
| 供应商服务 | `services/provider.rs` | 供应商切换、配置应用 |
| 预设配置 | `geminiProviderPresets.ts` | 前端预设供应商定义 |

#### 7.2 关键函数

**配置管理**（`gemini_config.rs`）：
```rust
// 获取配置目录
pub fn get_gemini_dir() -> PathBuf

// 读写 .env 文件
pub fn read_gemini_env() -> Result<HashMap<String, String>>
pub fn write_gemini_env_atomic(map: &HashMap<String, String>) -> Result<()>

// JSON 与 .env 转换
pub fn env_to_json(env_map: &HashMap<String, String>) -> Value
pub fn json_to_env(settings: &Value) -> Result<HashMap<String, String>>

// 配置验证
pub fn validate_gemini_settings(settings: &Value) -> Result<()>
pub fn validate_gemini_settings_strict(settings: &Value) -> Result<()>

// 写入认证标志
pub fn write_packycode_settings() -> Result<()>
pub fn write_google_oauth_settings() -> Result<()>
```

**供应商切换**（`services/provider.rs`）：
```rust
// 应用供应商配置
pub(crate) fn write_gemini_live(provider: &Provider) -> Result<()>

// 检测认证类型
fn detect_gemini_auth_type(provider: &Provider) -> GeminiAuthType

// 检测供应商类型
fn is_packycode_gemini(provider: &Provider) -> bool
fn is_google_official_gemini(provider: &Provider) -> bool
```

**MCP 管理**（`gemini_mcp.rs`）：
```rust
// 读写 MCP 服务器配置
pub fn read_mcp_servers_map() -> Result<HashMap<String, Value>>
pub fn set_mcp_servers_map(servers: &HashMap<String, Value>) -> Result<()>

// 读取完整配置
pub fn read_mcp_json() -> Result<Option<String>>
```

#### 7.3 认证类型检测逻辑

供应商类型检测优先级（从高到低）：

1. **Partner Promotion Key**（最可靠）
   ```rust
   provider.meta.partner_promotion_key == "google-official"  // OAuth
   provider.meta.partner_promotion_key == "packycode"        // API Key
   ```

2. **供应商名称匹配**
   ```rust
   name == "google" || name.starts_with("google ")  // OAuth
   name.contains("packycode")                       // API Key
   ```

3. **URL 关键词检测**
   ```rust
   website_url.contains("packycode")      // PackyCode
   base_url.contains("packycode")         // PackyCode
   ```

---

### 8. 常见问题

#### Q1: 环境变量是配置在哪里的？系统环境变量还是配置文件？

**A**:
- **配置文件，不是系统环境变量**
- 配置位置：`~/.gemini/.env` 文件
- Gemini CLI 会自动读取该文件，无需在 `~/.bashrc` 或 `~/.zshrc` 中 export
- 优势：
  - ✅ 不污染系统环境变量
  - ✅ 配置独立于 shell 会话
  - ✅ 更安全（文件权限 600）
  - ✅ 切换供应商时 CC-Switch 自动更新

#### Q2: Gemini CLI 配置文件在哪里？

**A**:
- 主配置目录：`~/.gemini/`
- API Key 和环境变量：`~/.gemini/.env`
- 认证模式和 MCP 服务器：`~/.gemini/settings.json`

#### Q3: 如何切换到 Google Official OAuth 模式？

**A**:
1. 在 CC-Switch 中选择 "Google Official" 预设
2. 点击"启用"切换
3. 重启 Gemini CLI
4. 按照提示完成 OAuth 授权

#### Q4: 如何添加自定义 Gemini 供应商？

**A**:
1. 点击"添加供应商"
2. 选择 Gemini 应用
3. 选择"自定义"预设
4. 填写：
   - 供应商名称
   - API Key（`GEMINI_API_KEY`）
   - Base URL（`GOOGLE_GEMINI_BASE_URL`）
   - 模型名称（`GEMINI_MODEL`）

#### Q5: 切换供应商后是否需要重启 Gemini CLI？

**A**:
- **不需要**：Gemini CLI 会自动检测 `.env` 文件变化
- 托盘快速切换立即生效

#### Q6: MCP 服务器配置如何管理？

**A**:
1. 点击 CC-Switch 右上角"MCP"按钮
2. 添加/编辑/删除 MCP 服务器
3. 勾选"启用"以同步到 Gemini CLI
4. 切换供应商时自动应用

#### Q7: 为什么 PackyCode 需要特殊处理？

**A**:
- PackyCode 是官方合作伙伴
- 需要特殊的安全标志（`selectedType: "gemini-api-key"`）
- 自动检测并应用正确的认证配置

#### Q8: 如何备份和恢复 Gemini 配置？

**A**:
1. **自动备份**：切换供应商前自动创建快照
2. **导出配置**：设置 → 导出配置
3. **恢复配置**：设置 → 导入配置
4. **备份位置**：`~/.cc-switch/backups/`（保留最近 10 个）

#### Q9: 支持哪些环境变量？

**A**:
- `GEMINI_API_KEY` / `GOOGLE_GEMINI_API_KEY`：API 密钥
- `GOOGLE_GEMINI_BASE_URL`：自定义端点
- `GEMINI_MODEL`：默认模型
- 其他自定义变量：根据 API 提供商文档添加

#### Q10: 配置冲突如何处理？

**A**:
- CC-Switch v3.7.0 新增环境变量冲突检测
- 自动检测跨应用配置冲突
- 提供可视化冲突指示和解决建议
- 更改前自动备份

#### Q11: 如何自定义配置目录（云同步）？

**A**:
1. 设置 → "自定义配置目录"
2. 选择云同步文件夹（Dropbox/OneDrive/iCloud）
3. 重启应用
4. 在其他设备重复操作实现同步

---

## English

### Overview

This document provides a comprehensive guide to configuring Gemini CLI in the CC-Switch project, including file structure, authentication modes, MCP server management, and provider switching mechanisms.

### Table of Contents

1. [Configuration File Structure](#1-configuration-file-structure)
2. [Authentication Modes](#2-authentication-modes)
3. [Environment Variables](#3-environment-variables)
4. [Provider Presets](#4-provider-presets)
5. [MCP Server Configuration](#5-mcp-server-configuration)
6. [Configuration Workflow](#6-configuration-workflow)
7. [Code Architecture](#7-code-architecture)
8. [FAQ](#8-faq)

---

### 1. Configuration File Structure

Gemini CLI uses a dual-file configuration system:

#### 1.1 File Locations

```
~/.gemini/
├── .env                    # API keys and environment variables
└── settings.json           # Authentication mode and MCP server config
```

#### 1.2 `.env` File

Stores API keys and custom environment variables.

**Important Notes**:
- 📍 **Configuration Location**: `~/.gemini/.env` file (**NOT system environment variables**)
- 🔄 **Auto-Loading**: Gemini CLI automatically reads this file, no manual export needed
- 🔒 **Security Isolation**: Configuration only affects Gemini CLI, not other programs

**Gemini CLI .env File Loading Order**:
1. Current working directory (searches upward from command execution location)
2. `~/.gemini/.env` (**Recommended location, used by CC-Switch**)
3. `~/.env` (fallback location)

**Format**:
```bash
# API Key (required unless using OAuth)
GEMINI_API_KEY=your-api-key-here

# Custom API endpoint (optional)
GOOGLE_GEMINI_BASE_URL=https://api.example.com

# Model name (optional)
GEMINI_MODEL=gemini-3-pro-preview
```

**Parsing Rules**:
- Supports blank lines and `#` comments
- Format: `KEY=VALUE`
- Keys can only contain letters, numbers, and underscores
- File permissions automatically set to `600` (owner read/write only)

**Difference from System Environment Variables**:
- ❌ **NOT** using `export GEMINI_API_KEY=...` in `~/.bashrc` or `~/.zshrc`
- ✅ **Instead** writing `GEMINI_API_KEY=...` directly in `~/.gemini/.env` file
- ✅ Advantages: No system environment pollution, independent of shell sessions

#### 1.3 `settings.json` File

Stores authentication mode and MCP server configuration.

**Basic Structure**:
```json
{
  "security": {
    "auth": {
      "selectedType": "gemini-api-key"  // or "oauth-personal"
    }
  },
  "mcpServers": {
    "server-id": {
      "command": "node",
      "args": ["/path/to/server.js"],
      "env": {}
    }
  }
}
```

---

### 2. Authentication Modes

Gemini CLI supports two authentication modes:

#### 2.1 API Key Authentication

**Use Cases**: Third-party providers like PackyCode, custom endpoints

**Configuration**:
- `.env` file: Contains `GEMINI_API_KEY` and other environment variables
- `settings.json`: `security.auth.selectedType = "gemini-api-key"`

**Example Configuration**:
```json
{
  "env": {
    "GEMINI_API_KEY": "sk-xxxxx",
    "GOOGLE_GEMINI_BASE_URL": "https://www.packyapi.com",
    "GEMINI_MODEL": "gemini-3-pro-preview"
  }
}
```

#### 2.2 OAuth Authentication

**Use Cases**: Google Official Gemini API

**Configuration**:
- `.env` file: Empty (no API key needed)
- `settings.json`: `security.auth.selectedType = "oauth-personal"`

**Example Configuration**:
```json
{
  "env": {}
}
```

---

### 3. Environment Variables

#### 3.1 Standard Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `GEMINI_API_KEY` | Yes* | API key | `sk-xxxxx` |
| `GOOGLE_GEMINI_API_KEY` | Yes* | API key (alternative name) | `sk-xxxxx` |
| `GOOGLE_GEMINI_BASE_URL` | No | Custom API endpoint | `https://api.example.com` |
| `GEMINI_MODEL` | No | Default model name | `gemini-3-pro-preview` |

*Not required for OAuth mode

#### 3.2 Configuration Validation

**Basic Validation** (when creating provider):
- Validates that `env` field is an object
- Validates that `config` field is an object or null
- Does not enforce `GEMINI_API_KEY` (allows filling later)

**Strict Validation** (when switching provider):
- For API Key mode, must include `GEMINI_API_KEY`
- For OAuth mode, allows empty `env` object

---

### 4. Provider Presets

CC-Switch provides three preset provider configurations:

#### 4.1 Google Official

**Authentication**: OAuth
**Features**:
- Uses Google Official OAuth authentication
- No API key required
- Direct connection to Google AI Studio

**Configuration**:
```typescript
{
  name: "Google Official",
  websiteUrl: "https://ai.google.dev/",
  apiKeyUrl: "https://aistudio.google.com/apikey",
  settingsConfig: {
    env: {}
  },
  category: "official",
  partnerPromotionKey: "google-official"
}
```

#### 4.2 PackyCode

**Authentication**: API Key
**Features**:
- Official partner
- Provides API relay service
- Exclusive promo code: `cc-switch`

**Configuration**:
```typescript
{
  name: "PackyCode",
  websiteUrl: "https://www.packyapi.com",
  settingsConfig: {
    env: {
      GOOGLE_GEMINI_BASE_URL: "https://www.packyapi.com",
      GEMINI_MODEL: "gemini-3-pro-preview"
    }
  },
  category: "third_party",
  isPartner: true,
  partnerPromotionKey: "packycode"
}
```

#### 4.3 Custom

**Authentication**: API Key
**Features**:
- Allows custom API endpoints
- Full configuration control

**Configuration**:
```typescript
{
  name: "Custom",
  websiteUrl: "",
  settingsConfig: {
    env: {
      GOOGLE_GEMINI_BASE_URL: "",
      GEMINI_MODEL: "gemini-3-pro-preview"
    }
  },
  category: "custom"
}
```

---

### 5. MCP Server Configuration

#### 5.1 Configuration Location

MCP server configuration is stored in the `mcpServers` field of `~/.gemini/settings.json`.

#### 5.2 Supported Transport Types

**stdio** (Standard Input/Output):
```json
{
  "server-name": {
    "command": "node",
    "args": ["/path/to/server.js"],
    "env": {
      "ENV_VAR": "value"
    }
  }
}
```

**HTTP** (HTTP Streaming):
```json
{
  "server-name": {
    "httpUrl": "http://localhost:8080/mcp"
  }
}
```

**SSE** (Server-Sent Events):
```json
{
  "server-name": {
    "url": "http://localhost:8080/sse"
  }
}
```

#### 5.3 Format Conversion Rules

CC-Switch uses a unified MCP configuration format internally and performs the following conversions when writing to Gemini CLI:

1. **Remove `type` field**: Gemini CLI infers transport type from field names
2. **HTTP transport conversion**: `url` → `httpUrl`
3. **Remove UI helper fields**: `enabled`, `source`, `id`, `name`, `description`, `tags`, etc.

#### 5.4 Import/Export

- **Import**: Reads `mcpServers` from `~/.gemini/settings.json`
- **Export**: Writes enabled MCP servers to `~/.gemini/settings.json`
- **Sync**: Automatically syncs enabled MCP servers when switching providers

---

### 6. Configuration Workflow

#### 6.1 Adding a Provider

1. **Choose preset or custom**:
   - Google Official (OAuth)
   - PackyCode (API Key)
   - Custom (API Key)

2. **Fill in configuration**:
   - Provider name
   - Website URL
   - API Key (if required)
   - Custom environment variables

3. **Validate configuration**:
   - Basic format validation (does not enforce API Key)
   - Save to `~/.cc-switch/config.json`

#### 6.2 Switching Providers

1. **Validate configuration**:
   - Strict validation (API Key mode must have key)
   - Check configuration completeness

2. **Write configuration files**:
   ```
   write_gemini_live()
   ├── Detect auth type (OAuth/PackyCode/Generic)
   ├── Write ~/.gemini/.env (environment variables)
   ├── Write ~/.gemini/settings.json (auth mode)
   └── Sync MCP servers
   ```

3. **Set authentication flags**:
   - **Google Official**: `selectedType = "oauth-personal"`
   - **PackyCode**: `selectedType = "gemini-api-key"`
   - **Others**: Keep default

4. **No restart required**:
   - Gemini CLI auto-detects `.env` changes
   - Takes effect immediately

#### 6.3 Configuration Rollback

Before switching providers, the system:
1. Creates a snapshot of current configuration (`LiveSnapshot::Gemini`)
2. Saves `.env` and `settings.json` contents
3. Automatically restores original configuration on failure

---

### 7. Code Architecture

#### 7.1 Core Modules

| Module | File | Function |
|--------|------|----------|
| Config Management | `gemini_config.rs` | `.env` file read/write, parsing, validation |
| MCP Management | `gemini_mcp.rs` | MCP server import/export/sync |
| Provider Service | `services/provider.rs` | Provider switching, config application |
| Preset Config | `geminiProviderPresets.ts` | Frontend provider preset definitions |

#### 7.2 Key Functions

**Config Management** (`gemini_config.rs`):
```rust
// Get config directory
pub fn get_gemini_dir() -> PathBuf

// Read/write .env file
pub fn read_gemini_env() -> Result<HashMap<String, String>>
pub fn write_gemini_env_atomic(map: &HashMap<String, String>) -> Result<()>

// JSON to .env conversion
pub fn env_to_json(env_map: &HashMap<String, String>) -> Value
pub fn json_to_env(settings: &Value) -> Result<HashMap<String, String>>

// Config validation
pub fn validate_gemini_settings(settings: &Value) -> Result<()>
pub fn validate_gemini_settings_strict(settings: &Value) -> Result<()>

// Write auth flags
pub fn write_packycode_settings() -> Result<()>
pub fn write_google_oauth_settings() -> Result<()>
```

**Provider Switching** (`services/provider.rs`):
```rust
// Apply provider config
pub(crate) fn write_gemini_live(provider: &Provider) -> Result<()>

// Detect auth type
fn detect_gemini_auth_type(provider: &Provider) -> GeminiAuthType

// Detect provider type
fn is_packycode_gemini(provider: &Provider) -> bool
fn is_google_official_gemini(provider: &Provider) -> bool
```

**MCP Management** (`gemini_mcp.rs`):
```rust
// Read/write MCP server config
pub fn read_mcp_servers_map() -> Result<HashMap<String, Value>>
pub fn set_mcp_servers_map(servers: &HashMap<String, Value>) -> Result<()>

// Read full config
pub fn read_mcp_json() -> Result<Option<String>>
```

#### 7.3 Authentication Type Detection Logic

Provider type detection priority (highest to lowest):

1. **Partner Promotion Key** (most reliable)
   ```rust
   provider.meta.partner_promotion_key == "google-official"  // OAuth
   provider.meta.partner_promotion_key == "packycode"        // API Key
   ```

2. **Provider Name Matching**
   ```rust
   name == "google" || name.starts_with("google ")  // OAuth
   name.contains("packycode")                       // API Key
   ```

3. **URL Keyword Detection**
   ```rust
   website_url.contains("packycode")      // PackyCode
   base_url.contains("packycode")         // PackyCode
   ```

---

### 8. FAQ

#### Q1: Where are these environment variables configured? System environment variables or config files?

**A**:
- **Config files, NOT system environment variables**
- Configuration location: `~/.gemini/.env` file
- Gemini CLI automatically reads this file, no need to export in `~/.bashrc` or `~/.zshrc`
- Advantages:
  - ✅ No system environment pollution
  - ✅ Independent of shell sessions
  - ✅ More secure (file permissions 600)
  - ✅ CC-Switch automatically updates when switching providers

#### Q2: Where are Gemini CLI configuration files located?

**A**:
- Main config directory: `~/.gemini/`
- API Key and environment variables: `~/.gemini/.env`
- Authentication mode and MCP servers: `~/.gemini/settings.json`

#### Q3: How to switch to Google Official OAuth mode?

**A**:
1. Select "Google Official" preset in CC-Switch
2. Click "Enable" to switch
3. Restart Gemini CLI
4. Follow prompts to complete OAuth authorization

#### Q4: How to add a custom Gemini provider?

**A**:
1. Click "Add Provider"
2. Select Gemini app
3. Choose "Custom" preset
4. Fill in:
   - Provider name
   - API Key (`GEMINI_API_KEY`)
   - Base URL (`GOOGLE_GEMINI_BASE_URL`)
   - Model name (`GEMINI_MODEL`)

#### Q5: Do I need to restart Gemini CLI after switching providers?

**A**:
- **No**: Gemini CLI auto-detects `.env` file changes
- Tray quick switch takes effect immediately

#### Q6: How to manage MCP server configurations?

**A**:
1. Click "MCP" button in top-right corner of CC-Switch
2. Add/edit/delete MCP servers
3. Check "Enable" to sync to Gemini CLI
4. Auto-applied when switching providers

#### Q7: Why does PackyCode need special handling?

**A**:
- PackyCode is an official partner
- Requires special security flags (`selectedType: "gemini-api-key"`)
- Auto-detects and applies correct auth configuration

#### Q8: How to backup and restore Gemini configuration?

**A**:
1. **Auto-backup**: Automatically creates snapshot before switching
2. **Export config**: Settings → Export Configuration
3. **Restore config**: Settings → Import Configuration
4. **Backup location**: `~/.cc-switch/backups/` (keeps 10 most recent)

#### Q9: What environment variables are supported?

**A**:
- `GEMINI_API_KEY` / `GOOGLE_GEMINI_API_KEY`: API key
- `GOOGLE_GEMINI_BASE_URL`: Custom endpoint
- `GEMINI_MODEL`: Default model
- Other custom variables: Add according to API provider documentation

#### Q10: How are configuration conflicts handled?

**A**:
- CC-Switch v3.7.0 adds environment variable conflict detection
- Auto-detects cross-app configuration conflicts
- Provides visual conflict indicators and resolution suggestions
- Auto-backup before changes

#### Q11: How to customize config directory (cloud sync)?

**A**:
1. Settings → "Custom Configuration Directory"
2. Choose cloud sync folder (Dropbox/OneDrive/iCloud)
3. Restart app
4. Repeat on other devices to enable sync

---

## 相关资源 / Related Resources

### 官方文档 / Official Documentation
- [Gemini CLI Documentation](https://github.com/google-gemini/gemini-cli)
- [Google AI Studio](https://aistudio.google.com/)
- [Claude Code](https://github.com/anthropics/claude-code)

### 项目文档 / Project Documentation
- [CC-Switch README](../README.md)
- [Changelog](../CHANGELOG.md)
- [MCP Unified Architecture](./v3.7.0-unified-mcp-refactor.md)
- [Backend Architecture](./rust-backend-architecture.md)

### 相关配置指南 / Related Configuration Guides
- [Claude Code Configuration](https://docs.anthropic.com/claude-code)
- [Codex Configuration](https://codex.sh/docs)
- [MCP Server Protocol](https://modelcontextprotocol.io/)

---

## 版本历史 / Version History

- **v3.7.0** (2025-11-19): Initial Gemini CLI support
  - Dual-file configuration system
  - OAuth and API Key authentication modes
  - MCP server management
  - Three provider presets

---

## 贡献 / Contributing

欢迎提交问题和建议！在提交 PR 前，请确保：
Welcome to submit issues and suggestions! Before submitting a PR, please ensure:

- 通过类型检查 / Pass type check: `pnpm typecheck`
- 通过格式检查 / Pass format check: `pnpm format:check`
- 通过单元测试 / Pass unit tests: `pnpm test:unit`

---

## 许可证 / License

MIT © Jason Young
