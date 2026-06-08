## Why

Go 中获取 `io.Closer` 类资源时，开发者需重复书写 `f, err := open(); if err != nil { return ... }; defer func() { _ = f.Close() }()` 样板。`try` 宏已消除 error 检查，但未自动注册清理。新增 `with` 宏可在展开阶段将 `(T, error)` 获取与 `defer Close()` 合并为单次赋值，减少遗漏 `defer` 的风险，语义接近 Python `with` 的 acquire/cleanup 部分（不含块级语法）。

## What Changes

- 新增官方宏包 `with/`（syntax-id `syntax-with`），import 路径 `github.com/arcane-craft/go-macro-contrib/with`
- 新增语法桩 `With[T io.Closer](v T, err error) T`，映射 `WithExpand`
- `WithExpand` 在 `try` 同等 error 传播语义基础上，于成功路径插入 `defer func() { _ = <resource>.Close() }()`
- v1 仅支持 k=1（单载荷 + error）callee；不提供 `With0`/`With2`/`With3`
- 允许 `SiteAssign` 与 `SiteReturn`；拒绝 `SiteExpr` 与 `SiteStmt`
- 新增 `mactest.ExpandCall` 单测与 README 宏参考条目
- 更新 `macro-contrib` 规格中的官方宏库路径表

## Capabilities

### New Capabilities

- `syntax-with`：定义 `with` 包 `With` 桩、`WithExpand` 展开语义（try 式 error 处理 + defer Close）、`io.Closer` 约束、Site 约束及 mactest 要求

### Modified Capabilities

- `macro-contrib`：官方宏库路径表新增 `with` 包；独立测试范围新增 `with` 的 `mactest.ExpandCall`

## Impact

- **代码**：新建 `with/stubs.go`、`with/expand.go`、`with/expand_test.go`、`with/expand_errors_test.go`；更新 `README.md`
- **规格**：新建 `openspec/specs/syntax-with/spec.md`；delta 更新 `macro-contrib/spec.md`
- **依赖**：无新外部依赖；复用 `go-macro` Call API 与 `macro/quote`（实现 MAY 参考 `try` 包内部逻辑，但 `with` MUST NOT 要求用户 import `try`）
- **兼容性**：纯 additive，无 BREAKING 变更
