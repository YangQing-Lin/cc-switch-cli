# 补充功能缺口表（基于 2025-12-23 的 10 条补充项）

来源：
- `plan/2025-12-23_17-34-59-unit-tests-90-coverage.md:1`
- `plan/2025-12-26_13-47-31-unit-tests-review-gapfill.md:1`

说明：
- 状态枚举：已完成 / 部分完成 / 缺失
- `refs`：使用 `file:line`（可跳转到真实文件）
- `验收命令`：可直接复制运行；如需隔离 HOME，可在命令前加 `HOME=$(mktemp -d) USERPROFILE=$HOME`

| # | 补充项（边界合同） | 状态 | refs | 验收命令 |
|---:|---|---|---|---|
| 1 | 统一测试入口 + 受限环境 cache 隔离（固定 `GOPATH/GOCACHE/GOMODCACHE` 到工作区 `.cache/`） | 已完成 | `test.sh:1; test.bat:1; docs/testing.md:12` | `./test.sh` |
| 2 | 覆盖率基线与缺口清单：能输出 total 与各包覆盖率，并确认统计口径为 total（statements） | 部分完成 | `docs/testing.md:251; docs/testing.md:283; test.sh:1` | `./test.sh && go tool cover -func=coverage.out | grep total` |
| 3 | 统一测试约定与公共工具：`t.TempDir/t.Setenv` 隔离 + `internal/testutil` 提供可复用 helper | 已完成 | `internal/testutil/testutil.go:86; internal/testutil/testutil.go:250; internal/testutil/testutil_test.go:262` | `go test ./internal/testutil -run TestBubbleTeaTestHelper -v` |
| 4 | 重写/补齐现有测试文件：新增断言与错误路径覆盖（避免“只跑不验”） | 已完成 | `internal/tui/coverage_sweep_test.go:1; internal/version/version_test.go:1; cmd/root_test.go:217` | `go test ./...` |
| 5 | 核心业务包单测补齐：`internal/config`/`internal/utils`/`internal/template` 等关键路径可测 | 已完成 | `internal/config/config_test.go:1; internal/utils/file_test.go:1; internal/template/manager_test.go:1` | `go test ./internal/config ./internal/utils ./internal/template` |
| 6 | 系统交互包单测补齐：`internal/claude`/`internal/portable`/`internal/version` 等避免真实 HOME/网络依赖 | 已完成 | `internal/claude/plugin.go:21; internal/portable/portable.go:8; internal/version/version_test.go:361` | `go test ./internal/claude ./internal/portable ./internal/version` |
| 7 | TUI 可测切面与断言：`Model.Update/View` 覆盖关键模式与更新逻辑，避免网络副作用 | 已完成 | `internal/tui/tui.go:169; internal/tui/portable.go:9; internal/tui/tui_test.go:145` | `HOME=$(mktemp -d) USERPROFILE=$HOME go test ./internal/tui -count=20` |
| 8 | CLI `cmd/` 可测切面：子命令成功/失败路径可通过 Cobra 执行并断言输出 | 已完成 | `cmd/root_test.go:217; cmd/version_test.go:1` | `go test ./cmd -run TestRootCommandDirFlagParsing -v` |
| 9 | `internal/version`（更新/安装）可测：HTTP stub + 压缩包解析/平台验证/错误信息覆盖 | 已完成 | `internal/version/version.go:103; internal/version/version_test.go:513; internal/version/version_test.go:672` | `go test ./internal/version -run TestExtractBinary -v` |
| 10 | 覆盖率门禁与 CI：GitHub Actions 多平台测试 + Coverage Gate 阈值检查 | 部分完成 | `.github/workflows/test.yml:11; .github/workflows/test.yml:105; docs/testing.md:283` | `rg -n \"COVERAGE_THRESHOLD\" .github/workflows/test.yml` |

抽查建议（回归抽样 3 条）：
1. 运行 `./test.sh` 并检查 `go tool cover -func=coverage.out | grep total` 有输出；
2. 运行 `HOME=$(mktemp -d) USERPROFILE=$HOME go test ./internal/tui -count=20`；
3. 运行 `rg -n "COVERAGE_THRESHOLD" .github/workflows/test.yml` 并确认阈值为 90。
