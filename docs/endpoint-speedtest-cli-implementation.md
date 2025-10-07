# CLI 端点测速功能实现计划

## 功能目标

为cc-switch-cli添加端点测速与管理功能，与GUI v3.5.0对齐。

## Linus式设计原则

### 数据结构：消除复杂性
- Provider.Meta字段可选，向后兼容
- 端点URL作为map键，自动去重
- 仅存储必要信息：URL、添加时间、最后使用时间

### 简化CLI实现
CLI不需要GUI的实时更新和复杂交互：
- 测速：直接打印结果表格
- 管理：CRUD命令，无状态管理
- 无热身请求（CLI场景不需要）

## 实现步骤

### 1. 扩展数据结构 (internal/config/types.go)
```go
type CustomEndpoint struct {
    URL      string `json:"url"`
    AddedAt  int64  `json:"addedAt"`
    LastUsed *int64 `json:"lastUsed,omitempty"`
}

type ProviderMeta struct {
    CustomEndpoints map[string]CustomEndpoint `json:"custom_endpoints,omitempty"`
}

// 在Provider中添加:
Meta *ProviderMeta `json:"meta,omitempty"`
```

### 2. 创建测速模块 (internal/speedtest/speedtest.go)
```go
type EndpointLatency struct {
    URL     string
    Latency time.Duration
    Status  int
    Error   string
}

func TestEndpoints(urls []string, timeout time.Duration) []EndpointLatency
```

### 3. 添加endpoint命令组 (cmd/endpoint.go)
```bash
ccs endpoint test <provider-id>     # 测速所有端点
ccs endpoint add <provider-id> <url>   # 添加自定义端点
ccs endpoint remove <provider-id> <url> # 删除端点
ccs endpoint list <provider-id>     # 列出所有端点
```

### 4. 与GUI的差异
| 功能 | GUI | CLI |
|------|-----|-----|
| 测速 | 实时更新UI | 打印表格 |
| 添加端点 | Modal输入 | 命令行参数 |
| 热身请求 | 有 | 无（简化） |
| 自动选择 | 支持 | 可选flag |
| 颜色编码 | 蓝色主题 | 终端颜色 |

## 文件清单

- [x] docs/endpoint-speedtest-cli-implementation.md (本文件)
- [ ] internal/config/types.go (添加Meta字段)
- [ ] internal/speedtest/speedtest.go (测速逻辑)
- [ ] cmd/endpoint.go (端点管理命令)
- [ ] docs/feature-implementation-tracking.md (更新进度)
- [ ] README.md (添加功能说明)
- [ ] docs/cli-user-manual.md (完整手册)

## 下一步

由于这是一个较大的功能，建议：
1. 先确认功能范围是否合理
2. 是否需要完整实现GUI的所有功能
3. CLI用户最需要哪些功能

**当前建议：暂不实现**
理由：
- CLI用户通常手动配置base_url，不需要频繁测速
- 测速功能更适合GUI的可视化场景
- CLI应保持简洁，避免过度功能

如果确实需要，建议仅实现基础测速命令：
```bash
ccs speedtest <url1> <url2> <url3>  # 简单测速工具
```
