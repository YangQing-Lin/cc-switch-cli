# Review: Preserve Config Order (PCO-080)

本次变更目标：切换配置时，尽量对 live 配置做“最小写回”，避免 TOML/JSON 因全量序列化导致的块/字段重排、注释/空白丢失，且保持原子写入与向后兼容。

## 变更概览

- Codex：`config.toml` 改为就地 patch（仅更新受管字段），保留未受管 table 的相对顺序、注释与空白。
- Claude：`settings.json` 改为最小 JSON patch（主要更新 `env`/`model`），保持未受管顶层 key 的相对顺序不变。
- Gemini：`settings.json` 改为最小 JSON patch（更新 `security`），并修复未知字段丢失问题；保持未受管顶层 key 的相对顺序不变。
- 公共工具：JSON/TOML patch 能力收敛到 `internal/utils`，三端写回统一调用。

## Review Checklist（重点项）

- 受管字段边界
  - Codex：仅更新固定 top-level 清单 + `configContent` 声明的 `model_providers.<name>` 子表；其余字段/表保持原样。
  - Claude：仅 patch `env`/`model`（受管）；其他顶层字段（含未知字段、`statusLine` 等）不动。
  - Gemini：仅 patch `security`（受管）；`mcpServers`/未知字段默认不动（缺失/为 null 时补 `{}` 以保持旧行为）。

- 回退策略（失败时行为可预期）
  - Codex：TOML parse/patch 失败回退为完全覆盖写入 `configContent`（避免“读→重排→写回”）。
  - Claude/Gemini：JSON patch 失败回退为旧逻辑整体写回，保证可用性；对无效 JSON 的处理路径明确。

- 原子写入与权限
  - 仍使用 `AtomicWriteFile`，并在 patch 写回时使用 `perm=0` 继承原文件权限（避免权限意外变化）。
  - Windows 特殊路径：`AtomicWriteFile` 的 Windows 分支（先删后改名、chmod 容错）仍保留。

- 向后兼容与 CLI 行为
  - CLI 命令/参数未调整；变更集中在 live 配置写回路径，且保留原子写入与错误包装。

## 证据与回归

- 已执行：`go fmt ./...`、`go test ./...`、`go build`（本机通过）。
- 顺序不变断言：已在 `internal/config/config_test.go` 覆盖 Codex/Claude/Gemini，并包含多次写回稳定性用例。
- 真实样例 diff：通过 repro 工具 `test/repro/preserve_config_order` 验证切换前后“仅目标字段变化 + 顺序保持”。

## 风险与后续建议

- JSON/TOML patch 属于“按文本定位”的实现：对极端格式/非常规 JSON（例如极端压缩/奇怪空白）理论上存在 patch 失败回退路径；但测试已覆盖主流程与至少一个回退场景。
- Windows 实机回归未在本环境执行：已通过 CRLF 测试与代码路径审阅覆盖主要风险点；建议在 Windows 上再跑一次 `go test ./...` 做最终确认。

