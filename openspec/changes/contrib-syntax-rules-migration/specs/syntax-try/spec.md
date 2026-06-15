## MODIFIED Requirements

### Requirement: Try 语法桩族

`try` 包 MUST 提供按「error 前载荷个数」划分的多个语法桩，均 MUST panic，并共享同一 `TryExpander`（`//macro: try`）。不得仅提供单一 `func Try[T any](T, error) T` 作为唯一桩。

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
- **THEN** macro 源文件类型检查 MUST 失败；若仍进入展开，`TryExpander` MUST 报错并提示改用 `Try2`

#### Scenario: Try0 适配仅 error

- **WHEN** callee 类型为 `(error)` 且用户编写 `Try0(close())` 或 `return Try0(expr)`
- **THEN** `Try0` 桩 MUST 仅接受一个 `error` 形参，且展开器 MUST 支持该调用名

### Requirement: 可选官方宏库与引入方式

`try` 包 MUST 在 `go-macro-contrib` 仓库内作为官方宏库发布，路径为 `github.com/arcane-craft/go-macro-contrib/try`。使用方 MUST 在宏主文件中 import 该路径，且 expand 工具的 `linked` map MUST 包含该 import path 与 `TryExpander`，方可展开 `Try` 族调用。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件使用 `Try(...)` 但未 import `github.com/arcane-craft/go-macro-contrib/try`
- **THEN** 展开管线 MUST NOT 注册 `try`

