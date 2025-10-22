# CODEX.md

This file guides CodeX / OpenAI GPT-based coding agents when contributing to this repository.

## 项目聚焦

cc-switch-cli 在 CodeX 模式下负责管理 OpenAI CLI、企业终端以及自建代理的多套配置。目标是快速切换 `gpt-5-codex`、`o4-mini` 等模型，同时维持凭证分级与调用审计。

## 快速开始

```bash
# 构建并体验 CodeX 模式
go build -o ccs
./ccs codex list

# 添加新的 OpenAI 配置
./ccs codex add prod-openai \
  --apikey sk-... \
  --base-url https://api.openai.com/v1 \
  --model gpt-5-codex \
  --category production

# 切换到指定配置
./ccs codex switch prod-openai

# 导出全部 CodeX 配置以备份
./ccs export --app codex > codex-backup.json
```

## 配置结构

- **主配置**：`~/.cc-switch/config.json` 中 `apps.codex` 节点
- **OpenAI Live 文件**：
  - `~/.codex/config.toml` — 模型与终端参数
  - `~/.codex/api.json` — API Key 与速率限制
  - `~/.codex/CODEX.md` — 团队内部的运行手册（可由模板生成）

## 开发展示面板

- 在 TUI 中按 `x` 进入 CodeX 模式，列表显示当前所有 OpenAI 配置
- 通过 `m` 打开模板管理器，使用 CodeX 专属模板生成指导文档
- 应用模板前会自动备份目标文件，方便回滚

## OpenAI 最佳实践

1. **凭证安全**：所有 Key 必须存放于 Secret 管理服务，并通过 `ccs codex update` 定期轮换
2. **模型适配**：根据任务类型选择 `gpt-5-codex`、`o4-mini` 或 `o4-long`，避免一把梭
3. **审计追踪**：调用日志写入内部审计平台，记录模型、区域、请求速率
4. **多环境管理**：使用 `category` 字段标记 `dev`、`staging`、`prod`，结合 `ccs codex list` 快速筛选
5. **团队协作**：将结果写入 `CODEX.md` 与 `CODEX.local.md`，分别对应团队共享与个性化配置

## 调试指引

- `ccs validate --app codex`（未来版本）将校验 Base URL、模型和凭证是否匹配
- 当前可通过 `ccs show --app codex <name>` 手动核对所有字段
- 使用 `--trace` 选项运行 CLI，可打印 OpenAI 请求与响应示例（敏感字段会自动脱敏）

## 贡献提示

- 新增 OpenAI 功能时，请同步更新 `docs/cli-user-manual.md` 与相关测试
- 若引入新的模型别名，将其添加至 `internal/config/provider.go` 的默认映射
- 按照项目规范提交 conventional commit 信息：`feat: support o4-long fallback`

- 编码：所有文档及模板需使用 **UTF-8 无 BOM** 编码
