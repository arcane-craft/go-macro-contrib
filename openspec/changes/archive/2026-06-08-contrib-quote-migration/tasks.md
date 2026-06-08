## 1. 环境与 spike

- [x] 1.1 本地 `go.mod` 添加 `replace github.com/arcane-craft/go-macro => ../go-macro`，确认 `macro/quote` 可解析
- [x] 1.2 Spike：验证 try 模板中 `#errorResults` 绑定 `[]ast.Expr` 在 `return` 语句内的展开行为；若不满足则调整 design 中的 fallback

## 2. try 迁移

- [x] 2.1 在 `try/expand.go` 引入 `macro/quote`，实现统一 `quote.Stmts` 模板与绑定表组装（含 `ctx.TempIdent`、`zeroValueExpr`、`success` 分支）
- [x] 2.2 保留语义分析逻辑（`outerResults`、`calleePayloadCount`、`checkStubMatchesK`、`findAssignStmt` 等），删除 `errorReturnIf` 等被替代的手写 AST 辅助函数
- [x] 2.3 确保 `StampStmtPos` 与 `SpliceTarget` 设置与迁移前一致
- [x] 2.4 运行 `go test ./try/...`，全部 mactest 通过

## 3. derivestringer 迁移

- [x] 3.1 提取或实现 `buildSprintfCall(format string, fields []ast.Expr) ast.Expr`
- [x] 3.2 用 `quote.Decls` 模板 + 上述 expr 注入替换手写 `*ast.FuncDecl` 拼装
- [x] 3.3 保留 `shouldGenerateString`、字段遍历与 `DeclExpandResult` 全量返回语义
- [x] 3.4 运行 `go test ./derivestringer/...`，全部 mactest 通过

## 4. 集成验证

- [x] 4.1 根目录 `go test ./...` 全绿
- [x] 4.2 确认 `inline`、`wirejson` 无意外改动
- [x] 4.3 确认已提交 `go.mod` 仍无 `replace`（replace 仅本地）；`require` 仍为 v0.3.0

## 5. 发布跟进（本 change 实现后可独立执行）

- [ ] 5.1 待 `go-macro` 发布含 `macro/quote` 的 tag 后，bump `require` 并移除本地 `replace`
- [ ] 5.2 打 `go-macro-contrib` 新 tag
