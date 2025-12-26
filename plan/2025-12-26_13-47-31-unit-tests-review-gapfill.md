---
mode: plan
cwd: /home/luke/ProjectGo/cc-switch-cli
task: 单元测试工作代码评审 + 修复缺口 + 对齐 90%+ 覆盖率计划
complexity: complex
planning_method: builtin
created_at: 2025-12-26T13:47:31
---

# Plan: 单元测试评审/修复 + 补齐缺口（对齐 90%+）

🎯 任务概述
当前仓库已按 `.claude/specs/unit-tests/dev-plan.md` 与 `.claude/specs/unit-tests/fix-plan.md` 完成了一轮单测重写与修复，但仍需做一次面向“可维护性 + 覆盖率门禁 + 跨平台稳定性”的代码评审与补齐。
同时需要对照 `plan/2025-12-23_17-34-59-unit-tests-90-coverage.md`（注意该文件名末尾存在空格；可用 `ls -lb plan` 观察到 `...md\ `）中的补充项，判断仓库缺失点并补上，最终把“总覆盖率门禁”对齐到 **90%+**。

📋 执行计划
1. 统一可复现的测试入口与环境（本地 + sandbox）：新增 `test.sh`（必要时补 `test.bat`）作为统一入口；在入口中将 `GOPATH/GOCACHE/GOMODCACHE` 指向工作区（例如 `.cache/`）以规避受限环境写入 `~/.cache`、`~/go/pkg/sumdb` 的权限问题，并在文档中说明。
2. 代码评审回放（fix-plan）：按 `.claude/specs/unit-tests/fix-plan.md` 的 P0/P1/P2 逐条抽查关键变更是否消除根因且无全局副作用（重点关注 `cmd/version_test.go` 的 HTTP client 注入、`internal/testutil` 的 stdout/stderr 捕获与 cwd 切换副作用、TUI sweep 测试断言质量、跨平台权限类测试的 skip 策略）。
3. 建立“补充功能缺口表”：以 `plan/2025-12-23_17-34-59-unit-tests-90-coverage.md` 的 10 条为基准，输出一张表（已完成/部分完成/缺失），每条给出对应文件位置与验收命令。
4. 覆盖率基线与目标：在统一入口下跑 `go test ./... -coverprofile=coverage.out -covermode=count`，记录 total 与各包覆盖率；将目标设为 **total ≥ 90.5%**（留 buffer），同时明确“总覆盖率”与“关键包覆盖率”的双指标。
5. 覆盖率补齐（优先级 A：收益最高的包）：补齐 `cmd`、`internal/utils`、`internal/testutil` 的低覆盖路径（当前基线示例：`cmd`≈77.9%、`internal/utils`≈78.8%、`internal/testutil`≈79.2%）；策略优先增加“错误分支/边界”测试，必要时做最小可测切面（依赖注入/可替换函数），保证 CLI 行为向后兼容。
6. 覆盖率补齐（优先级 B：系统交互与版本）：补齐 `internal/claude`、`internal/portable`、`internal/version`（基线示例：≈80.3/86.7/89.5）；避免依赖真实 HOME、真实可执行文件写入与真实网络，统一用 `t.TempDir()`、`t.Setenv()`、stub HTTP client。
7. 覆盖率补齐（优先级 C：TUI 小幅拉升）：为 `internal/tui` 增补少量“状态断言”用例，把 89.x% 拉过 90；优先修正 sweep 类测试“只跑不验”的路径，尽量不引入大规模重构。
8. 门禁对齐与 CI：把 `.github/workflows/test.yml` 的 `COVERAGE_THRESHOLD` 从当前值（现为 89）提升到 90（或采用 90.5 取整策略），并验证 Linux/Mac/Windows 三平台测试稳定。
9. 文档同步：更新 `docs/testing.md` 中的测试入口、覆盖率门禁说明与“当前覆盖率”信息（避免与真实数据漂移），补充受限环境下 cache 配置的说明。

⚠️ 风险与注意事项
- 在当前环境中直接运行 `go test ./...` 可能因默认 cache/sumdb 路径不可写而失败；必须通过脚本或环境变量显式设置 `GOPATH/GOCACHE/GOMODCACHE` 到工作区。
- 提升 `cmd/`、TUI 覆盖率可能需要少量依赖注入；要严格控制改动面，避免破坏现有 CLI/TUI 行为与用户配置路径逻辑。
- 权限失败类测试在 Windows/root 环境可能不稳定；优先使用更可控的失败注入或能力探测后 `t.Skip`。

📎 参考
- `.claude/specs/unit-tests/dev-plan.md:1`
- `.claude/specs/unit-tests/fix-plan.md:1`
- `plan/2025-12-23_17-34-59-unit-tests-90-coverage.md :1`（文件名末尾空格）
- `.github/workflows/test.yml:1`
- `docs/testing.md:1`
