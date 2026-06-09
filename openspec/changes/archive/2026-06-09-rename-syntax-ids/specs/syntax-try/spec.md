## MODIFIED Requirements

### Requirement: Try 语法桩族

`try` 包 MUST 提供按「error 前载荷个数」划分的多个语法桩，均 MUST panic，并共享同一 `TryExpand`（`//macro: try`）。不得仅提供单一 `func Try[T any](T, error) T` 作为唯一桩。

| 桩名 | 签名（概念） | callee 载荷数 k |
|------|--------------|-----------------|
| `Try0` | `(error)` → 无值 | k=0 |
| `Try` | `(T, error) → T` | k=1 |
| `Try2` | `(A, B, error) → (A, B)` | k=2 |
| `Try3` | `(A, B, C, error) → (A, B, C)` | k=3 |
| `Try4` | 四载荷 + error → 四元组 | k=4（可选） |

#### Scenario: Try 适配一元载荷

- **WHEN** 用户编写 `Try(os.Open(...))` 且 `os.Open` 类型为 `( *os.File, error)`
- **THEN** `Try` 桩 MUST 使 macro 源文件通过类型检查，且展开器 MUST 接受该调用

#### Scenario: 多载荷须用 Try2

- **WHEN** callee 类型为 `(A, B, error)` 且用户编写 `Try2(f())`
- **THEN** 桩 MUST 为 `(A, B, error) → (A, B)` 形式，且 macro 源文件 MUST 通过类型检查

#### Scenario: 错用 Try 处理多载荷

- **WHEN** callee 类型为 `(A, B, error)` 但用户编写 `Try(f())`（两载荷却用 `Try` 桩）
- **THEN** macro 源文件类型检查 MUST 失败；若仍进入展开，`TryExpand` MUST 报错并提示改用 `Try2`

#### Scenario: Try0 适配仅 error

- **WHEN** callee 类型为 `(error)` 且用户编写 `Try0(close())` 或 `return Try0(expr)`
- **THEN** `Try0` 桩 MUST 仅接受一个 `error` 形参，且展开器 MUST 支持该调用名

### Requirement: 可选官方宏库与引入方式

`try` 包 MUST 在 `go-macro-contrib` 仓库内作为官方宏库发布，路径为 `github.com/arcane-craft/go-macro-contrib/try`。使用方 MUST 在宏主文件中 import 该路径，且 expand 工具的 `linked` map MUST 包含该 import path 与 `TryExpand`，方可展开 `Try` 族调用。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件使用 `Try(...)` 但未 import `github.com/arcane-craft/go-macro-contrib/try`
- **THEN** 展开管线 MUST NOT 注册 `try`

#### Scenario: import 但未 link 时不展开

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/try`，但 expand 工具 `linked` 未含该 path
- **THEN** 对 `Try(...)` 的调用 MUST NOT 被展开

### Requirement: 多桩名注册到同一展开器

注册表 MUST 将 `Try0`, `Try`, `Try2`, `Try3`（及已实现的 `Try4`）的调用均映射到 `try` 的 `TryExpand`。

#### Scenario: Try2 调用分发

- **WHEN** 展开引擎遇到 `Try2(f())` 调用
- **THEN** MUST 调用 `TryExpand` 而非要求独立的 `Try2Expand` 函数

### Requirement: Try 必须使用 Stmts（禁止 Exprs 简化 return）

在 `SiteAssign`、`SiteReturn`、`SiteStmt` 语境，`TryExpand` MUST 设置显式 `CallExpandResult.Target`（分别为 `SpliceReplaceAssignStmt`、`SpliceReplaceReturnStmt`、`SpliceReplaceExprStmt`）并返回非空 `Stmts`。在 `SiteReturn` **MUST NOT** 使用 `Target: SpliceReplaceReturnResults` 或仅设置 `Exprs`。

本要求属于 **try provider**；框架按 `CallExpandResult.Target` 贴回 AST（见 `go-macro` `macro-expander` 规范）。

#### Scenario: SiteReturn 禁止 Exprs

- **WHEN** `TryExpand` 处理 `return Try(g())` 且 `ctx.Site()` 为 `SiteReturn`
- **THEN** 返回的 `CallExpandResult` MUST 含 `Stmts` 且 MUST NOT 仅设置 `Exprs`
