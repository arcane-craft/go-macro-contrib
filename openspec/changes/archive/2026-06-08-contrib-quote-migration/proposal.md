## Why

`try` 与 `derivestringer` 的 Expander  today 大量手写 `go/ast` 结构体拼装（assign、if err、return、method decl 等），可读性差且与最终 Go 代码距离远。`go-macro` 已在 `macro/quote` 子包提供模板化 AST 组装能力；在 **不改变展开语义** 的前提下，用 Quote 改写可降低维护成本，并与核心库 author-guide 推荐实践对齐。

## What Changes

- `try/TryExpand`：用 **一个通用** `quote.Stmts` 模板 + 动态注入（`ctx.TempIdent()`、`call`、`errorResults`、`success` 等）替换手写 assign / if err / success 分支的 AST 拼装；保留 `outerResults`、`calleePayloadCount`、`zeroValueExpr` 等语义分析逻辑
- `derivestringer/DeriveStringerExpand`：用 `buildSprintfCall()` 产出 `fmt.Sprintf(...)` 的 `ast.Expr` 并注入 `quote.Decls` 模板生成 `String()` 方法；保留字段遍历与 `shouldGenerateString` 逻辑
- **不**改动 `inline`、`wirejson`
- **不**改变宏调用语法、桩签名、`CallExpandResult` / `DeclExpandResult` 形状或 `mactest` 可观测行为
- 开发期通过本地 `replace github.com/arcane-craft/go-macro => ../go-macro` 对接未发布的 `macro/quote`；**已提交** `go.mod` 保持 `require v0.3.0`，待核心发版后再 bump
- **不**更新 README（实现细节，终端用户无感）

## Capabilities

### New Capabilities

（无——本次为内部实现重构，不引入新的用户可见能力。）

### Modified Capabilities

（无——`syntax-try` 与 `syntax-derive-stringer` 的展开语义与 mactest 要求不变，仅 Expander 实现方式变更。）

## Impact

- **代码**：`try/expand.go`、`derivestringer/derive.go`；删除被 Quote 替代的手写 AST 辅助函数（如 `errorReturnIf`）
- **依赖**：实现期本地 `replace` 到含 `macro/quote` 的 sibling `go-macro`；发布前须 bump `require` 至含 quote 的 `go-macro` tag 并移除 `replace`
- **测试**：现有 `mactest.ExpandCall` / `mactest.ExpandDecl` MUST 继续通过，不新增 quote 专项单测
- **API**：无公开 API 或 **BREAKING** 变更
