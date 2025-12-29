# Codex `config.toml` 受管字段与插入策略（PCO-030）

本仓库在切换 Codex 配置时，目标是 **只更新 CCS 管理的字段**，并尽量保持原文件的 table 顺序、注释与空白不变（避免全量 Marshal 重写）。

## 受管范围（当前实现）

**Top-level keys（固定清单）**：

- `model_provider`
- `model`
- `model_reasoning_effort`
- `disable_response_storage`

**Provider table（按 `configContent` 声明）**：

- 仅对 `configContent` 中出现的 `[model_providers.<name>]` 子表做更新/补齐
- 该子表内的 key 都视为受管（但会忽略已废弃的 `env_key`）

## 插入策略（缺失字段）

- Top-level key 缺失：插入到“第一个 table header（行首为 `[`）”之前，避免落入某个 table 的上下文
- `[model_providers.<name>]` 缺失：在文件末尾追加整段 table block（保留原文件尾部的空白/换行）
- provider 子表内字段缺失：插入到该 table 的末尾（尽量不扰动其后续空白/注释）

## 参考实现

- `internal/utils/codex_toml_patch.go:1`
- `internal/config/switch.go:133`
