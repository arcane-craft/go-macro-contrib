## 1. 准备与联调环境

- [x] 1.1 在本地 `go.mod` 添加 `replace github.com/arcane-craft/go-macro => ../go-macro`（不提交）
- [x] 1.2 确认 sibling `go-macro` 为 `feature-synatx-rule` 分支且 `go test ./...` 通过
- [x] 1.3 删除各包对 `macro/quote`、`CallContext`、`DeclContext` 的 import 引用（迁移过程中逐步清零）
- [x] 1.4 补全 change delta spec：`syntax-try` / `syntax-with` / `syntax-wire-json` 旧 API 术语清扫；`syntax-derive` 增加 Apply E2E 要求
- [x] 1.5 决议非法 call site 错误语义（design D11 / 路径 B）：`try`/`with` 仅 StmtPattern，非法位置断言 `no matching syntax rule`
- [x] 1.6 细化 derive `shouldGenerateString` 重建策略与测试模型（design D6.1 / D6.2；tasks 6.3a–d）

## 2. try 包迁移

- [x] 2.1 实现 `var TryExpander = macro.SyntaxCase(...)`：仅 StmtPattern（赋值 / `var` / `return` / `Try0;`），宽 pattern 优先；**不加** CallPattern catch-all（D11）
- [x] 2.2 将 `outerResults`、`calleePayloadCount`、`zeroValueExpr` 改写为 `EnclosingResults` / `ZeroSyntax`；`quote.Stmts` → `macro.Quote`
- [x] 2.3 更新 `//macro:` 标注为 `TryExpander`；删除 `TryExpand` 函数
- [x] 2.4 迁移 `try/expand_test.go` 至 `mactest.ExpandSyntax`；删除或改写 `try/fake_test.go`；非法表达式位置用例断言 `no matching syntax rule`（非 `expression position`，D11）
- [x] 2.5 `go test ./try/...` 通过

## 3. with 包迁移

- [x] 3.1 实现 `var WithExpander = macro.SyntaxCase(...)`（仅 StmtPattern：赋值 + return；非法 expr/stmt 靠无匹配失败，D11）
- [x] 3.2 复用或移植 try 的 error-handling 辅助；`macro.Quote` 生成 defer Close
- [x] 3.3 更新 `//macro:` 与测试至 `ExpandSyntax`；处理 `with/fake_test.go`；expr/stmt 拒绝用例断言 `no matching syntax rule`（D11）
- [x] 3.4 `go test ./with/...` 通过

## 4. inline 包迁移

- [x] 4.1 实现 `var InlineExpander`：少量 clause + Transform 内按 invoked name 分发
- [x] 4.2 可选：`SyntaxRules` unwrap 子路径（`Inline($inner)` → `#inner`）
- [x] 4.3 将 binding 来源从 `ctx.Call()` / `ctx.StubName()` 改为 `binds.Get("inner")` 与 pattern invoked name
- [x] 4.4 保留 AST 代入核心（`resolveCalleeFuncDecl`、`substituteExpr` 等），适配 `macro.Context` + `macro.Syntax`
- [x] 4.5 迁移测试至 `mactest.ExpandSyntax`；`go test ./inline/...` 通过

## 5. wirejson 包迁移

- [x] 5.1 实现 `var WireJSONExpander = macro.SyntaxCase`，pattern `type $item struct { WireJSON $field ... }`
- [x] 5.2 Transform 设置 json tag；`omitempty` 从 embed field tag 读取
- [x] 5.3 迁移测试至 `mactest.Expand`；断言 `ToDecls()` 或 Apply 后 TypeSpec
- [x] 5.4 `go test ./wirejson/...` 通过

## 6. derive 包迁移

- [x] 6.1 实现 `var DeriveExpander = macro.SyntaxCase`，pattern `type $item struct { Derive[$iface] $field ... }`；`validateStringerTypeArg` 改用 `binds.Get("iface")` + `importer.Default()` / file imports（无 `ctx.Package()`）
- [x] 6.2 `deriveTransform`：`macro.Quote` 绑定 `binds.Elems("field")` 产出 `TypeSpec'`；按 `shouldGenerateString` 可选 append 生成 `String()`；**不**复制既有 methods 进 `out`
- [x] 6.3a `targetDeclaresString(file, typeName)`：扫描文件 receiver 为 `T` / `*T` 的 `func String`
- [x] 6.3b `otherEmbedPromotesString(structFields, deriveAnchor, types.Info)`：完整 struct 字段遍历，skip anchor Derive 嵌入
- [x] 6.3c `deriveStubOnlyPromotesString(namedType, deriveAnchor, types.Info)`：method set 判定 `String` 是否仅来自 Derive 桩（禁简化）
- [x] 6.3d 组合 `shouldGenerateString(ctx, site, binds)`；表驱动单测覆盖 generate / 用户 String / Helper embed 三路径（可先不跑完整 Expander）
- [x] 6.4 迁移测试至 `mactest.Expand`：生成路径断言 `out.ToDecls()`；skip 路径 + `Foo` 保留 MUST `ValidateSplice` + `Apply` E2E（见 design D6.2）
- [x] 6.5 `go test ./derive/...` 通过

## 7. 收尾与发布

- [x] 7.1 根目录 `go test ./...` 全绿
- [x] 7.2 更新 `README.md`：最低兼容 `go-macro` 版本（与 bump 后 `go.mod` 一致）；维护者文档中 Expander 符号由 `*Expand` 改为 `*Expander`（如 `TryExpand`→`TryExpander`、`DeriveExpand`→`DeriveExpander`、`DeriveStringerExpand` 迁移表同步）
- [ ] 7.3 待 `go-macro` 发含 syntax-rules 的 tag：bump `require`、移除本地 `replace`、打 contrib tag
- [ ] 7.4 archive change：`openspec archive contrib-syntax-rules-migration`
