## MODIFIED Requirements

### Requirement: Inline 语法桩

`inline` 包 MUST 提供下列语法桩，函数体 MUST panic，并标注为宏桩不可直接调用；均 MUST 带 `//macro: inline` 并映射到同一 `InlineExpander`：

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

- **WHEN** 用户在 macro 主文件中编写 `x := Inline(42)`
- **THEN** `Inline` 桩 MUST 使该表达式通过类型检查（与展开后 `x := 42` 一致）

#### Scenario: 双值赋值类型检查

- **WHEN** 用户编写 `a, b := Inline2(pair())` 且 `pair` 返回 `(string, string)`
- **THEN** `Inline2` 桩 MUST 使该赋值通过类型检查

#### Scenario: 无返回值语句类型检查

- **WHEN** 用户编写 `inline.Inline0(func() { cleanup() })` 且 `cleanup` 无返回值
- **THEN** `Inline0` 桩 MUST 使该语句通过类型检查

### Requirement: InlineExpand 表达式展开

`inline` 包 MUST 提供 `//macro: inline` 标注的 `InlineExpander`（`macro.SyntaxCase` 或 `SyntaxRules` 组合）。展开器 MUST 根据 **pattern 匹配的 invoked name**（`Inline0`/`Inline`/`Inline2`/`Inline3`）确定期望返回值个数 `n`，并 MUST 用 `types.Info` 计算内层被调函数返回值个数；二者不一致时 MUST 返回错误并提示应使用的桩名。

**内联路径**（`binds.Get("inner")` 可归一化为对内层 `*ast.CallExpr` 的引用，且 callee 为同文件可内联 `*ast.FuncDecl`）：

- `n=1` 且表达式语境：MUST 返回 `out` 使 `ToExpr()` 为代入后的单表达式，MUST NOT 保留外层 callee 调用。
- `n=1` 且 `return` 语境：MUST 返回 `out` 使 `ToStmts()` 含代入后的单表达式 `return`。
- `n=0` 且语句语境：MUST 返回 `out` 使 `ToStmts()` 为代入后的函数体语句。
- `n=2` 或 `n=3` 且赋值（及适用的 `return`）语境：MUST 返回 `out` 使 `ToStmts()` 完成对外层左值或 `return` 的赋值语义。
- 可内联函数：同文件、直接标识符调用、形参与内层实参个数一致、函数体为可内联形状（`return` 结果个数与 `n` 一致，每个结果为单表达式；`n=0` 为无结果返回的语句体）。

**回退与拒绝**：

- `Inline` + 表达式语境 + 实参非可内联 `CallExpr`：MUST unwrap 为实参表达式（`ToExpr()`）。
- 已解析为同文件 `*ast.FuncDecl` 但不可内联：MUST 返回错误。
- `Inline2`/`Inline3` 出现在表达式语境，或 `Inline0` 出现在表达式语境：MUST 返回错误，说明允许的调用点。
- `Inline` 出现在赋值或语句语境（且未走内联 `Stmts` 路径）：MUST 返回错误，说明 `Inline` 仅用于表达式或 `return` 位置。

#### Scenario: SiteExpr 内联单返回值调用

- **WHEN** 同文件存在 `func add(a, b int) int { return a + b }` 且展开 `Inline(add(1, 2))` 于表达式位置
- **THEN** `InlineExpander` MUST 返回 `out` 使 `ToExpr()` 语义等价于代入后的 `a + b`，且 MUST NOT 包含对 `add` 的 `CallExpr`

#### Scenario: SiteAssign 内联双返回值调用

- **WHEN** 同文件存在 `func split() (string, string) { return "a", "b" }` 且展开 `a, b := Inline2(split())`
- **THEN** `InlineExpander` MUST 返回 `out` 使 `ToStmts()` 非空，且 MUST NOT 保留对 `split` 的调用

#### Scenario: 桩与 callee 返回值个数不匹配

- **WHEN** callee 返回 2 个值但用户使用 `Inline(split())`（且通过类型检查的其他路径不可用）
- **THEN** 展开器 MUST 在可检测时返回错误，提示使用 `Inline2`

#### Scenario: SiteExpr 非调用实参保持 unwrap

- **WHEN** 展开 `Inline(42)` 于表达式位置
- **THEN** `InlineExpander` MUST 返回 `out` 使 `ToExpr()` 为实参表达式

#### Scenario: SiteExpr 不可内联的同文件函数报错

- **WHEN** callee 已解析为同文件 `*ast.FuncDecl` 但不满足可内联形状
- **THEN** `InlineExpander` MUST 返回错误

### Requirement: 可选官方宏库与引入方式

`inline` 包 MUST 在 `go-macro-contrib` 仓库内作为官方宏库发布，路径为 `github.com/arcane-craft/go-macro-contrib/inline`。使用方 MUST 在宏主文件中 import 该路径，且 expand 工具的 `linked` map MUST 包含该 import path 与 `InlineExpander`，方可展开 `Inline(...)` 及同语法族其它桩。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件调用 `Inline(...)` 但未 import `github.com/arcane-craft/go-macro-contrib/inline`
- **THEN** 展开管线 MUST NOT 注册 `inline`

#### Scenario: import 但未 link 时不展开

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/inline`，但 expand 工具 `linked` 未含该 path
- **THEN** 对 `Inline(...)` 的调用 MUST NOT 被展开

### Requirement: 与框架边界

`inline` 包 MUST NOT 依赖 `macro` 包内的 error 载荷、k 校验或 Try 专用 API；MUST 仅使用统一 `macro.Context`、`macro.Syntax` 与 `macro.SyntaxCase` / `SyntaxRules`。

#### Scenario: 独立 syntax-id

- **WHEN** 注册表构建完成
- **THEN** 各 `Inline*` 桩 MUST 映射到 `inline` 的 `InlineExpander`，且 MUST NOT 与 `try` 共用展开器

### Requirement: mactest 单测

`InlineExpander` MUST 具备不依赖 `//go:build macro` 的 `mactest.ExpandSyntax` 单测，测试包路径 MUST 为 `go-macro-contrib` 仓库内的 `inline`（或 `inline_test`）。测试 MUST 覆盖：`Inline` 单值内联与 unwrap、`Inline2` 在赋值语境的内联、`Inline0` 在语句语境的内联（`func()` 包装）、桩与 `n` 不匹配错误、不可内联函数体错误。

#### Scenario: 纯 Expand 测试

- **WHEN** 在 `go-macro-contrib` 仓库内执行 `go test ./inline/...`
- **THEN** 测试 MUST 无需全链路 expand 即可通过

#### Scenario: Inline2 与 Inline 区分

- **WHEN** 测试对双返回值 callee 使用 `Inline` 桩展开
- **THEN** MUST 断言失败并提示 `Inline2`
