## 1. 语法桩

- [x] 1.1 在 `inline/stubs.go` 增加 `Inline0`、`Inline2`、`Inline3`（保留 `Inline`），签名与 `try` 多桩风格一致
- [x] 1.2 为各桩补充 `//macro: syntax-inline` 与「勿运行时调用」注释

## 2. 解析与内联核心

- [x] 2.1 实现 `innerCallFromArgs(call *ast.CallExpr)`：从宏调用实参归一化内层 `*ast.CallExpr`（含 `Inline0` 的 `func()` 包装提取）
- [x] 2.2 实现 `calleeResultCount` 与 `checkStubMatchesN(stub, n)`
- [x] 2.3 实现 `resolveCalleeFuncDecl`、`isInlineableFunc`（按 `n` 校验 `return` 列表）
- [x] 2.4 实现 `substituteParams`（支持 `n` 个 return 表达式或语句体）
- [x] 2.5 在 `InlineExpand` 中分支：`n=1`+`SiteExpr`→`Expr`；`n=0`+`SiteStmt`→`Stmts`；`n=2/3`+`SiteAssign`→`Stmts`；unwrap/报错

## 3. 测试

- [x] 3.1 `TestInlineExpandInlinesLocalCall`（`Inline` / 单值 / `SiteExpr`）
- [x] 3.2 `TestInline2ExpandAssignSite`（双返回值 + `SiteAssign`）
- [x] 3.3 `TestInline0ExpandStmtSite`（`Inline0(func(){...})` + `SiteStmt`）
- [x] 3.4 `TestInlineStubMismatch`（双值 callee 用 `Inline` 报错）
- [x] 3.5 保留/更新 unwrap 与不可内联错误测试；`go test ./inline/...`

## 4. 文档

- [x] 4.1 更新 `README.md`：四桩对照表、`Inline0` 的 `func()` 写法、0～3 返回值示例
- [x] 4.2 更新 `inline/expand.go` 注释
