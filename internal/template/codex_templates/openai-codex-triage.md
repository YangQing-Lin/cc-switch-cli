## 诊断流程

当 CodeX 模式出现请求失败或生成质量下降时，可将此模板应用到 `CODEX.md` 或工单文档作为标准排查步骤。

### 1. 快速定位
- 确认 `ccs codex switch` 是否指向期望的供应商与模型
- 通过 `ccs show --app codex <name>` 检查 `base_url` 与 `model` 字段
- 查看 `~/.codex/CODEX.md` 是否同步更新了最新的接入指引

### 2. 请求级诊断
- 捕获最近三次失败请求的 `request_id` 与 `status`
- 核对 `temperature`、`max_output_tokens` 等生成参数是否异常
- 对比成功请求与失败请求的上下文长度，识别截断或超限问题

### 3. 服务端确认
- 检查 OpenAI 状态页与内部监控告警
- 对企业终端，确认代理或私有反向代理未失效
- 校验 TLS 证书与自签 CA 在目标机器上是否可用

### 4. 修复建议
- 调整 `fallback_model`，为关键调用配置降级策略
- 对多区域部署，使用 `region` 标签测试备份终端
- 将改动记录到 `CODEX.md` 的「运行手册」章节，保持团队同步

- 编码：所有模板使用 **UTF-8 无 BOM** 编码
