## Context

`inline` 包当前仅提供 `Inline[T any](v T) T` 单桩，`InlineExpand` 在 `SiteExpr` 做 unwrap。`try` 包已用 `Try0`/`Try`/`Try2`/`Try3` + 单一 `TryExpand` + `checkStubMatchesK` 解决「被调表达式返回值个数」与桩签名不一致时的类型检查问题。本变更将同一模式用于 `inline` 与函数体内联。

## Goals / Non-Goals

**Goals:**

- 提供 `Inline0`、`Inline`、`Inline2`、`Inline3` 四个桩（`Inline` 表示 `n=1`，不引入 `Inline1` 以保持兼容）。
- 从宏调用的实参中取出**内层被内联调用**（与 `TryExpand` 相同：优先 `call.Args[0]` 为 `*ast.CallExpr`；`Inline2`/`Inline3` 亦可能呈现为多实参形式，归一化为对内层 `CallExpr` 的解析）。
- 用 `types.Info` 得到 callee 的 `*ast.FuncDecl` 与返回值个数 `n`，`checkStubMatchesN(stub, n)`。
- 可内联形状：同文件、`return` 列表长度与 `n` 一致且每项为单表达式（`n=0` 时为无结果或仅 `return`）、形参与内层调用实参一致。
- 按 `n` 与 `Site` 输出 `Expr` 或 `Stmts`（见下表）。

**Non-Goals:**

- `n > 3`、variadic、方法、跨包、复杂控制流体。
- 与 `try` 共用实现包（仅复制模式）。

## Decisions

### 1. 桩函数签名（对齐 try 风格）

| 桩 | n | 桩签名（示意） | 典型写法 |
|----|---|----------------|----------|
| `Inline0` | 0 | `func Inline0(f func())` | `inline.Inline0(func() { cleanup() })` |
| `Inline` | 1 | `func Inline[T any](v T) T` | `inline.Inline(add(1, 2))` |
| `Inline2` | 2 | `func Inline2[A,B any](a A, b B) (A, B)` | `a, b := inline.Inline2(split(s))` |
| `Inline3` | 3 | `func Inline3[A,B,C any](a A, b B, c C) (A, B, C)` | 三值赋值语境 |

`Inline2`/`Inline3` 的宏调用在 AST 上通常表现为**一个**内层 `CallExpr` 实参（多值传给多形参），与 `Try2(foo())` 一致；`InlineExpand` 从 `call.Args` 归一化出该内层调用。

**Inline0 与无返回值调用**：Go 类型系统不允许 `Inline0(f())`（`f()` 非表达式）。采用 **`func()` 包装**；展开器识别实参为 `func() { ... }` 时，从中提取对无返回值函数的直接调用并内联其 `FuncDecl`，或内联包装体内的单条调用。文档明确该写法。

### 2. 单一 `InlineExpand` + `checkStubMatchesN`

所有桩共享 `//macro: syntax-inline` 与 `InlineExpand`（与 `TryExpand` 相同）。根据 `ctx.StubName()` 期望的 `n` 与 `calleeResultCount(inner)` 比较，不匹配则 `macro.ErrorAt`（如 `use Inline2 for 2-result callee`）。

`calleeResultCount`：解析内层 `CallExpr` 的 callee 签名 `Results().Len()`，或 `types.Tuple` 长度；不依赖 error 尾项。

### 3. 调用点与 `ExpandResult` 形态

| n | 允许的 `CallSiteKind` | 展开结果 |
|---|------------------------|----------|
| 0 | `SiteStmt` | `Stmts`：代入后的函数体语句列表（无 `return` 值） |
| 1 | `SiteExpr`；可选 `SiteAssign`/`SiteReturn`（单值） | `SiteExpr` → `Expr`；赋值/return 语境可用 `Stmts` 或引擎支持的 RHS `Expr` |
| 2, 3 | `SiteAssign`；`SiteReturn` 若外层签名匹配 | `Stmts`：内联体代入后，对外层 LHS / `return` 的赋值或 `return` 语句序列 |

**理由**：Go 无多值表达式；`n≥2` 无法仅在 `SiteExpr` 用单个 `Expr` 表示，必须生成语句（与 `try` 在 `SiteAssign` 返回 `Stmts` 一致）。

**`Inline` 非调用实参**：在 `SiteExpr` 仍 **unwrap** 返回 `call.Args[0]`（兼容 `Inline(42)`）。

### 4. 内联体与参数替换

- 解析同文件 `*ast.FuncDecl`（首版，见原 design）。
- `return` 右侧：`n` 个表达式，与 `Results` 一一对应；`n=0` 时体为语句列表且无结果 `return` 或空 `return`。
- `substituteParams` 支持在多个 return 表达式上替换形参；`n≥2` 时生成 `Stmts` 完成对外层赋值的拼接（可复用 `macro.StampStmtPos`）。

### 5. 错误 vs 回退

| 情况 | 行为 |
|------|------|
| 桩 `Inline` + `SiteExpr` + 非 `CallExpr` 实参 | unwrap |
| 内层为 `CallExpr` 但不可解析 / 非本文件 | unwrap（`Inline`）或报错（已解析但不可内联，同前） |
| 桩 `n` 与 callee 结果数不一致 | **报错** |
| `Inline2` 出现在 `SiteExpr` | **报错**（须用于赋值/return） |
| `Inline0` 出现在 `SiteExpr` | **报错** |

## Risks / Trade-offs

- **[Risk] Inline0 需 `func()` 包装** → README 示例；长期可考虑语句级特殊形式（非首版）。
- **[Risk] 多值路径生成 `Stmts` 与旧 spec「仅 Expr」冲突** → 本变更 MODIFIED 明确按 `n` 区分。
- **[Risk] 实参多次求值** → 文档说明。

## Migration Plan

1. 实现多桩 + `expand.go` 分支。
2. 更新测试与 README。
3. 仅使用 `Inline` 且仅 unwrap 的用户无感；多返回值 callee 需改用 `Inline2`/`Inline3` 并在赋值语境使用。

## Open Questions

- `Inline` 在 `SiteAssign` 单值赋值是否统一走 `Stmts` 还是仅 `Expr` 替换 RHS（取决于 expander 能力，实现时以 mactest + 全链路 expand 为准）。
- 同包跨文件 `FuncDecl` 查找仍为首版限制。
