## 模式定位

CodeX 模式专注于管理 OpenAI 生态中与编码相关的配置与辅助流程，帮助你在 GPT-5 系列模型与企业版 API 之间快速切换。

## OpenAI 平台特性

**模型能力差异**
- `gpt-5-codex`：针对代码生成与重构进行了指令优化
- `o4-mini`：快速实验和调试的轻量模型
- `o4-long`：适合跨文件分析的大上下文模型

**部署环境**
- 公有云（默认 `https://api.openai.com/v1`）
- 企业私有终端（自定义 `--base-url`）
- 第三方代理（遵守数据落地策略）

## 推荐工作流

1. 使用 `ccs codex add` 为不同的 API Key 建立独立配置
2. 在 TUI 中使用 `x` 快速切换到 CodeX 模式，查看当前激活的 OpenAI 配置
3. 通过模板管理器应用 `CODEX.md` 预设，为项目提供公共的接入说明
4. 使用 `ccs codex validate`（即将推出）审查 Key、Base URL 与模型是否匹配

## 审查要点

- 所有 API Key 必须来源于安全保管的环境变量或 Secret 管理工具
- 记录实际调用的模型版本与区域，确保符合安全与合规要求
- 针对企业终端，确认 `base_url` 与私有证书链配置正确
- 对接多个环境时，善用 `category` 标签区分用途

## 最佳实践

- 将 `CODEX.md` 与 `CLAUDE.md` 并存，帮助团队清晰区分不同供应商的工作流
- 在 CI 中集成 `ccs codex switch <provider>`，确保部署阶段使用预期的配置
- 对敏感日志使用 `internal/utils/secret.go` 中的脱敏工具进行处理
- 定期执行 `ccs export --app codex`，备份所有 OpenAI 配置快照

- 编码：所有模板使用 **UTF-8 无 BOM** 编码
