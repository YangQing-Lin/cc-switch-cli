## 安全基线

此模板用于统一 CodeX 模式下的 OpenAI 安全与合规清单，适合作为项目内 `CODEX.md` 的基础段落。

### 凭证管理
- 仅通过受信任的 Secret 管理服务（如 HashiCorp Vault、AWS Secrets Manager）分发 API Key
- 为 CI/CD 创建独立 Key，并限制模型、速率与 IP 白名单
- 每次轮换 Key 后运行 `ccs codex list`，确认旧凭证已移除

### 数据分级
- 分类标注输入/输出数据，敏感内容开启 `data_redaction` 功能
- 对需要回溯的对话开启 `response_logging`，并将日志写入加密存储
- 禁止将客户数据提交给公有云模型，需使用企业私有终端或脱敏后再上传

### 调用限制
- 为高风险模型设置 `max_tokens` 与 `rate_limit` 双重限制
- 配置 `retry_policy`，确保在 429/500 错误时合理退避
- 使用 `observability` 标签上报指标到 Prometheus 或 Datadog

## 审计建议

1. 建立每周的调用量报告，核对费用与配额使用情况
2. 对所有 `base_url` 与模型 ID 进行校验，防止误指向测试环境
3. 将审计结果写入 `CODEX.local.md`，只在团队内部共享

- 编码：所有模板使用 **UTF-8 无 BOM** 编码
