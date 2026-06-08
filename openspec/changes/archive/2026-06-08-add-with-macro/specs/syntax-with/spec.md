## ADDED Requirements

### Requirement: With 语法桩

`with` 包 MUST 提供单一语法桩 `With[T io.Closer](v T, err error) T`。函数体 MUST panic，并注明勿在未展开时调用。桩 MUST 带 `//macro: syntax-with` 并映射到 `WithExpand`。

v1 MUST NOT 提供 `With0`、`With2`、`With3` 等多桩；callee 载荷数 k MUST 为 1（即 `(T, error)`）。

#### Scenario: With 适配 io.Closer 一元载荷

- **WHEN** 用户编写 `With(os.Open(...))` 且 `os.Open` 类型为 `(*os.File, error)`，且 `*os.File` 实现 `io.Closer`
- **THEN** `With` 桩 MUST 使 macro 源文件通过类型检查，且展开器 MUST 接受该调用

#### Scenario: 非 io.Closer 类型检查失败

- **WHEN** callee 返回 `(int, error)` 且用户编写 `With(f())`
- **THEN** macro 源文件类型检查 MUST 失败（泛型约束 `io.Closer` 不满足）

### Requirement: 官方宏库与引入方式

`with` 包 MUST 在 `go-macro-contrib` 仓库内作为官方宏库发布，路径为 `github.com/arcane-craft/go-macro-contrib/with`。使用方 MUST 在宏主文件中 import 该路径，且 expand 工具的 `linked` map MUST 包含该 import path 与 `WithExpand`，方可展开 `With` 调用。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件使用 `With(...)` 但未 import `github.com/arcane-craft/go-macro-contrib/with`
- **THEN** 展开管线 MUST NOT 注册 `syntax-with`

