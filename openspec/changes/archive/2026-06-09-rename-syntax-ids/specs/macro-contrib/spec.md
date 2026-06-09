## MODIFIED Requirements

### Requirement: 官方宏库路径

官方宏库 MUST 仅通过下列 import 路径提供：

| 包目录 | syntax-id（典型） | import |
|--------|-------------------|--------|
| `inline` | `inline` | `github.com/arcane-craft/go-macro-contrib/inline` |
| `try` | `try` | `github.com/arcane-craft/go-macro-contrib/try` |
| `with` | `with` | `github.com/arcane-craft/go-macro-contrib/with` |
| `derive` | `derive` | `github.com/arcane-craft/go-macro-contrib/derive` |
| `wirejson` | `wire-json` | `github.com/arcane-craft/go-macro-contrib/wirejson` |

`go-macro` 根 module MUST NOT 再包含 `inline/`、`try/`、`with/`、`contrib/`、`syntax/` 或上述包的副本。

#### Scenario: import derive

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/derive`
- **THEN** MUST 解析到 `go-macro-contrib` 仓库的 `derive` 包

#### Scenario: import with

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/with`
- **THEN** MUST 解析到 `go-macro-contrib` 仓库的 `with` 包

### Requirement: Call 与 Decl Expander 可选用 macro/quote

`try`、`with` 与 `derive` 包的 Expander 实现 **MAY** 使用 `github.com/arcane-craft/go-macro/macro/quote` 进行模板化 AST 组装。该选用 **MUST NOT** 改变 `try`、`with` 或 `syntax-derive` 已定义的展开语义、`mactest` 可观测结果或公开桩 API。

使用 `macro/quote` 的 Call Expander **MUST** 在返回 `CallExpandResult` 前对语句结果调用 `macro.StampStmtPos(ctx.MacroPos(), stmts)`（与手写 AST 相同）。Quote **MUST NOT** 代替 Expander 设置 `SpliceTarget`。

#### Scenario: try 行为与迁移前一致

- **WHEN** 在本地 `replace` 到含 `macro/quote` 的 `go-macro` 后，对 `try` 包执行 `mactest.ExpandCall` 既有用例
- **THEN** 展开结果的 `Target`、`Stmts` 形状与语义 **MUST** 与迁移前一致

#### Scenario: derive 行为与迁移前一致

- **WHEN** 在本地 `replace` 到含 `macro/quote` 的 `go-macro` 后，对 `derive` 包执行 `mactest.ExpandDecl` 既有用例（含 `Derive[fmt.Stringer]` 嵌入）
- **THEN** 展开结果的 `Fields`、`Methods`（含生成的 `String()`）**MUST** 与 `derivestringer` 时代语义一致
