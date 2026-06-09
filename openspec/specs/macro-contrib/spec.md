# macro-contrib Specification

## Purpose

定义 `go-macro-contrib` 官方宏库仓库的 module 边界、发布路径、与 `cmd/macro expand` 的集成方式、独立测试及对所依赖 `go-macro` 核心版本的约束。
## Requirements
### Requirement: contrib 独立子 module

官方宏库 MUST 在**独立 Git 仓库**中作为根 Go module 发布，module 路径为 `github.com/arcane-craft/go-macro-contrib`。`go-macro` 仓库 MUST NOT 再包含 `contrib/` 目录或顶层官方语法实现目录（如 `syntax/`、`derive/`、`wirejson/`）。

`go-macro` 根 module 的 `internal/expander`、`macro/expandtool` 及根 module 内所有测试 MUST NOT import 官方宏实现包（`inline`、`try`、`with`、`derive`、`wirejson`、`register`）。

#### Scenario: expander 不依赖 contrib 实现

- **WHEN** 编译 `go-macro` 根 module 的 `internal/expander` 包
- **THEN** MUST NOT import `github.com/arcane-craft/go-macro-contrib/inline`、`.../try`、`.../with`、`.../derive` 或 `.../wirejson`

#### Scenario: 根测试不 import contrib

- **WHEN** 编译 `go-macro` 根 module 任意 `*_test.go`
- **THEN** MUST NOT import `github.com/arcane-craft/go-macro-contrib/...`

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

### Requirement: 官方宏库与 cmd/macro expand 集成

`go-macro-contrib` MUST NOT 提供 `register` 包。官方宏库应由 `cmd/macro expand` 基于 provider 上的 `//macro:` 指令自动发现并链接：Call 宏 link `*Expand`（`CallExpander`），Decl 宏 link `*Expand`（`DeclExpander`），按 **syntax-id** 分别 `RegisterCall` / `RegisterDecl`。

`go-macro-contrib` MUST NOT 提供 `Main`/`Run` 作为用户 expand 入口（该职责在 `cmd/macro` / `macro/expandtool`）。

#### Scenario: 通过 expand 自动 link Call 宏

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/try` 并执行 `cmd/macro expand`
- **THEN** 展开 MUST 成功链接 `TryExpand`，且无需 blank import `register`

#### Scenario: 通过 expand 自动 link Decl 宏

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/wirejson` 且 struct 匿名嵌入 `wirejson.WireJSON`，并执行 `cmd/macro expand`
- **THEN** 展开 MUST 成功链接 `WireJSONExpand`（syntax-id `wire-json`）

### Requirement: contrib 独立测试

`go-macro-contrib` 仓库 MUST 具备独立 `go test ./...`，含 `inline`、`try`、`with` 的 `mactest.ExpandCall` 单测，以及 `derive`、`wirejson` 的 `mactest.ExpandDecl` 单测。

#### Scenario: contrib 测试不依赖 go tool macro expand

- **WHEN** 在 `go-macro-contrib` 仓库根执行 `go test ./...`
- **THEN** MUST 通过

### Requirement: contrib 依赖 go-macro 核心版本

`go-macro-contrib` 的**已提交** `go.mod` **MUST** `require` 已发布的 `github.com/arcane-craft/go-macro` 版本（semver tag，非 `v0.0.0` 占位）；**MUST NOT** 在已提交 `go.mod` 中包含指向 sibling 目录的 `replace` 指令。

所 pin 的 `go-macro` 版本 MUST 同时提供：**Call API**（`CallContext`、`CallExpandResult`、`CallExpander`）与 **Decl API**（`DeclContext`、`DeclExpandResult`、`DeclExpander`）及按 syntax-id 的 `RegisterCall` / `RegisterDecl`。

README **MUST** 注明最低兼容核心版本，且所述版本 **MUST** 与 `go.mod` 中 pin 的 `require` 一致。

本地联调时，contrib 仓库 **SHOULD** 与 `go-macro` 位于同级目录（`go-macro-contrib` 与 `go-macro` 并列）。开发者 **MAY** 在本地向 `go-macro-contrib/go.mod` 添加 `replace github.com/arcane-craft/go-macro => ../go-macro` 以联调未发布的核心变更；该 `replace` **MUST NOT** 作为发布/tag 前提交态的硬性要求。

#### Scenario: 独立仓可解析核心依赖

- **WHEN** 在仅 clone `go-macro-contrib`、已提交 `go.mod` 无 `replace`，且模块代理可解析所 pin 的 `go-macro` tag 时执行 `go test ./...`
- **THEN** MUST 成功解析 `github.com/arcane-craft/go-macro` 模块依赖并通过测试

#### Scenario: 本地 replace 联调核心

- **WHEN** 开发者在 `go-macro-contrib` 本地添加 `replace github.com/arcane-craft/go-macro => ../go-macro` 后执行 `go test ./...`
- **THEN** MUST 使用 sibling 核心源码解析依赖（联调行为；不要求写入已提交 `go.mod`）

### Requirement: Call 与 Decl Expander 可选用 macro/quote

`try`、`with` 与 `derive` 包的 Expander 实现 **MAY** 使用 `github.com/arcane-craft/go-macro/macro/quote` 进行模板化 AST 组装。该选用 **MUST NOT** 改变 `try`、`with` 或 `syntax-derive` 已定义的展开语义、`mactest` 可观测结果或公开桩 API。

使用 `macro/quote` 的 Call Expander **MUST** 在返回 `CallExpandResult` 前对语句结果调用 `macro.StampStmtPos(ctx.MacroPos(), stmts)`（与手写 AST 相同）。Quote **MUST NOT** 代替 Expander 设置 `SpliceTarget`。

#### Scenario: try 行为与迁移前一致

- **WHEN** 在本地 `replace` 到含 `macro/quote` 的 `go-macro` 后，对 `try` 包执行 `mactest.ExpandCall` 既有用例
- **THEN** 展开结果的 `Target`、`Stmts` 形状与语义 **MUST** 与迁移前一致

#### Scenario: derive 行为与迁移前一致

- **WHEN** 在本地 `replace` 到含 `macro/quote` 的 `go-macro` 后，对 `derive` 包执行 `mactest.ExpandDecl` 既有用例（含 `Derive[fmt.Stringer]` 嵌入）
- **THEN** 展开结果的 `Fields`、`Methods`（含生成的 `String()`）**MUST** 与 `derivestringer` 时代语义一致

### Requirement: quote 联调与已提交 go.mod 约束

开发者在本地联调 `macro/quote` 时 **MAY** 向 `go-macro-contrib/go.mod` 添加 `replace github.com/arcane-craft/go-macro => ../go-macro`。已提交（发布/tag 前）的 `go.mod` **MUST** 继续 `require` 已发布的 `go-macro` semver tag，且 **MUST NOT** 包含 `replace` 指令，直至核心发布含 `macro/quote` 的版本并 bump `require`。

#### Scenario: 本地 replace 解析 quote

- **WHEN** 开发者添加 `replace` 指向 sibling `go-macro` 且该核心含 `macro/quote`
- **THEN** `go test ./...` **MUST** 能解析 `github.com/arcane-craft/go-macro/macro/quote`

#### Scenario: 已提交 go.mod 无 replace

- **WHEN** 查看发布前已提交的 `go-macro-contrib/go.mod`
- **THEN** **MUST NOT** 含 `replace github.com/arcane-craft/go-macro`