#### Scenario: import 但未 link 时不展开

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/with`，但 expand 工具 `linked` 未含该 path
- **THEN** 对 `With(...)` 的调用 MUST NOT 被展开

### Requirement: error 必须在返回列表最后

`WithExpand` MUST 要求：**内层** `expr` 与 **外层**（调用所在函数）的返回参数列表中，`error` 均为**最后一个**返回参数。否则 MUST 拒绝展开并报错。

#### Scenario: 内层 error 非最后

- **WHEN** `With` 的实参调用类型为 `(error, int)` 或其它将 `error` 置于非末尾的形式
- **THEN** `WithExpand` MUST 返回错误，说明 callee 返回值排列不合法

#### Scenario: 外层 error 非最后

- **WHEN** 调用出现在 `func f() (err error, n int)` 中
- **THEN** `WithExpand` MUST 返回错误，说明当前函数不能使用 `With` 宏

### Requirement: 外层函数必须含 error 返回

包含调用的函数 MUST 在返回签名中以 `error` 结尾。否则 MUST 拒绝展开。

#### Scenario: 无 error 返回的函数

- **WHEN** 调用出现在 `func f() int` 内
- **THEN** `WithExpand` MUST 返回错误

### Requirement: 内层 callee 载荷数校验

展开器 MUST 用 `go/types` 校验 callee 在 `error` 前的载荷数 k：v1 仅允许 k=1。k≠1 时 MUST 返回 error，并说明 v1 仅支持 `(T, error)`。

#### Scenario: 多载荷 callee 拒绝

- **WHEN** 调用为 `With(f())` 但 `f` 实际返回 `(A, B, error)`
- **THEN** `WithExpand` MUST 报错并说明仅支持单载荷 + error

### Requirement: io.Closer 展开期校验

除桩泛型约束外，`WithExpand` MUST 在展开前用 `types.Implements` 校验 k=1 时载荷类型实现 `io.Closer`。若不满足，MUST 返回 error，且错误信息 MUST 指明类型须实现 `io.Closer`。

#### Scenario: 动态类型非 Closer 拒绝展开

- **WHEN** callee 返回 `(T, error)` 且 `T` 在类型检查时可赋值给 `io.Closer` 约束但展开期判定未实现 `io.Closer`
- **THEN** `WithExpand` MUST 返回错误

### Requirement: Site 约束

`WithExpand` MUST 仅支持 `SiteAssign` 与 `SiteReturn`。`SiteExpr` 与 `SiteStmt` MUST 拒绝展开。

#### Scenario: 表达式位置拒绝

- **WHEN** `WithExpand` 处理 `_ = 1 + With(g())` 且 `ctx.Site()` 为 `SiteExpr`
- **THEN** MUST 返回错误，且错误信息 MUST 含 `expression position` 或等价说明

#### Scenario: 语句位置拒绝

- **WHEN** `WithExpand` 处理 `With(closer())` 作为独立语句（`SiteStmt`）
- **THEN** MUST 返回错误

### Requirement: With 必须使用 Stmts

在 `SiteAssign` 与 `SiteReturn` 语境，`WithExpand` MUST 设置显式 `CallExpandResult.Target`（分别为 `SpliceReplaceAssignStmt`、`SpliceReplaceReturnStmt`）并返回非空 `Stmts`。MUST NOT 使用 `SpliceReplaceReturnResults` 或仅设置 `Exprs`。

#### Scenario: SiteReturn 禁止 Exprs

- **WHEN** `WithExpand` 处理 `return With(g())` 且 `ctx.Site()` 为 `SiteReturn`
- **THEN** 返回的 `CallExpandResult` MUST 含 `Stmts` 且 MUST NOT 仅设置 `Exprs`

### Requirement: With 宏展开语义

`WithExpand` MUST 将 `With` 调用展开为：

1. 多返回值赋值（`_v, _err := callee`）；
2. `if _err != nil { return <与外层签名一致的零值与 err> }`；
3. `defer func() { _ = _v.Close() }()`（错误路径 MUST NOT 含 defer）；
4. 按语境完成成功赋值或 `return`（与 `try.Try` k=1 语义一致）。

#### Scenario: 展开 os.Open 赋值

- **WHEN** 外层为 `func Read(path string) ([]byte, error)` 且为 `f := With(os.Open(path))`
- **THEN** 展开后 MUST 在错误时 `return nil, err`（或外层等价零值），成功路径 MUST 含 `defer func() { _ = <temp>.Close() }()`，且 MUST 将资源赋给 `f`

#### Scenario: return With 完整处理

- **WHEN** 外层为 `func f() (*os.File, error)` 且源码为 `return With(g())`，`g()` 为 `(*os.File, error)`
- **THEN** `WithExpand` MUST 返回 `Stmts` 替换整条 `return`，含赋值、`if err != nil`、defer、成功 `return <resource>, nil`

#### Scenario: 错误路径无 defer

- **WHEN** 展开 `f := With(g())` 且 `g()` 可能返回非 nil error
- **THEN** `if _err != nil { ... }` 分支 MUST NOT 包含 `defer Close`

### Requirement: 具名返回支持

错误路径上的 `return` MUST 优先使用外层具名标识符（若存在）；无具名时 MUST 使用与外层签名一致的零值与 `err` 变量。行为 MUST 与 `syntax-try` 的具名返回规则一致。

#### Scenario: 具名 error

- **WHEN** 外层为 `func f() (data []byte, err error)` 且为 `With` 赋值语境
- **THEN** 错误路径 MUST `return nil, err`（使用具名 `err`）

### Requirement: 非法调用错误信息

非法用法 MUST 在展开期失败，错误 MUST 含文件、行号、原因及修复提示。

#### Scenario: 多载荷提示

- **WHEN** callee 为 `(A, B, error)` 但用户调用 `With(f())`
- **THEN** `WithExpand` MUST 返回错误，且错误信息 MUST 说明 v1 仅支持 `(T, error)`

### Requirement: mactest 单测

`WithExpand` MUST 具备不依赖 `//go:build macro` 的 `mactest.ExpandCall` 单测，覆盖：`SiteAssign` 正常展开（含 defer）、`SiteReturn`、`具名返回`、拒绝 `SiteExpr`、拒绝无 error 外层函数、拒绝多载荷 callee。

#### Scenario: 纯 Expand 测试

- **WHEN** 在 `go-macro-contrib` 仓库内执行 `go test ./with/...`
- **THEN** 测试 MUST 无需全链路 `cmd/macro expand` 即可通过
