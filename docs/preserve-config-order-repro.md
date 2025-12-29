# Preserve Config Order — Repro (PCO-010)

目标：用最小样例复现并定界“切换配置后写回文件导致顺序变化”的问题，作为后续 patch/测试的输入。

## 一键复现（推荐）

在仓库根目录运行：

```bash
go run ./test/repro/preserve_config_order
```

它会在临时目录里：

- 写入一份 **刻意设置顺序** 的 `.codex/config.toml`（包含多 table + 注释 + 空行，且 `[sandbox_workspace_write]` 在 `[model_providers]` 之前）
- 写入一份 **刻意设置 key 顺序** 的 `.claude/settings.json` / `.gemini/settings.json`（包含未知字段）
- 调用 `SwitchProviderForApp(...)` 触发写回
- 打印切换前后：
  - TOML table 顺序摘要（重点：`[sandbox_workspace_write]` vs `[model_providers]`）
  - JSON 顶层 key 顺序摘要（Claude/Gemini）

默认会保留临时目录，方便你继续用 `diff` 手动复查；如果你只想看摘要输出后清理：

```bash
go run ./test/repro/preserve_config_order --cleanup
```

## 手动复查（可选）

如果你想手动看文件内容差异，可把程序输出的临时目录路径（`tmpDir=...`）拿来对比：

```bash
diff -u <(cat "$tmpDir/.codex/config.toml.before") "$tmpDir/.codex/config.toml"
diff -u <(cat "$tmpDir/.claude/settings.json.before") "$tmpDir/.claude/settings.json"
diff -u <(cat "$tmpDir/.gemini/settings.json.before") "$tmpDir/.gemini/settings.json"
```

说明：

- “before” 文件由 repro 程序在临时目录中生成（用于对比）。
- 目前（未修复前）预期能观察到：
  - TOML 中 `[sandbox_workspace_write]` 与 `[model_providers]` 的相对顺序发生变化
  - Claude/Gemini settings.json 的顶层 key 顺序发生变化（通常变为 JSON 序列化后的排序顺序）
- 额外观测（当前实现）：Gemini settings.json 的未知字段（如 `yyy_unknown`）会在写回时丢失（因为当前写回走了默认 JSON 序列化，未触发自定义 `MarshalJSON`）。
