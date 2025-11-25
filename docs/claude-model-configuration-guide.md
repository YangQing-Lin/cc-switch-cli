# Claude 模型配置机制技术文档

> CC Switch 中 Claude 供应商的四层模型配置实现指南
>
> 版本：v3.7.x
>
> 最后更新：2025-11-25

---

## 目录

- [1. 概述](#1-概述)
- [2. 配置项定义](#2-配置项定义)
- [3. 前端实现](#3-前端实现)
- [4. 后端实现](#4-后端实现)
- [5. 数据流转](#5-数据流转)
- [6. 关键代码位置](#6-关键代码位置)
- [7. 开发指南](#7-开发指南)

---

## 1. 概述

CC Switch 支持为 Claude 供应商配置四个独立的模型字段，用于精细化控制不同场景下使用的模型版本。这些配置最终会写入 `~/.claude/settings.json`，供 Claude Code CLI 读取。

### 1.1 设计目标

- **灵活性**：支持针对不同层级（Haiku/Sonnet/Opus）配置默认模型
- **向后兼容**：处理旧版本的 `ANTHROPIC_SMALL_FAST_MODEL` 键迁移
- **原子性**：配置更新过程支持事务回滚
- **双重归一化**：保存和启用时分别进行键名标准化

### 1.2 配置项列表

| 字段名 | 环境变量名 | 用途 | 优先级 |
|-------|-----------|------|-------|
| 主模型 | `ANTHROPIC_MODEL` | 全局默认模型 | 基准 |
| Haiku 默认模型 | `ANTHROPIC_DEFAULT_HAIKU_MODEL` | 快速任务使用的模型 | 高 |
| Sonnet 默认模型 | `ANTHROPIC_DEFAULT_SONNET_MODEL` | 标准任务使用的模型 | 高 |
| Opus 默认模型 | `ANTHROPIC_DEFAULT_OPUS_MODEL` | 复杂任务使用的模型 | 高 |

---

## 2. 配置项定义

### 2.1 环境变量映射

在 `~/.claude/settings.json` 中的结构：

```json
{
  "env": {
    "ANTHROPIC_API_KEY": "sk-ant-xxx",
    "ANTHROPIC_BASE_URL": "https://api.anthropic.com",
    "ANTHROPIC_MODEL": "claude-sonnet-4",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-haiku-3.5",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-sonnet-4",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-opus-4"
  }
}
```

### 2.2 旧键兼容逻辑

**已废弃的键**：`ANTHROPIC_SMALL_FAST_MODEL`

**回退规则**（当新键不存在时）：

- **Haiku**：`DEFAULT_HAIKU` → `SMALL_FAST` → `MODEL`
- **Sonnet**：`DEFAULT_SONNET` → `MODEL` → `SMALL_FAST`
- **Opus**：`DEFAULT_OPUS` → `MODEL` → `SMALL_FAST`

**处理策略**：
- 读取时：从旧键回退填充新键
- 写入时：删除 `ANTHROPIC_SMALL_FAST_MODEL`

---

## 3. 前端实现

### 3.1 UI 组件层

**文件**：`src/components/providers/forms/ClaudeFormFields.tsx`

**关键代码**（第 165-261 行）：

```tsx
{shouldShowModelSelector && (
  <div className="space-y-3">
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {/* 主模型 */}
      <div className="space-y-2">
        <FormLabel htmlFor="claudeModel">
          {t("providerForm.anthropicModel", { defaultValue: "主模型" })}
        </FormLabel>
        <Input
          id="claudeModel"
          type="text"
          value={claudeModel}
          onChange={(e) => onModelChange("ANTHROPIC_MODEL", e.target.value)}
          placeholder={t("providerForm.modelPlaceholder")}
          autoComplete="off"
        />
      </div>

      {/* Haiku 默认模型 */}
      <div className="space-y-2">
        <FormLabel htmlFor="claudeDefaultHaikuModel">
          {t("providerForm.anthropicDefaultHaikuModel", {
            defaultValue: "Haiku 默认模型",
          })}
        </FormLabel>
        <Input
          id="claudeDefaultHaikuModel"
          type="text"
          value={defaultHaikuModel}
          onChange={(e) =>
            onModelChange("ANTHROPIC_DEFAULT_HAIKU_MODEL", e.target.value)
          }
          placeholder={t("providerForm.haikuModelPlaceholder")}
          autoComplete="off"
        />
      </div>

      {/* Sonnet 默认模型 */}
      <div className="space-y-2">
        <FormLabel htmlFor="claudeDefaultSonnetModel">
          {t("providerForm.anthropicDefaultSonnetModel", {
            defaultValue: "Sonnet 默认模型",
          })}
        </FormLabel>
        <Input
          id="claudeDefaultSonnetModel"
          type="text"
          value={defaultSonnetModel}
          onChange={(e) =>
            onModelChange("ANTHROPIC_DEFAULT_SONNET_MODEL", e.target.value)
          }
          placeholder={t("providerForm.modelPlaceholder")}
          autoComplete="off"
        />
      </div>

      {/* Opus 默认模型 */}
      <div className="space-y-2">
        <FormLabel htmlFor="claudeDefaultOpusModel">
          {t("providerForm.anthropicDefaultOpusModel", {
            defaultValue: "Opus 默认模型",
          })}
        </FormLabel>
        <Input
          id="claudeDefaultOpusModel"
          type="text"
          value={defaultOpusModel}
          onChange={(e) =>
            onModelChange("ANTHROPIC_DEFAULT_OPUS_MODEL", e.target.value)
          }
          placeholder={t("providerForm.modelPlaceholder")}
          autoComplete="off"
        />
      </div>
    </div>
  </div>
)}
```

**显示条件**：
- 仅当 `category !== "official"` 时显示（官方登录模式不显示）

### 3.2 状态管理层

**文件**：`src/components/providers/forms/hooks/useModelState.ts`

**核心逻辑**：

#### 3.2.1 初始化读取（第 26-56 行）

```typescript
useEffect(() => {
  try {
    const cfg = settingsConfig ? JSON.parse(settingsConfig) : {};
    const env = cfg?.env || {};

    const model = typeof env.ANTHROPIC_MODEL === "string"
      ? env.ANTHROPIC_MODEL : "";
    const small = typeof env.ANTHROPIC_SMALL_FAST_MODEL === "string"
      ? env.ANTHROPIC_SMALL_FAST_MODEL : "";

    // 回退填充逻辑
    const haiku = typeof env.ANTHROPIC_DEFAULT_HAIKU_MODEL === "string"
      ? env.ANTHROPIC_DEFAULT_HAIKU_MODEL
      : small || model;
    const sonnet = typeof env.ANTHROPIC_DEFAULT_SONNET_MODEL === "string"
      ? env.ANTHROPIC_DEFAULT_SONNET_MODEL
      : model || small;
    const opus = typeof env.ANTHROPIC_DEFAULT_OPUS_MODEL === "string"
      ? env.ANTHROPIC_DEFAULT_OPUS_MODEL
      : model || small;

    setClaudeModel(model || "");
    setDefaultHaikuModel(haiku || "");
    setDefaultSonnetModel(sonnet || "");
    setDefaultOpusModel(opus || "");
  } catch {
    // ignore
  }
}, [settingsConfig]);
```

#### 3.2.2 更新处理（第 58-96 行）

```typescript
const handleModelChange = useCallback(
  (
    field:
      | "ANTHROPIC_MODEL"
      | "ANTHROPIC_DEFAULT_HAIKU_MODEL"
      | "ANTHROPIC_DEFAULT_SONNET_MODEL"
      | "ANTHROPIC_DEFAULT_OPUS_MODEL",
    value: string,
  ) => {
    // 更新本地状态
    if (field === "ANTHROPIC_MODEL") setClaudeModel(value);
    if (field === "ANTHROPIC_DEFAULT_HAIKU_MODEL") setDefaultHaikuModel(value);
    if (field === "ANTHROPIC_DEFAULT_SONNET_MODEL") setDefaultSonnetModel(value);
    if (field === "ANTHROPIC_DEFAULT_OPUS_MODEL") setDefaultOpusModel(value);

    try {
      const currentConfig = settingsConfig
        ? JSON.parse(settingsConfig)
        : { env: {} };
      if (!currentConfig.env) currentConfig.env = {};

      // 新键写入
      const trimmed = value.trim();
      if (trimmed) {
        currentConfig.env[field] = trimmed;
      } else {
        delete currentConfig.env[field];
      }

      // 删除旧键
      delete currentConfig.env["ANTHROPIC_SMALL_FAST_MODEL"];

      onConfigChange(JSON.stringify(currentConfig, null, 2));
    } catch (err) {
      console.error("Failed to update model config:", err);
    }
  },
  [settingsConfig, onConfigChange],
);
```

### 3.3 表单提交层

**文件**：`src/components/providers/forms/ProviderForm.tsx`

**提交逻辑**（第 386-434 行）：

```typescript
const handleSubmit = (values: ProviderFormData) => {
  // ... 验证逻辑

  // Claude: 使用表单配置
  settingsConfig = values.settingsConfig.trim();

  const payload: ProviderFormValues = {
    ...values,
    name: values.name.trim(),
    websiteUrl: values.websiteUrl?.trim() ?? "",
    settingsConfig, // 包含四个模型配置的 JSON 字符串
  };

  onSubmit(payload);
};
```

---

## 4. 后端实现

### 4.1 命令层

**文件**：`src-tauri/src/commands/provider.rs`

**关键命令**：

- `add_provider`（第 29-37 行）：添加新供应商
- `update_provider`（第 39-48 行）：更新现有供应商
- `switch_provider`（第 78-87 行）：切换当前供应商

### 4.2 服务层 - 保存逻辑

**文件**：`src-tauri/src/services/provider.rs`

#### 4.2.1 归一化逻辑（第 421-491 行）

```rust
fn normalize_claude_models_in_value(settings: &mut Value) -> bool {
    let mut changed = false;
    let env = match settings.get_mut("env") {
        Some(v) if v.is_object() => v.as_object_mut().unwrap(),
        _ => return changed,
    };

    let model = env
        .get("ANTHROPIC_MODEL")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());
    let small_fast = env
        .get("ANTHROPIC_SMALL_FAST_MODEL")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());

    let current_haiku = env
        .get("ANTHROPIC_DEFAULT_HAIKU_MODEL")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());
    let current_sonnet = env
        .get("ANTHROPIC_DEFAULT_SONNET_MODEL")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());
    let current_opus = env
        .get("ANTHROPIC_DEFAULT_OPUS_MODEL")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());

    // 回退填充逻辑
    let target_haiku = current_haiku
        .or_else(|| small_fast.clone())
        .or_else(|| model.clone());
    let target_sonnet = current_sonnet
        .or_else(|| model.clone())
        .or_else(|| small_fast.clone());
    let target_opus = current_opus
        .or_else(|| model.clone())
        .or_else(|| small_fast.clone());

    // 写入新键（如果原本不存在）
    if env.get("ANTHROPIC_DEFAULT_HAIKU_MODEL").is_none() {
        if let Some(v) = target_haiku {
            env.insert("ANTHROPIC_DEFAULT_HAIKU_MODEL".to_string(), Value::String(v));
            changed = true;
        }
    }
    if env.get("ANTHROPIC_DEFAULT_SONNET_MODEL").is_none() {
        if let Some(v) = target_sonnet {
            env.insert("ANTHROPIC_DEFAULT_SONNET_MODEL".to_string(), Value::String(v));
            changed = true;
        }
    }
    if env.get("ANTHROPIC_DEFAULT_OPUS_MODEL").is_none() {
        if let Some(v) = target_opus {
            env.insert("ANTHROPIC_DEFAULT_OPUS_MODEL".to_string(), Value::String(v));
            changed = true;
        }
    }

    // 删除旧键
    if env.remove("ANTHROPIC_SMALL_FAST_MODEL").is_some() {
        changed = true;
    }

    changed
}
```

#### 4.2.2 添加供应商（第 744-779 行）

```rust
pub fn add(state: &AppState, app_type: AppType, provider: Provider) -> Result<bool, AppError> {
    let mut provider = provider;
    // 归一化 Claude 模型键
    Self::normalize_provider_if_claude(&app_type, &mut provider);
    Self::validate_provider_settings(&app_type, &provider)?;

    let app_type_clone = app_type.clone();
    let provider_clone = provider.clone();

    Self::run_transaction(state, move |config| {
        config.ensure_app(&app_type_clone);
        let manager = config
            .get_manager_mut(&app_type_clone)
            .ok_or_else(|| Self::app_not_found(&app_type_clone))?;

        let is_current = manager.current == provider_clone.id;
        manager
            .providers
            .insert(provider_clone.id.clone(), provider_clone.clone());

        // 如果是当前供应商，创建后置操作写入 live 文件
        let action = if is_current {
            let backup = Self::capture_live_snapshot(&app_type_clone)?;
            Some(PostCommitAction {
                app_type: app_type_clone.clone(),
                provider: provider_clone.clone(),
                backup,
                sync_mcp: false,
                refresh_snapshot: false,
            })
        } else {
            None
        };

        Ok((true, action))
    })
}
```

#### 4.2.3 更新供应商（第 782-846 行）

```rust
pub fn update(
    state: &AppState,
    app_type: AppType,
    provider: Provider,
) -> Result<bool, AppError> {
    let mut provider = provider;
    // 归一化 Claude 模型键
    Self::normalize_provider_if_claude(&app_type, &mut provider);
    Self::validate_provider_settings(&app_type, &provider)?;

    let provider_id = provider.id.clone();
    let app_type_clone = app_type.clone();
    let provider_clone = provider.clone();

    Self::run_transaction(state, move |config| {
        let manager = config
            .get_manager_mut(&app_type_clone)
            .ok_or_else(|| Self::app_not_found(&app_type_clone))?;

        if !manager.providers.contains_key(&provider_id) {
            return Err(AppError::localized(
                "provider.not_found",
                format!("供应商不存在: {provider_id}"),
                format!("Provider not found: {provider_id}"),
            ));
        }

        let is_current = manager.current == provider_id;

        // 合并 meta 数据
        let merged = if let Some(existing) = manager.providers.get(&provider_id) {
            let mut updated = provider_clone.clone();
            match (existing.meta.as_ref(), updated.meta.take()) {
                (Some(old_meta), None) => {
                    updated.meta = Some(old_meta.clone());
                }
                (None, None) => {
                    updated.meta = None;
                }
                (_old, Some(new_meta)) => {
                    updated.meta = Some(new_meta);
                }
            }
            updated
        } else {
            provider_clone.clone()
        };

        manager.providers.insert(provider_id.clone(), merged);

        // 如果是当前供应商，创建后置操作
        let action = if is_current {
            let backup = Self::capture_live_snapshot(&app_type_clone)?;
            Some(PostCommitAction {
                app_type: app_type_clone.clone(),
                provider: provider_clone.clone(),
                backup,
                sync_mcp: false,
                refresh_snapshot: false,
            })
        } else {
            None
        };

        Ok((true, action))
    })
}
```

### 4.3 服务层 - 应用逻辑

#### 4.3.1 写入 Live 文件（第 1540-1546 行）

```rust
fn write_claude_live(provider: &Provider) -> Result<(), AppError> {
    let settings_path = get_claude_settings_path(); // ~/.claude/settings.json
    let mut content = provider.settings_config.clone();

    // 再次归一化（确保写入文件时格式正确）
    let _ = Self::normalize_claude_models_in_value(&mut content);

    // 原子写入（临时文件 + rename）
    write_json_file(&settings_path, &content)?;

    Ok(())
}
```

#### 4.3.2 切换供应商流程（第 1616-1622 行）

```rust
fn write_live_snapshot(app_type: &AppType, provider: &Provider) -> Result<(), AppError> {
    match app_type {
        AppType::Codex => Self::write_codex_live(provider),
        AppType::Claude => Self::write_claude_live(provider),
        AppType::Gemini => Self::write_gemini_live(provider),
    }
}
```

---

## 5. 数据流转

### 5.1 保存配置流程

```
用户编辑表单
    ↓
Input onChange 触发
    ↓
useModelState.handleModelChange
    ├─ 更新本地状态（useState）
    ├─ 解析 settingsConfig JSON
    ├─ 写入 env[field] = value
    ├─ 删除旧键 ANTHROPIC_SMALL_FAST_MODEL
    └─ onConfigChange(新的 JSON 字符串)
    ↓
ProviderForm.form.setValue("settingsConfig", ...)
    ↓
点击保存按钮
    ↓
ProviderForm.handleSubmit
    ├─ 验证表单
    └─ 调用 onSubmit(payload)
    ↓
providersApi.add() / providersApi.update()
    ↓
Tauri IPC 调用
    ↓
[后端] add_provider / update_provider 命令
    ↓
ProviderService::add / update
    ├─ normalize_provider_if_claude（归一化 #1）
    ├─ validate_provider_settings
    └─ run_transaction
        ├─ 插入/更新 config.providers[id]
        ├─ 保存到 ~/.cc-switch/config.json
        └─ 如果是 current，创建 PostCommitAction
            └─ 调用 write_live_snapshot
                └─ write_claude_live
                    ├─ normalize_claude_models_in_value（归一化 #2）
                    └─ 写入 ~/.claude/settings.json
```

### 5.2 启用供应商流程

```
用户点击"启用"按钮
    ↓
providersApi.switch(id)
    ↓
Tauri IPC 调用
    ↓
[后端] switch_provider 命令
    ↓
ProviderService::switch
    ↓
run_transaction
    ├─ capture_live_snapshot（备份当前配置）
    ├─ backfill_claude_current（回填旧供应商配置）
    ├─ 更新 config.current = new_provider_id
    ├─ 保存到 ~/.cc-switch/config.json
    └─ apply_post_commit
        ├─ write_live_snapshot
        │   └─ write_claude_live
        │       ├─ normalize_claude_models_in_value（归一化）
        │       └─ 写入 ~/.claude/settings.json
        └─ McpService::sync_all_enabled（同步 MCP 配置）
```

### 5.3 配置文件位置

| 文件路径 | 用途 | 格式 |
|---------|------|------|
| `~/.cc-switch/config.json` | SSOT 配置存储 | JSON |
| `~/.claude/settings.json` | Claude Code 读取的实时配置 | JSON |
| `~/.cc-switch/backups/` | 配置自动备份（保留 10 个） | JSON |

---

## 6. 关键代码位置

### 6.1 前端代码

| 功能模块 | 文件路径 | 关键行号 |
|---------|---------|---------|
| **UI 组件** | `src/components/providers/forms/ClaudeFormFields.tsx` | 165-261 |
| **状态管理 Hook** | `src/components/providers/forms/hooks/useModelState.ts` | 完整文件 |
| - 初始化读取 | 同上 | 26-56 |
| - 更新处理 | 同上 | 58-96 |
| **表单提交** | `src/components/providers/forms/ProviderForm.tsx` | 386-434 |
| **API 调用** | `src/lib/api/providers.ts` | 25-39 |

### 6.2 后端代码

| 功能模块 | 文件路径 | 关键行号 |
|---------|---------|---------|
| **命令层** | `src-tauri/src/commands/provider.rs` | 29-87 |
| - add_provider | 同上 | 29-37 |
| - update_provider | 同上 | 39-48 |
| - switch_provider | 同上 | 78-87 |
| **服务层** | `src-tauri/src/services/provider.rs` | 完整文件 |
| - 归一化逻辑 | 同上 | 421-491 |
| - 归一化包装 | 同上 | 493-500 |
| - 添加供应商 | 同上 | 744-779 |
| - 更新供应商 | 同上 | 782-846 |
| - 写入 live 文件 | 同上 | 1540-1546 |
| - 写入快照 | 同上 | 1616-1622 |
| **事务机制** | 同上 | 501-573 |

---

## 7. 开发指南

### 7.1 修改模型配置字段

**场景**：需要添加新的模型配置项，例如 `ANTHROPIC_DEFAULT_EXTENDED_MODEL`

#### 前端修改步骤：

1. **修改 Hook 状态**（`useModelState.ts`）：
   ```typescript
   const [defaultExtendedModel, setDefaultExtendedModel] = useState("");
   ```

2. **添加初始化逻辑**（`useModelState.ts:useEffect`）：
   ```typescript
   const extended = typeof env.ANTHROPIC_DEFAULT_EXTENDED_MODEL === "string"
     ? env.ANTHROPIC_DEFAULT_EXTENDED_MODEL
     : model || small;
   setDefaultExtendedModel(extended || "");
   ```

3. **添加更新处理**（`useModelState.ts:handleModelChange`）：
   ```typescript
   if (field === "ANTHROPIC_DEFAULT_EXTENDED_MODEL")
     setDefaultExtendedModel(value);
   ```

4. **添加 UI 输入框**（`ClaudeFormFields.tsx`）：
   ```tsx
   <div className="space-y-2">
     <FormLabel htmlFor="claudeDefaultExtendedModel">
       Extended 默认模型
     </FormLabel>
     <Input
       id="claudeDefaultExtendedModel"
       type="text"
       value={defaultExtendedModel}
       onChange={(e) =>
         onModelChange("ANTHROPIC_DEFAULT_EXTENDED_MODEL", e.target.value)
       }
       autoComplete="off"
     />
   </div>
   ```

5. **更新 Props 类型**：
   - `ClaudeFormFields.tsx` 的 `ClaudeFormFieldsProps` 接口
   - `ProviderForm.tsx` 调用 `ClaudeFormFields` 的 props

#### 后端修改步骤：

1. **修改归一化逻辑**（`provider.rs:normalize_claude_models_in_value`）：
   ```rust
   let current_extended = env
       .get("ANTHROPIC_DEFAULT_EXTENDED_MODEL")
       .and_then(|v| v.as_str())
       .map(|s| s.to_string());

   let target_extended = current_extended
       .or_else(|| model.clone())
       .or_else(|| small_fast.clone());

   if env.get("ANTHROPIC_DEFAULT_EXTENDED_MODEL").is_none() {
       if let Some(v) = target_extended {
           env.insert("ANTHROPIC_DEFAULT_EXTENDED_MODEL".to_string(), Value::String(v));
           changed = true;
       }
   }
   ```

### 7.2 调试技巧

#### 前端调试：

```typescript
// 在 useModelState.ts 的 handleModelChange 中添加日志
console.log("Model change:", field, value);
console.log("Current config:", JSON.parse(settingsConfig || "{}"));
```

#### 后端调试：

```rust
// 在 provider.rs 的 normalize_claude_models_in_value 中添加日志
eprintln!("Normalizing Claude models...");
eprintln!("Before: {:?}", settings);
// ... 处理逻辑
eprintln!("After: {:?}", settings);
eprintln!("Changed: {}", changed);
```

#### 查看最终写入的配置：

```bash
# 查看 live 文件
cat ~/.claude/settings.json | jq '.env'

# 查看 SSOT 配置
cat ~/.cc-switch/config.json | jq '.claude.providers["provider-id"].settings_config.env'
```

### 7.3 测试清单

添加或修改模型配置字段后，必须测试以下场景：

- [ ] **新建供应商**：填写新字段，保存后检查 `config.json`
- [ ] **编辑供应商**：修改新字段，保存后检查是否正确更新
- [ ] **启用供应商**：检查新字段是否正确写入 `~/.claude/settings.json`
- [ ] **旧配置迁移**：使用包含 `ANTHROPIC_SMALL_FAST_MODEL` 的旧配置，检查是否正确迁移
- [ ] **空值处理**：留空新字段，检查是否正确删除键
- [ ] **切换供应商**：切换到不同供应商，检查配置是否正确替换
- [ ] **回滚机制**：模拟写入失败（chmod 444），检查是否回滚到旧配置

### 7.4 注意事项

1. **双重归一化必须保持一致**：
   - 前端的回退逻辑（`useModelState.ts:useEffect`）
   - 后端的归一化逻辑（`provider.rs:normalize_claude_models_in_value`）
   - 两者的优先级顺序必须完全一致

2. **旧键删除时机**：
   - 仅在写入时删除 `ANTHROPIC_SMALL_FAST_MODEL`
   - 读取时不删除，而是使用其值回退填充新键

3. **空值语义**：
   - 空字符串 `""` 表示用户主动清空该字段
   - `undefined` 表示字段从未设置
   - 归一化时仅处理 `undefined`，不覆盖空字符串

4. **事务安全**：
   - 所有配置修改都通过 `run_transaction` 包装
   - 写入失败时会自动回滚内存和文件

5. **原子写入**：
   - 使用 `write_json_file` 确保原子性（临时文件 + rename）
   - 防止写入过程中崩溃导致配置损坏

---

## 附录

### A. 相关 Issue / PR

- [提 issue 链接占位]

### B. 测试用例

**前端测试**（建议添加到 `tests/hooks/useModelState.test.ts`）：

```typescript
describe("useModelState", () => {
  it("should migrate ANTHROPIC_SMALL_FAST_MODEL to DEFAULT_HAIKU", () => {
    const settingsConfig = JSON.stringify({
      env: {
        ANTHROPIC_SMALL_FAST_MODEL: "claude-haiku-3.5",
      },
    });

    const { result } = renderHook(() =>
      useModelState({
        settingsConfig,
        onConfigChange: jest.fn(),
      })
    );

    expect(result.current.defaultHaikuModel).toBe("claude-haiku-3.5");
  });
});
```

**后端测试**（建议添加到 `src-tauri/tests/provider_service.rs`）：

```rust
#[test]
fn normalize_removes_small_fast_model() {
    let mut settings = json!({
        "env": {
            "ANTHROPIC_SMALL_FAST_MODEL": "claude-haiku-3.5"
        }
    });

    let changed = ProviderService::normalize_claude_models_in_value(&mut settings);

    assert!(changed);
    assert!(settings["env"].get("ANTHROPIC_SMALL_FAST_MODEL").is_none());
    assert_eq!(
        settings["env"]["ANTHROPIC_DEFAULT_HAIKU_MODEL"],
        "claude-haiku-3.5"
    );
}
```

### C. 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Interface                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  ClaudeFormFields                                         │  │
│  │  ├─ 主模型                      (claudeModel)             │  │
│  │  ├─ Haiku 默认模型             (defaultHaikuModel)       │  │
│  │  ├─ Sonnet 默认模型            (defaultSonnetModel)      │  │
│  │  └─ Opus 默认模型              (defaultOpusModel)        │  │
│  └───────────────────────────────────────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      State Management                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  useModelState Hook                                       │  │
│  │  ├─ useState: 管理四个字段的状态                          │  │
│  │  ├─ useEffect: 初始化读取（旧键回退）                    │  │
│  │  └─ handleModelChange: 写入 env 对象，删除旧键            │  │
│  └───────────────────────────────────────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Tauri IPC Layer                            │
│  providersApi.add() / update() / switch()                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Backend Commands                           │
│  add_provider / update_provider / switch_provider               │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    ProviderService                              │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  1. normalize_provider_if_claude (归一化 #1)             │  │
│  │     └─ normalize_claude_models_in_value                   │  │
│  │        ├─ 读取: MODEL, SMALL_FAST, DEFAULT_*              │  │
│  │        ├─ 回退填充: DEFAULT_* ← SMALL_FAST ← MODEL       │  │
│  │        └─ 删除: ANTHROPIC_SMALL_FAST_MODEL                │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │  2. validate_provider_settings                            │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │  3. run_transaction                                       │  │
│  │     ├─ 插入/更新: config.providers[id]                   │  │
│  │     ├─ 保存: ~/.cc-switch/config.json                    │  │
│  │     └─ PostCommitAction                                   │  │
│  │        └─ write_live_snapshot                             │  │
│  │           └─ write_claude_live                            │  │
│  │              ├─ normalize (归一化 #2)                    │  │
│  │              └─ 写入: ~/.claude/settings.json            │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

**文档维护者**：Claude Code (AI Assistant)
**最后审核**：[待填写]
**反馈渠道**：[GitHub Issues](https://github.com/farion1231/cc-switch/issues)
