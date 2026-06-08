## Context

`go-macro-contrib` 的 `try/expand.go` 与 `derivestringer/derive.go` 通过手写 `go/ast` 构造展开结果。`go-macro` 已在 sibling 仓库实现 `macro/quote`（模板 + `#hole` 填洞 → `ast.Expr` / `[]ast.Stmt` / `[]ast.Decl`），author-guide 已文档化与 `CallExpandResult` / `DeclExpandResult` 的衔接。

本次变更 **仅替换 AST 拼装层**；`syntax-try` 与 `syntax-derive-stringer` 对用户可见语义不变。开发期通过本地 `replace github.com/arcane-craft/go-macro => ../go-macro` 使用 quote；已提交 `go.mod` 保持 `require v0.3.0`。

## Goals / Non-Goals

**Goals:**

- `try.TryExpand` 用单一通用 `quote.Stmts` 模板表达「assign callee → if err → success」控制流
- `derivestringer.DeriveStringerExpand` 用 `quote.Decls` + `buildSprintfCall()` 生成 `String()` 方法
- 临时变量继续 `ctx.TempIdent()`，零值继续 `zeroValueExpr`，注入 quote 绑定表
- 现有 `mactest` 全部通过；删除被替代的手写 AST 辅助函数

**Non-Goals:**

- 迁移 `inline`、`wirejson`
- 改变桩签名、syntax-id、splice `Target` 或展开产物形状
- 新增 quote 专项单测或 golden 测试
- 更新 README
- 在已提交 `go.mod` 中 bump `go-macro` 版本（留待核心发版）

## Decisions

### D1：try 使用单一通用模板

**选择**：一个 `quote.Stmts` 模板，通过绑定表动态注入 LHS（k+1 个 ident）、RHS（callee expr）、`errorResults`（`[]ast.Expr`）、`success`（`ast.Stmt` 或 fast-path `[]ast.Stmt`）。

**理由**：与用户确认一致；k 与 Site 差异由 Go 代码在调用 quote 前组装绑定，避免 k=0..3 × Site 的模板矩阵。

**备选**：按 Site 或 k 拆分多模板——更直观但重复多，未采纳。

### D2：临时名与零值仍由 Go 辅助函数产出

**选择**：`ctx.TempIdent("_err")` / `ctx.TempIdent("_v")` 生成 idents；`zeroValueExpr` + `outerResults` 组装 `errorResults` 再注入 `#errorResults`。

**理由**：避免模板写死名字导致与用户代码冲突；零值依赖 `types.Type`，不适合搬进静态模板。

### D3：derivestringer 的 Sprintf 调用手写构建

**选择**：保留（或提取）`buildSprintfCall(format string, fields []ast.Expr) ast.Expr`，将整段 `ast.Expr` 注入 `quote.Decls` 模板中的 `#body` 或等价洞。

**理由**：`fmt.Sprintf` 的 variadic 实参列表由字段遍历动态决定，整段 expr 注入比模板内展开 `#field1, #field2, ...` 更简单可靠。

### D4：Quote 错误包装为 macro.ErrorAt

**选择**：`quote.Stmts` / `quote.Decls` 返回的 parse/bind 错误 MUST 经 `macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%v", err)` 上报。

**理由**：与现有 Expander 错误体验一致。

### D5：贴回契约不变

**选择**：Quote 成功后仍显式设置 `CallExpandResult.Target` / `DeclExpandResult` 字段，Call 宏仍调用 `macro.StampStmtPos(ctx.MacroPos(), stmts)`。

**理由**：`macro/quote` 不设置 SpliceTarget，框架契约不变。

### D6：依赖与版本策略

**选择**：实现与联调阶段本地 `replace`；已提交 `go.mod` 保持 `require github.com/arcane-craft/go-macro v0.3.0`。`go-macro` 发布含 `macro/quote` 的 tag 后，单独变更 bump `require` 并移除 `replace`。

**理由**：符合 `macro-contrib` spec 的发布约束与用户确认。

## Risks / Trade-offs

- **[Risk] `#errorResults` 绑定 `[]ast.Expr` 在 `return` 中展开行为不符预期** → 实现前对 try 核心模板做 spike；若 quote 不支持，改为 `#err` + 手写 `ReturnStmt.Results` 仅 success 路径用 quote
- **[Risk] 删除 `errorReturnIf` 后遗漏 Site 边角（Try0 + SiteReturn 仅 assign+if）** → 完全依赖现有 mactest 覆盖，不删减测试
- **[Risk] 本地 replace 与 CI（无 sibling go-macro）** → CI 仍 pin v0.3.0 且无 quote import 时编译失败；须在 merge 前确保 go-macro 已发版并 bump require，或本 PR 与核心发版同批次
- **[Trade-off] 模板抽象度 vs 可读性** → 统一模板在 Expander 内需少量绑定组装代码，但核心控制流一眼可读

## Migration Plan

1. 本地 `go.mod` 添加 `replace => ../go-macro`
2. 改写 `try/expand.go`，跑 `go test ./try/...`
3. 改写 `derivestringer/derive.go`，跑 `go test ./derivestringer/...`
4. 根目录 `go test ./...`
5. 移除已无引用的手写 AST 辅助函数
6. **发布前**（独立步骤）：go-macro 发 tag → bump `require` → 去掉 `replace` → contrib tag

**回滚**：恢复手写 AST 实现；不 import `macro/quote` 即可。

## Open Questions

（无——范围与实现策略已在 propose 前与用户确认。）
