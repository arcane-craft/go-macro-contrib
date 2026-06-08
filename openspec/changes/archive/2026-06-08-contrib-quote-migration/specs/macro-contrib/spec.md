## ADDED Requirements

### Requirement: Call 与 Decl Expander 可选用 macro/quote

`try` 与 `derivestringer` 包的 Expander 实现 **MAY** 使用 `github.com/arcane-craft/go-macro/macro/quote` 进行模板化 AST 组装。该选用 **MUST NOT** 改变 `syntax-try` 或 `syntax-derive-stringer` 已定义的展开语义、`mactest` 可观测结果或公开桩 API。

使用 `macro/quote` 的 Call Expander **MUST** 在返回 `CallExpandResult` 前对语句结果调用 `macro.StampStmtPos(ctx.MacroPos(), stmts)`（与手写 AST 相同）。Quote **MUST NOT** 代替 Expander 设置 `SpliceTarget`。

#### Scenario: try 行为与迁移前一致

- **WHEN** 在本地 `replace` 到含 `macro/quote` 的 `go-macro` 后，对 `try` 包执行 `mactest.ExpandCall` 既有用例
- **THEN** 展开结果的 `Target`、`Stmts` 形状与语义 **MUST** 与迁移前一致

#### Scenario: derivestringer 行为与迁移前一致

- **WHEN** 在本地 `replace` 到含 `macro/quote` 的 `go-macro` 后，对 `derivestringer` 包执行 `mactest.ExpandDecl` 既有用例
- **THEN** 展开结果的 `Fields`、`Methods`（含生成的 `String()`）**MUST** 与迁移前一致

### Requirement: quote 联调与已提交 go.mod 约束

开发者在本地联调 `macro/quote` 时 **MAY** 向 `go-macro-contrib/go.mod` 添加 `replace github.com/arcane-craft/go-macro => ../go-macro`。已提交（发布/tag 前）的 `go.mod` **MUST** 继续 `require` 已发布的 `go-macro` semver tag，且 **MUST NOT** 包含 `replace` 指令，直至核心发布含 `macro/quote` 的版本并 bump `require`。

#### Scenario: 本地 replace 解析 quote

- **WHEN** 开发者添加 `replace` 指向 sibling `go-macro` 且该核心含 `macro/quote`
- **THEN** `go test ./...` **MUST** 能解析 `github.com/arcane-craft/go-macro/macro/quote`

#### Scenario: 已提交 go.mod 无 replace

- **WHEN** 查看发布前已提交的 `go-macro-contrib/go.mod`
- **THEN** **MUST NOT** 含 `replace github.com/arcane-craft/go-macro`
