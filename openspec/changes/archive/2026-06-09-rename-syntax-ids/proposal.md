## Why

当前 `inline`、`try`、`with` 三个官方宏包的 syntax-id 使用 `syntax-` 前缀（`syntax-inline`、`syntax-try`、`syntax-with`），与同仓库中 `derive`（`derive`）、`wirejson`（`wire-json`）的命名风格不一致。将 syntax-id 简化为与包名一致的 `inline`、`try`、`with`，可降低认知负担、统一注册表标识，并与包 import 路径一一对应。

## What Changes

- **BREAKING**：`//macro:` 指令中的 syntax-id 由 `syntax-inline` → `inline`、`syntax-try` → `try`、`syntax-with` → `with`
- 更新 `inline/`、`try/`、`with/` 包内所有桩函数与 Expander 上的 `//macro:` 标注
- 更新 `mactest.ExpandCall` 测试中的 syntax-id 参数
- 更新 `README.md` 官方宏库表格
- 更新 OpenSpec 规格文档中所有 syntax-id 引用

展开语义、桩 API、Expander 函数签名 **不变**；仅标识符更名。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `macro-contrib`：官方宏库路径表中三包的 syntax-id 列更新为新名称
- `syntax-inline`：所有 `syntax-inline` 标识符引用改为 `inline`
- `syntax-try`：所有 `syntax-try` 标识符引用改为 `try`
- `syntax-with`：所有 `syntax-with` 标识符引用改为 `with`

## Impact

- **代码**：`inline/stubs.go`、`inline/expand.go`、`inline/expand_test.go`；`try/stubs.go`、`try/expand.go`、`try/expand_test.go`；`with/stubs.go`、`with/expand.go`、`with/expand_test.go`
- **文档**：`README.md`；`openspec/specs/` 下上述四个 spec 文件
- **工具**：`.cursor/skills/openspec-viewer/generate_viewer.py` 中的 spec 依赖图节点名（若仍引用旧 id）
- **外部**：依赖旧 syntax-id 的用户宏主文件需同步更新 `//macro:` 标注（**BREAKING**）
- **go-macro 核心**：无代码变更；`cmd/macro expand` 按新 syntax-id 发现 provider 即可
