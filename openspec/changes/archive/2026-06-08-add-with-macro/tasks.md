## 1. 语法桩

- [x] 1.1 新建 `with/stubs.go`：`With[T io.Closer](v T, err error) T`，`//macro: syntax-with`，panic 桩与文档注释
- [x] 1.2 确认 expand 工具可自动 link `WithExpand`（无需 register 包）

## 2. WithExpand 核心

- [x] 2.1 新建 `with/expand.go`：`WithExpand` 入口，校验单实参、stub 名 `With`、k=1
- [x] 2.2 实现 `outerResults`、`calleePayloadCount`、`isErrorType`（对齐 `try` 语义）
- [x] 2.3 实现 `io.Closer` 展开期校验（`types.Implements`）
- [x] 2.4 实现 `withQuoteTemplate`：try 式赋值 + `if err != nil` + `defer func() { _ = res.Close() }()` + 成功赋值/return
- [x] 2.5 实现 `SiteAssign` / `SiteReturn` 分支，拒绝 `SiteExpr` / `SiteStmt`；设置 `SpliceReplaceAssignStmt` / `SpliceReplaceReturnStmt`；`macro.StampStmtPos`

## 3. 测试

- [x] 3.1 `TestWithExpandAssign`：`f := With(open())` 含 defer、错误分支、成功赋值
- [x] 3.2 `TestWithExpandReturn`：`return With(g())` 含 defer 与成功 return
- [x] 3.3 `TestWithExpandNamedReturn`：具名 `err` 错误路径
- [x] 3.4 `TestWithExpandRejectExprSite`、`TestWithExpandRejectNoErrorReturn`、`TestWithExpandRejectMultiPayload`
- [x] 3.5 `expand_errors_test.go` 覆盖非法 callee / 非 Closer；`go test ./with/...`

## 4. 文档与规格

- [x] 4.1 更新 `README.md`：宏表新增 `with`、用法示例、与 `try` 选型说明
- [x] 4.2 更新 README 维护者「包与 syntax-id」表
- [x] 4.3 全仓 `go test ./...` 通过
