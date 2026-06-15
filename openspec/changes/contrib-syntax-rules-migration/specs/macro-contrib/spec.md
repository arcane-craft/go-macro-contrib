## MODIFIED Requirements

### Requirement: 官方宏库与 cmd/macro expand 集成

`go-macro-contrib` MUST NOT 提供 `register` 包。官方宏库应由 `cmd/macro expand` 基于 provider 上的 `//macro:` 指令自动发现并链接：每个 syntax-id 对应单一统一 `macro.Expander`（`var XxxExpander`），经 `expandtool.Register(syntaxID, XxxExpander)` 注册。

`go-macro-contrib` MUST NOT 提供 `Main`/`Run` 作为用户 expand 入口（该职责在 `cmd/macro` / `macro/expandtool`）。

#### Scenario: 通过 expand 自动 link Call 宏

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/try` 并执行 `cmd/macro expand`
- **THEN** 展开 MUST 成功链接 `TryExpander`，且无需 blank import `register`

#### Scenario: 通过 expand 自动 link Decl 宏

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/wirejson` 且 struct 匿名嵌入 `wirejson.WireJSON`，并执行 `cmd/macro expand`
- **THEN** 展开 MUST 成功链接 `WireJSONExpander`（syntax-id `wire-json`）

### Requirement: contrib 独立测试

`go-macro-contrib` 仓库 MUST 具备独立 `go test ./...`，含 `inline`、`try`、`with` 的 `mactest.ExpandSyntax` 单测，以及 `derive`、`wirejson` 的 `mactest.Expand` 单测。

#### Scenario: contrib 测试不依赖 go tool macro expand

- **WHEN** 在 `go-macro-contrib` 仓库根执行 `go test ./...`
- **THEN** MUST 通过

### Requirement: contrib 依赖 go-macro 核心版本

`go-macro-contrib` 的**已提交** `go.mod` **MUST** `require` 已发布的 `github.com/arcane-craft/go-macro` 版本（semver tag，非 `v0.0.0` 占位）；**MUST NOT** 在已提交 `go.mod` 中包含指向 sibling 目录的 `replace` 指令。

所 pin 的 `go-macro` 版本 MUST 提供：**统一 Expander API**（`macro.Context`、`macro.Syntax`、`macro.Expander`、`macro.SyntaxRules`、`macro.SyntaxCase`、`macro.Quote`）及 `expandtool.Register`。

README **MUST** 注明最低兼容核心版本，且所述版本 **MUST** 与 `go.mod` 中 pin 的 `require` 一致。

本地联调时，contrib 仓库 **SHOULD** 与 `go-macro` 位于同级目录。开发者 **MAY** 在本地向 `go-macro-contrib/go.mod` 添加 `replace github.com/arcane-craft/go-macro => ../go-macro` 以联调未发布的核心变更；该 `replace` **MUST NOT** 作为发布/tag 前提交态的硬性要求。

#### Scenario: 独立仓可解析核心依赖

- **WHEN** 在仅 clone `go-macro-contrib`、已提交 `go.mod` 无 `replace`，且模块代理可解析所 pin 的 `go-macro` tag 时执行 `go test ./...`
- **THEN** MUST 成功解析 `github.com/arcane-craft/go-macro` 模块依赖并通过测试

#### Scenario: 本地 replace 联调核心

- **WHEN** 开发者在 `go-macro-contrib` 本地添加 `replace github.com/arcane-craft/go-macro => ../go-macro` 后执行 `go test ./...`
- **THEN** MUST 使用 sibling 核心源码解析依赖（联调行为；不要求写入已提交 `go.mod`）

## REMOVED Requirements

### Requirement: Call 与 Decl Expander 可选用 macro/quote

**Reason**: `go-macro` 已删除 `macro/quote` 子包，模板化 AST 统一为 `macro.Quote` + `SyntaxCase` / `SyntaxRules`。

**Migration**: Expander 实现 MUST 使用 `macro.Quote`；Call 宏在产出语句后 MUST 调用 `macro.StampStmtPos(site.MacroPos(), stmts)`。贴回由 pattern match 产出的 Plan 经引擎 `Apply` 完成，作者 MUST NOT 设置 `SpliceTarget`。

### Requirement: quote 联调与已提交 go.mod 约束

**Reason**: 合并入 syntax-rules 迁移；联调对象由 `macro/quote` 变为整个 syntax-rules 核心。

**Migration**: 本地 `replace` 指向含 `macro.SyntaxCase` 的 `go-macro`；发布前 bump `require` 至含 syntax-rules 的 tag。

## ADDED Requirements

### Requirement: 官方宏 Expander 统一为 SyntaxCase

`inline`、`try`、`with`、`derive`、`wirejson` 各包 MUST 导出 `//macro:` 标注的 `var XxxExpander macro.Expander`，其实现 MUST 为 `macro.SyntaxCase` 或 `macro.SyntaxRules`（允许组合），签名 MUST 为 `func(macro.Context, macro.Syntax) (macro.Syntax, error)`。

#### Scenario: try Expander 形态

- **WHEN** 查看 `try` 包 Expander 导出
- **THEN** MUST 存在 `TryExpander` 且 MUST NOT 存在 `TryExpand func(macro.CallContext, ...) (macro.CallExpandResult, error)`

#### Scenario: derive Expander 形态

- **WHEN** 查看 `derive` 包 Expander 导出
- **THEN** MUST 存在 `DeriveExpander` 且 MUST NOT 依赖 `DeclContext` 或 `DeclExpandResult`

### Requirement: Expander 可选用 macro.Quote

`try`、`with`、`derive` 与 `wirejson` 的 Expander 实现 **MAY** 使用 `macro.Quote` 进行模板化 AST 组装。该选用 **MUST NOT** 改变各 `syntax-*` spec 已定义的展开语义或 mactest 可观测结果。

使用 `macro.Quote` 产出语句的 Call Expander **MUST** 在返回 `out` 前对语句调用 `macro.StampStmtPos(site.MacroPos(), stmts)`。

#### Scenario: try 行为与迁移前一致

- **WHEN** 在本地 `replace` 到含 syntax-rules 的 `go-macro` 后，对 `try` 包执行等价于迁移前的 `mactest.ExpandSyntax` 用例
- **THEN** 展开结果的语句形状与语义 **MUST** 与迁移前 `CallExpandResult.Stmts` 一致

#### Scenario: derive 行为与迁移前一致

- **WHEN** 在本地 `replace` 到含 syntax-rules 的 `go-macro` 后，对 `derive` 包执行等价于迁移前的 `mactest.Expand` 用例（含 `Derive[fmt.Stringer]` 嵌入）
- **THEN** Apply 后 TypeSpec 字段与生成 `String()` 的语义 **MUST** 与 v0.6.0 一致