#### Scenario: import 但未 link 时不展开

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/try`，但 expand 工具 `linked` 未含该 path
- **THEN** 对 `Try(...)` 的调用 MUST NOT 被展开

### Requirement: 多桩名注册到同一展开器

注册表 MUST 将 `Try0`, `Try`, `Try2`, `Try3`（及已实现的 `Try4`）的调用均映射到 `try` 的 `TryExpander`（同一 `macro.SyntaxCase` 内多 clause，按 invoked name 区分）。

#### Scenario: Try2 调用分发

- **WHEN** 展开引擎遇到 `Try2(f())` 调用
- **THEN** MUST 调用 `TryExpander` 而非要求独立的 `Try2Expander` 变量

### Requirement: Try 必须使用 Stmts（禁止 Exprs 简化 return）

在赋值、`return`、语句语境，`TryExpander` MUST 返回 `macro.Syntax` 且 `ToStmts()` 成功并产出非空 `[]ast.Stmt`。在 `return Try(...)` 语境 **MUST NOT** 仅以 `ToExpr()` 简化整条 `return`。

本要求属于 **try provider**；贴回边界与 Plan 由 pattern match 产出，引擎经 `ValidateSplice` + `Apply` 执行（见 `go-macro` `macro-splice-apply` 规范）。

#### Scenario: SiteReturn 禁止 Exprs

- **WHEN** `TryExpander` 处理 `return Try(g())`
- **THEN** 返回的 `out` MUST 经 `ToStmts()` 得到非空语句列表，且 MUST NOT 仅以单个 `Expr` 替换整条 `return`

### Requirement: Try 宏展开语义

`TryExpander` MUST 将 `Try*` 调用展开为多返回值赋值、`if err != nil { return … }`（return 实参与外层签名一致），并按语境替换原调用点（MatchedSpan 由 pattern 决定）。

#### Scenario: 展开 os.Open

- **WHEN** 外层为 `func ReadFile() ([]byte, error)` 且为 `file := Try(os.Open("hello.txt"))`
- **THEN** 展开后 MUST 在错误时 `return nil, err`

#### Scenario: return Try 完整错误处理

- **WHEN** 外层为 `func f() (T, error)` 且源码为 `return Try(g())`，`g()` 为 `(T, error)`
- **THEN** `TryExpander` MUST 产出 `Stmts` 替换整条 `return`，含 `_v, _err := g()`、`if _err != nil { return <zero T>, _err }`、`return _v, nil`，且 MUST NOT 仅用 `Exprs` 简化

#### Scenario: return Try2 完整错误处理

- **WHEN** 外层为 `func h() (A, B, error)` 且为 `return Try2(g())`
- **THEN** MUST 用 `Stmts` 生成完整多值赋值、错误分支与成功 `return a, b, nil`

#### Scenario: return 块可扩展 error 包装

- **WHEN** 展开 `return Try(g())` 生成 `if _err != nil { … }`
- **THEN** 该块 MUST 为完整语句块，以便后续插入 `fmt.Errorf` 等 error 包装

### Requirement: error 必须在返回列表最后

`TryExpander` MUST 要求：**内层** `binds.Get("inner")` 所指 callee 与 **外层**（调用所在函数）的返回参数列表中，`error` 均为**最后一个**返回参数。外层签名 MUST 通过 `macro.EnclosingResults(ctx, site)` 取得。否则 MUST 拒绝展开并报错。

#### Scenario: 内层 error 非最后

- **WHEN** `Try` 的实参调用类型为 `(error, int)` 或其它将 `error` 置于非末尾的形式
- **THEN** `TryExpander` MUST 返回错误，说明 callee 返回值排列不合法

#### Scenario: 外层 error 非最后

- **WHEN** 调用出现在 `func f() (err error, n int)` 中
- **THEN** `TryExpander` MUST 返回错误，说明当前函数不能使用 Try 族宏

### Requirement: 外层函数必须含 error 返回

包含调用的函数 MUST 在返回签名中以 `error` 结尾。否则 `TryExpander` MUST 拒绝展开。

#### Scenario: 无 error 返回的函数

- **WHEN** 调用出现在 `func f() int` 内
- **THEN** `TryExpander` MUST 返回错误

#### Scenario: 仅返回 error 的外层与 Try0

- **WHEN** 外层为 `func f() error` 且为 `return Try0(expr)`，`expr` 为 `(error)`
- **THEN** 展开后错误路径 MUST `return err`，成功 MUST `return nil`

### Requirement: 内层 callee 与桩名一致性

`TryExpander` MUST 用 `go/types` 校验 callee 载荷数 `k` 与 **pattern 匹配的 invoked name**（`Try`/`Try0`/`Try2`/`Try3` 等）一致：`Try` 仅允许 k=1；`Try0` 仅 k=0；`Try2` 仅 k=2；以此类推。

#### Scenario: 展开期校验载荷数

- **WHEN** 调用为 `Try2(f())` 但 `f` 实际返回 `(T, error)`
- **THEN** `TryExpander` MUST 报错并建议改用 `Try`

### Requirement: 具名返回支持

错误路径上的 `return` MUST 优先使用外层具名标识符（若存在）；无具名时 MUST 使用与外层签名一致的零值与 `err` 变量。

#### Scenario: 具名 error

- **WHEN** 外层为 `func f() (data []byte, err error)` 且为 `Try` 赋值语境
- **THEN** 错误路径 MUST `return nil, err`（使用具名 `err`）

### Requirement: 非法调用错误信息

非法用法 MUST 在展开期失败，错误 MUST 含文件、行号、原因及修复提示（含应使用的桩名如 `Try2`）。

#### Scenario: 错桩名提示

- **WHEN** callee 为 `(A, B, error)` 但用户调用 `Try(f())`
- **THEN** `TryExpander` MUST 返回错误，且错误信息 MUST 建议使用 `Try2`

## ADDED Requirements

### Requirement: TryExpander 实现形态

`try` 包 MUST 提供 `var TryExpander = macro.SyntaxCase(...)`。Clause MUST 覆盖赋值（`:=` / `=` / `var ... =`）、`return $vals ... , Try($inner)`、语句 `Try0($inner);` 等合法形态；宽 Stmt pattern MUST 排在窄 Call pattern 之前。

#### Scenario: 赋值 pattern 绑定 lhs

- **WHEN** 展开 `x, err := Try(helper())`
- **THEN** `TryExpander` MUST 通过 pattern 捕获 `$lhs` 与 `$inner`，且 MUST NOT 依赖 `findAssignStmt` 遍历 AST

### Requirement: Try 拒绝表达式位置

`TryExpander` MUST 仅通过 `SyntaxCase` **StmtPattern** clause 匹配赋值、`return` 与 `Try0` 语句语境。表达式位置（如 `_ = 1 + Try(g())`）MUST 因无匹配 StmtPattern 而展开失败。`TryExpander` MUST NOT 为拒绝非法位置添加 CallPattern catch-all。

非法语境下错误 MUST 为框架 `macro: no matching syntax rule`（MAY 附带末条 match 原因，如 `assign stmt not found`、`unsupported call parent context`）。MUST NOT 要求含 `expression position`（见 change design D11）。

#### Scenario: 表达式位置拒绝

- **WHEN** 展开 `_ = 1 + Try(g())` 于表达式位置
- **THEN** `TryExpander` MUST 返回错误，且错误信息 MUST 含 `no matching syntax rule`
