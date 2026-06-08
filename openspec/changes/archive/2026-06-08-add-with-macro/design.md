## Context

`go-macro-contrib` 已有 `try` 宏：将 `(T, error)` callee 展开为赋值 + `if err != nil { return ... }` + 成功赋值。获取 `io.Closer` 资源时，用户仍需手写 `defer func() { _ = f.Close() }()`。Python `with` 的 acquire/cleanup 语义在 Go 中常对应「try 式 error 检查 + defer Close」，但不宜要求块级语法（go-macro v0.4.0 无 `SpliceReplaceBlockStmt`）。

本变更新增独立 `with` 包，在 `try` 同等控制流上于**成功路径**插入 defer，v1 仅支持 k=1（`With` 单桩）。

## Goals / Non-Goals

**Goals:**

- 提供 `With[T io.Closer](v T, err error) T` 桩与 `WithExpand`，展开语义 = `try.Try` + `defer func() { _ = res.Close() }()`
- 支持 `SiteAssign`（`f := With(os.Open(...))`）与 `SiteReturn`（`return With(...)`）
- 复用 `try` 包的展开模式（`outerResults`、`calleePayloadCount`、`quote` 模板、`SpliceReplace*Stmt`），实现上 MAY 复制或抽取共享逻辑，但 `with` 包 MUST 自包含、不要求用户 import `try`
- 宏主文件通过 `io.Closer` 泛型约束完成类型检查
- `mactest.ExpandCall` 覆盖正常展开、错误路径、非法 Site、非 Closer callee

**Non-Goals:**

- 块级 `with { ... }` 语法
- `With0` / `With2` / `With3` 多桩族（后续按需扩展）
- 自定义清理方法名（仅 `Close() error`，即 `io.Closer`）
- Close 错误向上传播或与 body error 合并
- 修改 `go-macro` 核心 API

## Decisions

### 1. 独立 `with` 包，syntax-id `syntax-with`

与 `try`/`inline` 并列，README 与 `macro-contrib` 路径表各增一行。语义边界清晰：`try` = error 传播；`with` = error 传播 + 自动 defer Close。

**备选**：扩展 `try` 包增加 `TryClose` 桩 —— 拒绝，混淆「要不要 defer」的选型。

### 2. v1 仅单桩 `With`（k=1）

callee MUST 为 `(T, error)` 且 `T` 满足 `io.Closer`。展开期用 `types.Implements(payloadType, io.Closer)` 二次校验；泛型约束不足以覆盖所有动态类型场景。

**备选**：首版 mirror 整个 `Try` 族 —— 拒绝，多载荷时「defer 哪一个」规则未定，YAGNI。

### 3. 展开语句序列（SiteAssign）

相对 `try.Try` 在 `if err != nil` 与成功赋值之间插入 defer：

```
1. _v, _err := <callee>
2. if _err != nil { return <outer zeros>, _err }
3. defer func() { _ = _v.Close() }()
4. <lhs> := _v          // 原 AssignStmt 左值
```

错误路径**不**注册 defer（资源未成功获取）。

`SiteReturn`：在错误分支之后、成功 `return` 之前插入 defer；成功 return 携带 `_v` 与 `nil` error（与 `try` return 路径一致）。

### 4. defer 形态

固定使用 `defer func() { _ = <res>.Close() }()`，不用裸 `defer res.Close()`，避免 staticcheck 等对丢弃 Close error 的警告。

Close 返回值 MUST 以 `_ =` 丢弃；v1 不向上传播 Close error。

### 5. Site 与 Splice 约束

| Site | 行为 |
|------|------|
| `SiteAssign` | `SpliceReplaceAssignStmt`，非空 `Stmts` |
| `SiteReturn` | `SpliceReplaceReturnStmt`，非空 `Stmts` |
| `SiteExpr` | 拒绝，错误信息含 `expression position` |
| `SiteStmt` | 拒绝（无绑定变量，defer 无意义） |

与 `try` 相同：外层函数返回列表 MUST 以 `error` 结尾；内层 callee 的 `error` MUST 在返回列表末尾。

### 6. 实现策略

- `with/expand.go`：`WithExpand` 主入口；内部函数命名与 `try` 对齐（`outerResults`、`calleePayloadCount`、`withQuoteTemplate` 等）
- `withQuoteTemplate`：在 `tryQuoteTemplate` 基础上于 `if` 块之后追加 defer 模板行
- 返回前 `macro.StampStmtPos(ctx.MacroPos(), stmts)`
- **不** import `try` 包（避免 contrib 包间耦合）；允许复制 ~100 行共享辅助函数

### 7. 桩签名

```go
//macro: syntax-with
func With[T io.Closer](v T, err error) T {
    panic("With is a macro stub and must not be called at runtime")
}
```

doc 注明：宏桩，运行时勿调用；用于 `(T, error)` 获取且 `T` 为 `io.Closer` 的场景。

## Risks / Trade-offs

- **[Risk] 用户误以为 `with` 提供块级作用域** → README 明确对比 Python `with`，示例展示后续语句仍在同一函数作用域
- **[Risk] 与 `try` 功能重叠** → 文档说明选型：`try` 用于不需自动 Close 的值；`with` 用于 `io.Closer`
- **[Risk] 复制 `try` 辅助函数导致漂移** → 测试对齐 `try` 的错误路径行为；未来可抽取 `contrib/internal/errexpand`（非 v1）
- **[Risk] `return With(...)` 在 defer 前返回资源，调用方仍持有引用** → 文档说明 defer 在函数退出时执行，与手写 defer 一致
- **[Risk] 非 `io.Closer` 但有关闭需求的类型** → v1 不支持；用户继续手写 defer 或使用 `try`

## Migration Plan

1. 实现 `with/` 包与测试
2. 更新 README 宏参考表
3. 归档 change 时合并 `openspec/specs/syntax-with/spec.md` 与 `macro-contrib` delta
4. 无下游 BREAKING；用户按需 `go get` 并 import `with`

## Open Questions

- 无（v1 范围已在 explore 阶段确认：方向 B、k=1、io.Closer、独立包）
