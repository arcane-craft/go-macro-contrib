## MODIFIED Requirements

### Requirement: With 语法桩

`with` 包 MUST 提供单一语法桩 `With[T io.Closer](v T, err error) T`。函数体 MUST panic，并注明勿在未展开时调用。桩 MUST 带 `//macro: with` 并映射到 `WithExpand`。

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
- **THEN** 展开管线 MUST NOT 注册 `with`

#### Scenario: import 但未 link 时不展开

- **WHEN** 宏主文件 import `github.com/arcane-craft/go-macro-contrib/with`，但 expand 工具 `linked` 未含该 path
- **THEN** 对 `With(...)` 的调用 MUST NOT 被展开

### Requirement: 具名返回支持

错误路径上的 `return` MUST 优先使用外层具名标识符（若存在）；无具名时 MUST 使用与外层签名一致的零值与 `err` 变量。行为 MUST 与 `try` 的具名返回规则一致。

#### Scenario: 具名 error

- **WHEN** 外层为 `func f() (data []byte, err error)` 且为 `With` 赋值语境
- **THEN** 错误路径 MUST `return nil, err`（使用具名 `err`）
