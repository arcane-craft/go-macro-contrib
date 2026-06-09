## 1. inline 包 syntax-id 更名

- [x] 1.1 将 `inline/stubs.go`、`inline/expand.go` 中 `//macro: syntax-inline` 改为 `//macro: inline`
- [x] 1.2 将 `inline/expand_test.go` 中 `mactest.ExpandCall` 的 syntax-id 参数由 `syntax-inline` 改为 `inline`
- [x] 1.3 执行 `go test ./inline/...` 确认通过

## 2. try 包 syntax-id 更名

- [x] 2.1 将 `try/stubs.go`、`try/expand.go` 中 `//macro: syntax-try` 改为 `//macro: try`（统一注释格式为 `//macro:` 无空格）
- [x] 2.2 将 `try/expand_test.go` 中 `mactest.ExpandCall` 的 syntax-id 参数由 `syntax-try` 改为 `try`
- [x] 2.3 执行 `go test ./try/...` 确认通过

## 3. with 包 syntax-id 更名

- [x] 3.1 将 `with/stubs.go`、`with/expand.go` 中 `//macro: syntax-with` 改为 `//macro: with`
- [x] 3.2 将 `with/expand_test.go` 中 `mactest.ExpandCall` 的 syntax-id 参数由 `syntax-with` 改为 `with`
- [x] 3.3 执行 `go test ./with/...` 确认通过

## 4. 文档与工具

- [x] 4.1 更新 `README.md` 官方宏库表格：三行 syntax-id 改为 `inline`、`try`、`with`
- [x] 4.2 更新 `.cursor/skills/openspec-viewer/generate_viewer.py` 中 `syntax-inline`/`syntax-try` 节点键名为 `inline`/`try`（若存在 `syntax-with` 一并更新）

## 5. 验收

- [x] 5.1 全仓库执行 `rg 'syntax-(inline|try|with)'`，排除 `openspec/changes/archive/` 与 `openspec/changes/rename-syntax-ids/`，确认零匹配
- [x] 5.2 执行 `go test ./...` 确认全仓测试通过
