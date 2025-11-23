# MCP 配置指南 (v3.7.0+)

## 目录

- [概述](#概述)
- [架构设计](#架构设计)
- [数据模型](#数据模型)
- [核心实现](#核心实现)
- [API 接口](#api-接口)
- [配置文件格式](#配置文件格式)
- [使用指南](#使用指南)
- [预设配置](#预设配置)
- [开发参考](#开发参考)

---

## 概述

cc-switch v3.7.0 引入了统一的 MCP（Model Context Protocol）管理系统，允许用户在一个界面中管理所有 MCP 服务器，并通过多选框控制每个服务器应用到哪些 AI 客户端（Claude/Codex/Gemini）。

### 核心特性

- **统一管理**：单一界面管理所有 MCP 服务器
- **灵活配置**：每个服务器可独立选择启用的应用
- **自动同步**：配置变更自动同步到各应用的 live 配置
- **类型安全**：完整的 Rust + TypeScript 类型定义
- **扩展字段**：支持自定义字段（timeout、retry 等）

### 统一管理架构

```
┌───────────────────────────────────────┐
│        统一 MCP 管理面板              │
│  ┌────────┬────────┬────────┬────┐   │
│  │ 服务器 │ Claude │ Codex  │Gem │   │
│  ├────────┼────────┼────────┼────┤   │
│  │ fetch  │   ✓    │   ✓    │    │   │
│  │ time   │   ✓    │        │ ✓  │   │
│  └────────┴────────┴────────┴────┘   │
└───────────────────────────────────────┘
            ↓
      单一数据源 (SSOT)
   ~/.cc-switch/config.json
            ↓
      自动同步到各应用
   ┌──────┬──────┬──────┐
   │Claude│Codex │Gemini│
   └──────┴──────┴──────┘
```

---

## 架构设计

### 分层架构

```
┌─────────────────────────────────────┐
│  Frontend (React + TypeScript)      │
│  - UI 组件                          │
│  - API 封装 (src/lib/api/mcp.ts)    │
└─────────────────────────────────────┘
            ↓ Tauri IPC
┌─────────────────────────────────────┐
│  Commands Layer (Rust)              │
│  - Tauri 命令                       │
│  - 参数验证                         │
│  (src-tauri/src/commands/mcp.rs)    │
└─────────────────────────────────────┘
            ↓
┌─────────────────────────────────────┐
│  Services Layer (Rust)              │
│  - 业务逻辑                         │
│  - 状态管理                         │
│  (src-tauri/src/services/mcp.rs)    │
└─────────────────────────────────────┘
            ↓
┌─────────────────────────────────────┐
│  Data Layer (Rust)                  │
│  - 配置读写                         │
│  - 格式转换                         │
│  - 文件同步                         │
│  (src-tauri/src/mcp.rs)             │
└─────────────────────────────────────┘
            ↓
┌─────────────────────────────────────┐
│  Storage                            │
│  - ~/.cc-switch/config.json (SSOT)  │
│  - ~/.claude.json                   │
│  - ~/.codex/config.toml             │
│  - ~/.gemini/settings.json          │
└─────────────────────────────────────┘
```

### 配置同步流程

```
用户操作 → 前端 API → Tauri 命令
    ↓
更新中央配置 (config.json)
    ↓
自动同步到各应用 live 配置
    ↓
┌─────────────┬─────────────┬─────────────┐
│  Claude     │  Codex      │  Gemini     │
│  (JSON)     │  (TOML)     │  (JSON)     │
└─────────────┴─────────────┴─────────────┘
```

---

## 数据模型

### McpApps - 应用启用状态

```rust
#[derive(Debug, Clone, Serialize, Deserialize, Default, PartialEq)]
pub struct McpApps {
    pub claude: bool,
    pub codex: bool,
    pub gemini: bool,
}
```

**辅助方法**：
- `is_enabled_for(&app)` - 检查应用是否启用
- `set_enabled_for(&app, enabled)` - 设置应用启用状态
- `enabled_apps()` - 返回所有启用的应用列表

### McpServer - MCP 服务器定义

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpServer {
    /// 唯一标识符
    pub id: String,

    /// 显示名称
    pub name: String,

    /// 连接配置 (stdio/http/sse)
    pub server: serde_json::Value,

    /// 启用的应用
    pub apps: McpApps,

    /// 可选字段
    pub description: Option<String>,
    pub homepage: Option<String>,
    pub docs: Option<String>,
    pub tags: Vec<String>,
}
```

### McpServerSpec - 连接配置

#### stdio 类型

```json
{
  "type": "stdio",
  "command": "uvx",
  "args": ["mcp-server-fetch"],
  "env": {
    "API_KEY": "xxx"
  },
  "cwd": "/path/to/workdir"
}
```

#### http/sse 类型

```json
{
  "type": "http",
  "url": "http://localhost:3000/mcp",
  "headers": {
    "Authorization": "Bearer token",
    "X-Custom-Header": "value"
  }
}
```

### 扩展字段支持

系统支持以下常用扩展字段（自动转换）：

**通用字段**：
- `timeout` / `timeout_ms`
- `startup_timeout_ms` / `startup_timeout_sec`
- `connection_timeout` / `read_timeout`
- `debug` / `log_level`
- `disabled`

**stdio 特有**：
- `shell` / `encoding`
- `working_dir`
- `restart_on_exit` / `max_restart_count`

**http/sse 特有**：
- `retry_count` / `max_retry_attempts` / `retry_delay`
- `cache_tools_list`
- `verify_ssl` / `insecure`
- `proxy`

---

## 核心实现

### 服务层 API

**文件位置**：`src-tauri/src/services/mcp.rs`

```rust
impl McpService {
    /// 获取所有 MCP 服务器
    pub fn get_all_servers(state: &AppState)
        -> Result<HashMap<String, McpServer>, AppError>;

    /// 添加或更新 MCP 服务器
    /// 自动同步到所有启用的应用
    pub fn upsert_server(state: &AppState, server: McpServer)
        -> Result<(), AppError>;

    /// 删除 MCP 服务器
    /// 自动从所有应用的 live 配置中移除
    pub fn delete_server(state: &AppState, id: &str)
        -> Result<bool, AppError>;

    /// 切换指定应用的启用状态
    /// 自动同步或移除 live 配置
    pub fn toggle_app(
        state: &AppState,
        server_id: &str,
        app: AppType,
        enabled: bool,
    ) -> Result<(), AppError>;

    /// 手动同步所有启用的服务器到对应应用
    pub fn sync_all_enabled(state: &AppState)
        -> Result<(), AppError>;

    /// 从各应用导入 MCP 配置
    pub fn import_from_claude(state: &AppState)
        -> Result<usize, AppError>;
    pub fn import_from_codex(state: &AppState)
        -> Result<usize, AppError>;
    pub fn import_from_gemini(state: &AppState)
        -> Result<usize, AppError>;
}
```

### 同步逻辑

**文件位置**：`src-tauri/src/mcp.rs`

```rust
/// 将单个 MCP 服务器同步到 Claude
pub fn sync_single_server_to_claude(
    config: &MultiAppConfig,
    id: &str,
    server_spec: &Value,
) -> Result<(), AppError>;

/// 将单个 MCP 服务器同步到 Codex
pub fn sync_single_server_to_codex(
    config: &MultiAppConfig,
    id: &str,
    server_spec: &Value,
) -> Result<(), AppError>;

/// 将单个 MCP 服务器同步到 Gemini
pub fn sync_single_server_to_gemini(
    config: &MultiAppConfig,
    id: &str,
    server_spec: &Value,
) -> Result<(), AppError>;

/// 从各应用 live 配置中移除
pub fn remove_server_from_claude(id: &str) -> Result<(), AppError>;
pub fn remove_server_from_codex(id: &str) -> Result<(), AppError>;
pub fn remove_server_from_gemini(id: &str) -> Result<(), AppError>;

/// 从各应用导入配置
pub fn import_from_claude(config: &mut MultiAppConfig)
    -> Result<usize, AppError>;
pub fn import_from_codex(config: &mut MultiAppConfig)
    -> Result<usize, AppError>;
pub fn import_from_gemini(config: &mut MultiAppConfig)
    -> Result<usize, AppError>;
```

---

## API 接口

### Tauri 命令

**文件位置**：`src-tauri/src/commands/mcp.rs`

```rust
/// 获取所有 MCP 服务器
#[tauri::command]
pub async fn get_mcp_servers(
    state: State<'_, AppState>,
) -> Result<HashMap<String, McpServer>, String>;

/// 添加或更新 MCP 服务器
#[tauri::command]
pub async fn upsert_mcp_server(
    state: State<'_, AppState>,
    server: McpServer,
) -> Result<(), String>;

/// 删除 MCP 服务器
#[tauri::command]
pub async fn delete_mcp_server(
    state: State<'_, AppState>,
    id: String,
) -> Result<bool, String>;

/// 切换应用启用状态
#[tauri::command]
pub async fn toggle_mcp_app(
    state: State<'_, AppState>,
    server_id: String,
    app: String,
    enabled: bool,
) -> Result<(), String>;
```

### 前端 API

**文件位置**：`src/lib/api/mcp.ts`

```typescript
export const mcpApi = {
  /**
   * 获取所有 MCP 服务器（统一结构）
   */
  async getAllServers(): Promise<McpServersMap> {
    return await invoke("get_mcp_servers");
  },

  /**
   * 添加或更新 MCP 服务器
   */
  async upsertUnifiedServer(server: McpServer): Promise<void> {
    return await invoke("upsert_mcp_server", { server });
  },

  /**
   * 删除 MCP 服务器
   */
  async deleteUnifiedServer(id: string): Promise<boolean> {
    return await invoke("delete_mcp_server", { id });
  },

  /**
   * 切换应用启用状态
   */
  async toggleApp(
    serverId: string,
    app: AppId,
    enabled: boolean,
  ): Promise<void> {
    return await invoke("toggle_mcp_app", { serverId, app, enabled });
  },
};
```

### TypeScript 类型定义

**文件位置**：`src/types.ts`

```typescript
export interface McpApps {
  claude: boolean;
  codex: boolean;
  gemini: boolean;
}

export interface McpServer {
  id: string;
  name: string;
  server: McpServerSpec | Record<string, any>;
  apps: McpApps;
  description?: string;
  homepage?: string;
  docs?: string;
  tags?: string[];
}

export type McpServersMap = Record<string, McpServer>;

export type AppId = "claude" | "codex" | "gemini";
```

---

## 配置文件格式

### 中央配置 (~/.cc-switch/config.json)

**单一数据源（SSOT）**，所有 MCP 配置的权威来源。

```json
{
  "mcp": {
    "servers": {
      "fetch": {
        "id": "fetch",
        "name": "mcp-server-fetch",
        "server": {
          "type": "stdio",
          "command": "uvx",
          "args": ["mcp-server-fetch"]
        },
        "apps": {
          "claude": true,
          "codex": true,
          "gemini": false
        },
        "description": "HTTP fetch tool for web requests",
        "homepage": "https://github.com/modelcontextprotocol/servers",
        "docs": "https://github.com/modelcontextprotocol/servers/tree/main/src/fetch",
        "tags": ["stdio", "http", "web"]
      },
      "time": {
        "id": "time",
        "name": "@modelcontextprotocol/server-time",
        "server": {
          "type": "stdio",
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-time"]
        },
        "apps": {
          "claude": true,
          "codex": false,
          "gemini": true
        },
        "tags": ["stdio", "time", "utility"]
      }
    }
  }
}
```

### Claude 配置 (~/.claude.json)

自动同步启用了 Claude 的服务器。

```json
{
  "mcpServers": {
    "fetch": {
      "type": "stdio",
      "command": "uvx",
      "args": ["mcp-server-fetch"]
    },
    "time": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-time"]
    }
  }
}
```

### Codex 配置 (~/.codex/config.toml)

自动同步启用了 Codex 的服务器，使用 TOML 格式。

```toml
# 正确格式：[mcp_servers] 顶层表
[mcp_servers.fetch]
type = "stdio"
command = "uvx"
args = ["mcp-server-fetch"]

[mcp_servers.filesystem]
type = "stdio"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/Users/username"]

[mcp_servers.filesystem.env]
READ_ONLY = "true"
```

**注意**：系统会自动清理错误格式 `[mcp.servers]` 并迁移到正确格式。

### Gemini 配置 (~/.gemini/settings.json)

自动同步启用了 Gemini 的服务器。

```json
{
  "mcpServers": {
    "time": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-time"]
    }
  }
}
```

---

## 使用指南

### 添加 MCP 服务器

**方法 1：使用前端 API**

```typescript
import { mcpApi } from "@/lib/api/mcp";

const newServer: McpServer = {
  id: "my-server",
  name: "My Custom Server",
  server: {
    type: "stdio",
    command: "node",
    args: ["server.js"],
  },
  apps: {
    claude: true,
    codex: true,
    gemini: false,
  },
  description: "My custom MCP server",
  tags: ["custom", "nodejs"],
};

await mcpApi.upsertUnifiedServer(newServer);
```

**方法 2：直接编辑配置文件**

编辑 `~/.cc-switch/config.json`，然后调用同步：

```typescript
// 手动同步所有配置
await invoke("sync_all_mcp_servers");
```

### 切换应用启用状态

```typescript
// 启用 fetch 服务器的 Gemini 应用
await mcpApi.toggleApp("fetch", "gemini", true);

// 禁用 fetch 服务器的 Codex 应用
await mcpApi.toggleApp("fetch", "codex", false);
```

### 删除 MCP 服务器

```typescript
// 删除服务器（自动从所有应用移除）
const deleted = await mcpApi.deleteUnifiedServer("fetch");
if (deleted) {
  console.log("服务器已删除");
}
```

### 从现有配置导入

```rust
// 从 Claude 导入
let count = McpService::import_from_claude(&state)?;
println!("从 Claude 导入了 {} 个服务器", count);

// 从 Codex 导入
let count = McpService::import_from_codex(&state)?;
println!("从 Codex 导入了 {} 个服务器", count);

// 从 Gemini 导入
let count = McpService::import_from_gemini(&state)?;
println!("从 Gemini 导入了 {} 个服务器", count);
```

**导入规则**：
- 新服务器：创建并启用对应应用
- 已存在的服务器：仅更新应用启用状态，不覆盖其他字段

---

## 预设配置

**文件位置**：`src/config/mcpPresets.ts`

系统提供常用 MCP 服务器的预设配置：

```typescript
export const mcpPresets: McpPreset[] = [
  {
    id: "fetch",
    name: "mcp-server-fetch",
    tags: ["stdio", "http", "web"],
    server: {
      type: "stdio",
      command: "uvx",
      args: ["mcp-server-fetch"],
    },
    homepage: "https://github.com/modelcontextprotocol/servers",
    docs: "https://github.com/modelcontextprotocol/servers/tree/main/src/fetch",
  },
  {
    id: "time",
    name: "@modelcontextprotocol/server-time",
    tags: ["stdio", "time", "utility"],
    server: {
      type: "stdio",
      command: "npx",
      args: ["-y", "@modelcontextprotocol/server-time"],
    },
    homepage: "https://github.com/modelcontextprotocol/servers",
    docs: "https://github.com/modelcontextprotocol/servers/tree/main/src/time",
  },
  {
    id: "memory",
    name: "@modelcontextprotocol/server-memory",
    tags: ["stdio", "memory", "graph"],
    server: {
      type: "stdio",
      command: "npx",
      args: ["-y", "@modelcontextprotocol/server-memory"],
    },
    homepage: "https://github.com/modelcontextprotocol/servers",
    docs: "https://github.com/modelcontextprotocol/servers/tree/main/src/memory",
  },
  {
    id: "sequential-thinking",
    name: "@modelcontextprotocol/server-sequential-thinking",
    tags: ["stdio", "thinking", "reasoning"],
    server: {
      type: "stdio",
      command: "npx",
      args: ["-y", "@modelcontextprotocol/server-sequential-thinking"],
    },
    homepage: "https://github.com/modelcontextprotocol/servers",
    docs: "https://github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking",
  },
  {
    id: "context7",
    name: "@upstash/context7-mcp",
    tags: ["stdio", "docs", "search"],
    server: {
      type: "stdio",
      command: "npx",
      args: ["-y", "@upstash/context7-mcp"],
    },
    homepage: "https://context7.com",
    docs: "https://github.com/upstash/context7/blob/master/README.md",
  },
];
```

### 使用预设

```typescript
import { mcpPresets, getMcpPresetWithDescription } from "@/config/mcpPresets";
import { useTranslation } from "react-i18next";

const { t } = useTranslation();

// 获取带国际化描述的预设
const fetchPreset = getMcpPresetWithDescription(
  mcpPresets[0],
  t
);

// 转换为完整的 McpServer 并添加
const server: McpServer = {
  ...fetchPreset,
  apps: {
    claude: true,
    codex: true,
    gemini: false,
  },
};

await mcpApi.upsertUnifiedServer(server);
```

---

## 开发参考

### 关键文件清单

#### 后端 (Rust)

| 文件 | 说明 |
|------|------|
| `src-tauri/src/app_config.rs` | 数据模型定义（McpApps, McpServer, McpRoot） |
| `src-tauri/src/services/mcp.rs` | 业务逻辑层（CRUD、同步、导入） |
| `src-tauri/src/mcp.rs` | 数据层（文件读写、格式转换、同步） |
| `src-tauri/src/commands/mcp.rs` | Tauri 命令层 |
| `src-tauri/src/claude_mcp.rs` | Claude MCP 操作 |
| `src-tauri/src/gemini_mcp.rs` | Gemini MCP 操作 |
| `src-tauri/src/codex_config.rs` | Codex 配置操作 |

#### 前端 (TypeScript/React)

| 文件 | 说明 |
|------|------|
| `src/types.ts` | TypeScript 类型定义 |
| `src/lib/api/mcp.ts` | API 封装层 |
| `src/lib/schemas/mcp.ts` | Zod 验证模式 |
| `src/config/mcpPresets.ts` | 预设配置 |
| `src/components/mcp/` | UI 组件（待实现） |

### 添加新的应用支持

如果需要支持新的 AI 应用（如 Cursor），需要修改以下内容：

1. **更新数据模型** (`app_config.rs`)：
```rust
pub struct McpApps {
    pub claude: bool,
    pub codex: bool,
    pub gemini: bool,
    pub cursor: bool,  // 新增
}
```

2. **添加同步函数** (`mcp.rs`)：
```rust
pub fn sync_single_server_to_cursor(...) -> Result<(), AppError> {
    // 实现同步逻辑
}

pub fn remove_server_from_cursor(id: &str) -> Result<(), AppError> {
    // 实现移除逻辑
}

pub fn import_from_cursor(config: &mut MultiAppConfig) -> Result<usize, AppError> {
    // 实现导入逻辑
}
```

3. **更新服务层** (`services/mcp.rs`)：
```rust
fn sync_server_to_app_internal(...) -> Result<(), AppError> {
    match app {
        AppType::Claude => { ... }
        AppType::Codex => { ... }
        AppType::Gemini => { ... }
        AppType::Cursor => {  // 新增
            mcp::sync_single_server_to_cursor(cfg, &server.id, &server.server)?;
        }
    }
    Ok(())
}
```

4. **更新前端类型** (`types.ts`)：
```typescript
export type AppId = "claude" | "codex" | "gemini" | "cursor";
```

### 数据校验

**后端校验** (`mcp.rs:8`):

```rust
fn validate_server_spec(spec: &Value) -> Result<(), AppError> {
    // 1. 校验类型：stdio/http/sse
    // 2. stdio 必须有 command
    // 3. http/sse 必须有 url
}
```

**前端校验** (`lib/schemas/mcp.ts`):

```typescript
export const mcpServerSchema = z.object({
  id: z.string().min(1, "请输入服务器 ID"),
  name: z.string().optional(),
  description: z.string().optional(),
  tags: z.array(z.string()).optional(),
  homepage: z.string().url().optional(),
  docs: z.string().url().optional(),
  enabled: z.boolean().optional(),
  server: mcpServerSpecSchema,
});
```

### 错误处理

```rust
pub enum AppError {
    McpValidation(String),      // MCP 配置验证错误
    InvalidInput(String),        // 无效输入
    Io(String),                  // IO 错误
    // ...
}
```

前端捕获：

```typescript
try {
  await mcpApi.upsertUnifiedServer(server);
} catch (error) {
  console.error("添加 MCP 服务器失败:", error);
  // 显示错误提示
}
```

### 日志记录

系统使用 Rust `log` 库记录关键操作：

```rust
log::info!("导入新 MCP 服务器 '{id}'");
log::warn!("跳过无效 MCP 服务器 '{id}': {e}");
log::error!("config.json 中存在无效的 MCP 条目 '{id}': {err}");
```

---

## 最佳实践

### 1. 配置管理

- **始终通过 API 修改配置**，避免直接编辑 live 配置文件
- **使用预设配置**作为起点，减少配置错误
- **定期备份** `~/.cc-switch/config.json`

### 2. 同步策略

- 配置变更会**自动同步**，无需手动触发
- 如需强制同步，使用 `McpService::sync_all_enabled()`
- 导入后会自动保存并同步

### 3. 扩展字段

- 使用通用字段名（如 `timeout_ms` 而非自定义名称）
- 扩展字段会自动转换，但复杂类型可能被跳过
- 查看日志了解哪些字段被成功转换

### 4. 错误恢复

- 配置文件有自动备份：`config.v1.backup.<timestamp>.json`
- 迁移失败时可手动回滚到备份
- 单个服务器校验失败不会影响其他服务器

---

## 常见问题

### Q: 如何查看当前配置？

```bash
# 查看中央配置
cat ~/.cc-switch/config.json | jq .mcp.servers

# 查看 Claude 配置
cat ~/.claude.json | jq .mcpServers

# 查看 Codex 配置
cat ~/.codex/config.toml
```

### Q: 配置没有同步怎么办？

```typescript
// 手动触发全量同步
await invoke("sync_all_mcp_servers");
```

### Q: 如何添加自定义字段？

直接在 `server` 对象中添加，系统会自动转换：

```typescript
server: {
  type: "stdio",
  command: "my-command",
  args: ["arg1"],
  timeout_ms: 30000,      // 自定义字段
  max_retry_count: 3,     // 自定义字段
}
```

### Q: Codex 格式错误怎么办？

系统会自动检测并清理错误格式 `[mcp.servers]`，迁移到正确格式 `[mcp_servers]`。无需手动操作。

---

## 更新日志

### v3.7.0 (2025-11-14)

- 统一 MCP 管理架构
- 自动迁移旧配置
- 支持扩展字段
- 完整的类型定义
- Codex 格式容错和自动修复

---

## 参考资源

- [Model Context Protocol 官方文档](https://modelcontextprotocol.io)
- [MCP Servers 仓库](https://github.com/modelcontextprotocol/servers)
- [Tauri 文档](https://tauri.app)
- [项目 GitHub](https://github.com/jasonyoungyang/cc-switch)
