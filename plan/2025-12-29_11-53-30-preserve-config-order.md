---
mode: plan
cwd: /home/luke/ProjectGo/cc-switch-cli
task: 切换配置时保持未管理配置块/段落顺序不变（Codex/Claude/Gemini）
complexity: medium
planning_method: builtin
created_at: 2025-12-29T11:53:30+08:00
---

# Plan: 切换配置保持块顺序不变

🎯 任务概述
当前切换 Provider 会重写 live 配置文件（Codex 的 `config.toml`、Claude/Gemini 的 `settings.json`），由于走了 map/struct → Marshal 的序列化路径，导致未被 CCS 管理的段/块在输出时被重新排序。
目标是在“只更新 CCS 管理字段”的前提下，尽量对原文件做就地 patch，保留其原有的块顺序、空白与注释（尤其是 TOML）。

📋 执行计划
1. **复现与定界**：构造最小输入样例（含 `[sandbox_workspace_write]` 与 `[model_providers]` 的顺序），用当前 `writeCodexConfig` 跑一次确认顺序变化；同样用 Claude/Gemini 的 `settings.json` 复现 key 顺序变化。
2. **Codex：改为 TOML 就地更新**：在 `internal/config/switch.go` 的 `writeCodexConfig` 中，避免 `map` 合并后 `toml.Marshal` 整体重写；改用 go-toml `unstable.Parser` 扫描原始 TOML，定位并替换“受管 KeyValue”的 value 字节区间，确保未相关 table 块顺序完全不动。
3. **Codex：受管范围明确化**：把“受管字段列表/规则”收敛到单处（例如 `model_provider/model/model_reasoning_effort/disable_response_storage` + 指定的 `model_providers.<name>` 子表），并明确“缺失字段的插入位置策略”（例如插到第一个 table 前、或插到 `[model_providers]` 相关段末尾）。
4. **Claude：JSON patch 写入**：在 `writeClaudeConfig` 中避免 `utils.WriteJSONFile`（它会通过 `MarshalIndent` 导致 key 排序变化）；改为读取原始 JSON 字符串后仅用 JSON patch 更新受管字段（如 `env.*` 与 `model`），保持其他块原样。
5. **Gemini：JSON patch 写入**：同样在 `writeGeminiSettingsFile` 里避免整体 Marshal；只 patch `security.auth.selectedType`，保留 `mcpServers` 与未知字段原位置。
6. **抽象公共工具**：新增 `internal/utils/jsonpatch` 与 `internal/utils/tomlpatch`（或同目录等价文件），统一处理：读取 →（可选）解析校验 → 按 patch 规则更新 → 原子写入（继续用 `AtomicWriteFile`）。
7. **补测试（关注“顺序不变”而非仅语义）**：
   - Codex：在 `internal/config/config_test.go` 增加用例，断言写回内容中 `[sandbox_workspace_write]` 仍位于 `[model_providers]` 之前（用 `strings.Index` 比较）。
   - Claude/Gemini：增加用例，断言未管理顶层 key 的相对顺序不变（同样用 `Index`），并验证只改动了目标字段值。
8. **回归验证与评审闭环**：按仓库规范执行 `go fmt ./...`、`go test ./...`、`go build`；然后把“初始需求 + 关键实现点 + 测试点”发给 Codex 做代码评审，根据评审意见修正或收尾。

⚠️ 风险与注意事项
- `github.com/pelletier/go-toml/v2/unstable` 属于不保证兼容的 API：需要用尽量薄的一层封装，并在测试里覆盖典型 TOML（带注释/空行/多 table）以降低升级风险。
- TOML/JSON 文件可能包含无效内容：需要设计回退策略（例如“解析失败则按原行为完全覆盖”或“解析失败直接写入 provider 给定的原始 configContent”），同时要避免再次引入“顺序变化”。
- CRLF/LF、文件末尾换行、注释保留：TOML patch 以字节区间替换为主，尽量不触碰其他区域；JSON patch 也要尽量原地替换，避免全量 prettify。

📎 参考
- `internal/config/switch.go:133`
- `internal/config/switch.go:206`
- `internal/config/gemini.go:268`
- `internal/utils/file.go:18`
- `internal/config/types.go:153`
- `internal/config/config_test.go:1348`
