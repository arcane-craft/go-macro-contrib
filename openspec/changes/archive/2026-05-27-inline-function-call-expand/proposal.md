## Why

当前 `inline.Inline(expr)` 仅去掉宏包装，把实参表达式原样写回，无法在宏展开阶段把「对某函数的调用」替换为该函数体的执行语义。用户期望的是接近编译器 `inline` 的能力：传入**函数调用表达式**，展开为在调用点执行被调函数实现（参数代入、无额外调用开销的 AST 形态）。被调函数返回值个数为 0～3 时，需像 `try` 的 `Try0`/`Try`/`Try2`/`Try3` 一样，通过**多个语法桩**在宏源文件中通过类型检查。

## What Changes

- 新增语法桩 **`Inline0`、`Inline`（单值，保留现名）、`Inline2`、`Inline3`**，均映射同一 `InlineExpand`；桩签名与 callee 返回值个数 `n`（0～3）对齐，展开时校验 `ctx.StubName()` 与 `n` 一致（模式同 `checkStubMatchesK`）。
- 扩展 `InlineExpand`：对可内联的函数调用做形参代入；`n=1` 且在 `SiteExpr` 时返回单个 `Expr`；`n=0` 在 `SiteStmt`、`n=2/3` 在 `SiteAssign`（及适用的 `SiteReturn`）时返回 `Stmts`。
- **`Inline0`**：桩为 `func Inline0(f func())`，实参为 `func() { ... }` 包装（因 Go 无法将无返回值调用作为表达式实参）；展开时内联被包装的无返回值函数体。
- 保留 unwrap 回退：非可内联调用、或桩与 `n` 不匹配时按规则报错或回退（见 design）。
- 增补 mactest / README；更新 `syntax-inline` spec（**BREAKING**：单桩 `Inline[T](v T) T` 扩展为多桩族，且 `Inline0`/`Inline2`/`Inline3` 允许非纯 `SiteExpr` 展开路径）。

## Capabilities

### New Capabilities

（无。）

### Modified Capabilities

- `syntax-inline`：多桩定义、`InlineExpand` 按返回值个数与调用点展开、桩与 callee 匹配校验、mactest。

## Impact

- **代码**：`inline/stubs.go`、`inline/expand.go`、`inline/expand_test.go`、`README.md`
- **依赖**：仍仅用 `macro.Context`；不依赖 `try` 包 API（可本地实现 `calleeResultCount` / `checkStubMatchesN`）。
- **兼容性**：`Inline(expr)` 单值 unwrap/内联路径保持；新增桩为 additive；用户需为 0/2/3 返回值 callee 选用对应桩名。
