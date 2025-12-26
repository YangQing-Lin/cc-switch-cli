---
mode: plan
cwd: /home/zhang/tools/cc-switch-cli
task: 为项目重写并补齐单元测试，整体覆盖率达到 90%+
complexity: complex
planning_method: builtin
created_at: 2025-12-23T17:34:59+08:00
---

# Plan: 单元测试覆盖率 90%+（重写全部测试）

🎯 任务概述
本项目为 Go CLI（含 Cobra + BubbleTea TUI）。目标是在不引入集成测试的前提下，重写现有全部测试，并为所有包补齐单元测试，使 `go test` 覆盖率统计的总语句覆盖率达到 90% 以上。

📋 执行计划
1. 建立可重复的测试运行方式：补充统一的测试入口（脚本/Make 目标），固定覆盖率口径（`go test ./... -coverprofile=coverage.out` + `go tool cover -func`），并在受限环境下通过设置 `GOCACHE/GOMODCACHE` 到工作区解决缓存写入问题。
2. 建立覆盖率基线与缺口清单：跑一次全量覆盖率，输出各包覆盖率排行榜（最低优先修），并确认统计口径是“全包总覆盖率”而非单包。
3. 统一测试约定与公共工具：确定测试组织（表驱动 + 子测试 + `t.Helper()`），用 `t.TempDir()`/`t.Setenv()` 隔离文件系统与 HOME；扩展 `internal/testutil` 用于输出捕获、临时目录、跨平台跳过。
4. 重写现有测试文件（全部 10 个）：逐包替换现有 `*_test.go`（含 `test/integration/basic_test.go`），修正当前“未真正验证行为”的测试（如 portable 相关），补齐断言、边界与错误路径。
5. 补齐“核心业务包”单测：重点覆盖 `internal/config`（CRUD/迁移/校验/MCP）、`internal/utils`（原子写/备份/错误处理）、`internal/template`（内置模板加载/用户模板持久化/Apply/Diff）等，优先把高调用路径覆盖到 90%+。
6. 补齐“系统交互包”单测：新增 `internal/claude`（HOME 下读写与幂等）、`internal/lock`（锁获取/陈旧锁/Touch/GetPID）、`internal/testutil` 自身测试；portable 模式通过在测试二进制目录创建 `portable.ini` 进行可控验证。
7. 为 TUI 建立可测切面并补齐单测：围绕 `Model.Update/View` 构造纯单元测试，通过模拟 BubbleTea 消息与按键覆盖模式切换、列表/表单状态机、模板/备份/MCP 子流程渲染；对外部依赖（更新检查/下载、模板文件路径、时间）必要时做轻量依赖注入或通过环境/transport stub 限制副作用。
8. 为 CLI `cmd/` 建立可测切面并补齐单测：将 `rootCmd`/子命令改为可构造的 command factory（避免全局状态污染），并对副作用（启动 TUI、打开文件、更新下载）做可替换函数/接口；用 Cobra 的 `ExecuteC` + buffer 捕获 stdout/stderr 覆盖各命令的成功/失败路径。
9. 为 `internal/version`（更新/安装）补齐“安全可测”的单测：HTTP 请求用自定义 `http.RoundTripper` stub 断网可测；压缩包解析/平台名推断/错误信息覆盖到位；对会写回自身二进制的路径增加可注入目标路径或保护开关，避免测试环境自毁。
10. 覆盖率门禁与 CI：新增覆盖率阈值检查（<90% 直接失败），并在 GitHub Actions 增加测试工作流（多平台可选），上传 `coverage.out`/HTML 报告；同步更新 `docs/testing.md` 以匹配新的测试入口与统计口径。

⚠️ 风险与注意事项
- `cmd/root.go` 的 `startTUI()` 直接 `tea.NewProgram(...).Run()` 会阻塞，必须通过依赖注入/可替换 runner 才能单测覆盖。
- `internal/version` 的自更新逻辑会操作当前可执行文件，单测必须避免真实安装路径写入（需要保护开关或注入目标路径）。
- TUI 代码量较大且含外部依赖（模板文件、更新检查），若不做切面会导致覆盖率难以快速提升。
- 当前 Codex sandbox 下 Go module/cache 默认路径不可写，需在测试入口中显式设置 `GOMODCACHE`/`GOCACHE` 到工作区（或启用 vendoring）。

📎 参考
- `docs/testing.md:1`
- `cmd/root.go:23`
- `cmd/root.go:153`
- `internal/tui/tui.go:115`
- `internal/version/version.go:18`
- `internal/template/manager.go:23`
- `internal/claude/plugin.go:21`
- `internal/lock/lock.go:34`