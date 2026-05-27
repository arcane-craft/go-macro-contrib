## MODIFIED Requirements

### Requirement: Inline 语法桩

`inline` 包 MUST 提供下列语法桩，函数体 MUST panic，并标注为宏桩不可直接调用；均 MUST 带 `//macro: syntax-inline` 并映射到同一 `InlineExpand`：

| 桩名 | 返回值个数 n | 用途 |
|------|-------------|------|
| `Inline0` | 0 | 无返回值被调函数（经 `func()` 包装传入） |
| `Inline` | 1 | 单返回值表达式或调用 |
| `Inline2` | 2 | 双返回值，用于多值赋值/return |
| `Inline3` | 3 | 三返回值，用于多值赋值/return |

桩签名 MUST 满足：

- `Inline0(f func())`
- `Inline[T any](v T) T`
- `Inline2[A, B any](a A, b B) (A, B)`
- `Inline3[A, B, C any](a A, b B, c C) (A, B, C)`

#### Scenario: 单值宏源类型检查

- **WHEN** 用户编写 `x := Inline(42)`
- **THEN** `Inline` 桩 MUST 使该表达式通过类型检查（与展开后 `x := 42` 一致）

#### Scenario: 双值赋值类型检查

- **WHEN** 用户编写 `a, b := Inline2(pair())` 且 `pair` 返回 `(string, string)`
- **THEN** `Inline2` 桩 MUST 使该赋值通过类型检查

#### Scenario: 无返回值语句类型检查

- **WHEN** 用户编写 `inline.Inline0(func() { cleanup() })` 且 `cleanup` 无返回值
- **THEN** `Inline0` 桩 MUST 使该语句通过类型检查

### Requirement: InlineExpand 表达式展开

`inline` 包 MUST 提供 `//macro: syntax-inline` 标注的 `InlineExpand`。展开器 MUST 根据 `ctx.StubName()` 确定期望返回值个数 `n`（`Inline0`→0，`Inline`→1，`Inline2`→2，`Inline3`→3），并 MUST 用 `types.Info` 计算内层被调函数返回值个数；二者不一致时 MUST 返回错误并提示应使用的桩名。

**内联路径**（实参可归一化为对内层 `*ast.CallExpr` 的引用，且 callee 为同文件可内联 `*ast.FuncDecl`）：

- `n=1` 且 `SiteExpr`：MUST 返回 `ExpandResult{Expr: <代入后的单表达式>}`，MUST NOT 保留外层 callee 调用。
- `n=0` 且 `SiteStmt`：MUST 返回 `ExpandResult{Stmts: ...}`，为代入后的函数体语句。
- `n=2` 或 `n=3` 且 `SiteAssign`（及适用的 `SiteReturn`）：MUST 返回 `ExpandResult{Stmts: ...}`，完成对外层左值或 `return` 的赋值语义。
- 可内联函数：同文件、直接标识符调用、形参与内层实参个数一致、函数体为可内联形状（`return` 结果个数与 `n` 一致，每个结果为单表达式；`n=0` 为无结果返回的语句体）。

**回退与拒绝**：

- `Inline` + `SiteExpr` + 实参非可内联 `CallExpr`：MUST 返回 `ExpandResult{Expr: <实参表达式>}`（unwrap）。
- 已解析为同文件 `*ast.FuncDecl` 但不可内联：MUST 返回错误。
- `Inline2`/`Inline3` 出现在 `SiteExpr`，或 `Inline0` 出现在 `SiteExpr`：MUST 返回错误，说明允许的调用点。

#### Scenario: SiteExpr 内联单返回值调用

- **WHEN** 同文件存在 `func add(a, b int) int { return a + b }` 且展开 `Inline(add(1, 2))`，`ctx.Site()` 为 `SiteExpr`
- **THEN** `InlineExpand` MUST 返回非空 `Expr`，语义等价于代入后的 `a + b`，且 MUST NOT 包含对 `add` 的 `CallExpr`

#### Scenario: SiteAssign 内联双返回值调用

- **WHEN** 同文件存在 `func split() (string, string) { return "a", "b" }` 且展开 `a, b := Inline2(split())`
- **THEN** `InlineExpand` MUST 返回非空 `Stmts`，且 MUST NOT 保留对 `split` 的调用

#### Scenario: 桩与 callee 返回值个数不匹配

- **WHEN** callee 返回 2 个值但用户使用 `Inline(split())`
- **THEN** `InlineExpand` MUST 返回错误，提示使用 `Inline2`

#### Scenario: SiteExpr 非调用实参保持 unwrap

- **WHEN** 展开 `Inline(42)` 且 `ctx.Site()` 为 `SiteExpr`
- **THEN** `InlineExpand` MUST 返回 `ExpandResult{Expr: <实参表达式>}`

#### Scenario: SiteExpr 不可内联的同文件函数报错

- **WHEN** callee 已解析为同文件 `*ast.FuncDecl` 但不满足可内联形状
- **THEN** `InlineExpand` MUST 返回错误

#### Scenario: 非表达式语境对 Inline 的限制

- **WHEN** `Inline` 出现在 `SiteAssign`、`SiteReturn` 或 `SiteStmt` 且未走内联 `Stmts` 路径
- **THEN** 行为由实现与 mactest 定义；首版 MAY 在单值 `SiteAssign` 支持内联为 `Stmts`，否则返回错误说明仅 `SiteExpr` 支持纯 unwrap

## ADDED Requirements

### Requirement: 多桩与多返回值 mactest

`InlineExpand` 的 `mactest` MUST 覆盖：`Inline` 单值内联与 unwrap、`Inline2` 在 `SiteAssign` 的内联、`Inline0` 在 `SiteStmt` 的内联（`func()` 包装）、桩与 `n` 不匹配错误、不可内联函数体错误。

#### Scenario: 纯 Expand 测试

- **WHEN** 在 `go-macro-contrib` 仓库内执行 `go test ./inline/...`
- **THEN** 测试 MUST 无需全链路 expand 即可通过

#### Scenario: Inline2 与 Inline 区分

- **WHEN** mactest 对双返回值 callee 使用 `Inline`
- **THEN** 测试 MUST 断言展开失败并提示 `Inline2`
