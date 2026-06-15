## Context

`go-macro-contrib` 当前 pin `github.com/arcane-craft/go-macro v0.4.0`，五个官方宏通过 `CallExpander` / `DeclExpander` 返回 `CallExpandResult` / `DeclExpandResult`，并显式设置 `SpliceTarget`。`try`、`with`、`derive` 使用已删除的 `macro/quote` 子包。

Sibling 仓库 `go-macro` 的 `feature-synatx-rule` 分支已实现：

- 统一 `macro.Expander func(Context, Syntax) (Syntax, error)`
- `macro.SyntaxRules` / `macro.SyntaxCase`（`Clause`：Pattern + Template / Transform / Fender）
- pattern match 写入 site meta 槽 → 引擎 `ValidateSplice` + `Apply`（作者不再设置 `SpliceTarget`）
- `macro.Quote`（`#` 洞）；`macro.EnclosingResults` / `ZeroSyntax`
- `expandtool.Register(syntaxID, expander)` 统一注册；**无** Decl adapter
- `mactest.ExpandSyntax`（Call）、`mactest.Expand`（Decl）

开发约束（用户已确认）：本地 `replace => ../go-macro`；迁移顺序 try → with → inline → wirejson → derive；OpenSpec proposal 先行。

## Goals / Non-Goals

**Goals:**

- 五个宏全部迁移为 `var XxxExpander = macro.SyntaxCase(...)`（或 inline unwrap 子路径用 `SyntaxRules`）
- 用户可见展开语义与 v0.6.0 一致；现有 mactest 场景等价覆盖
- `derive` 的 `shouldGenerateString` 行为与 v0.6.0 对齐（扫描 AST / method set，非简化版）
- 测试迁移至 `ExpandSyntax` / `Expand`；删除对 `fakeCallContext` 的依赖或改为对新 API 的薄封装
- OpenSpec delta 与实现同步；发布前 bump `go-macro` require

**Non-Goals:**

- 新增宏能力、修改桩签名或 syntax-id
- 修改 `go-macro` 核心代码（除联调 bugfix）
- 在已提交 `go.mod` 中提前写入 `replace`
- 等待 `macro.FieldList` core helper（derive 用手动 `macro.Quote` 绑定 `binds.Elems("field")`）

## Decisions

### D1：依赖与发布策略

**选择**：实现与联调阶段在本地 `go.mod` 添加 `replace github.com/arcane-craft/go-macro => ../go-macro`；已提交 `go.mod` 保持当前 tag 直至 `go-macro` 发布含 syntax-rules 的版本，再单独 bump `require` 并移除 `replace` 后打 contrib tag。

**理由**：符合 `macro-contrib` spec 与用户确认；避免 CI 在无 sibling 时解析失败。

### D2：Expander 导出名为 `XxxExpander`

**选择**：`TryExpander`、`WithExpander`、`InlineExpander`、`DeriveExpander`、`WireJSONExpander` 均为包级 `var`，值为 `macro.SyntaxCase(...)`。

**理由**：与 `go-macro` author-guide 及 `init provider` 脚手架一致；`//macro:` 标注写在 var 的 doc 上。

### D3：迁移顺序

**选择**：try → with → inline → wirejson → derive。

**理由**：try/with 共享 error-handling 模式，可先验证 Call `SyntaxCase` + `Quote`；inline 复杂度高但独立于 Decl；wirejson 作为 Decl 练手；derive 语义最细，放最后。

### D4：try / with — 多 clause SyntaxCase

**选择**：宽 Stmt pattern 优先，例如：

- `$lhs ... := Try($inner)` / `$lhs ... = Try($inner)` / `var $lhs ... = Try($inner)`
- `return $vals ... , Try($inner)`
- `Try0($inner);`（语句）
- invoked name 区分 `Try0`/`Try`/`Try2`/`Try3`

Transform 内用 `macro.EnclosingResults`、`macro.ZeroSyntax`、`macro.Quote` 生成 `out`；语句路径在 `ToStmts()` 后 `macro.StampStmtPos(site.MacroPos(), stmts)`。

**非法 call site**：`try` / `with` **MUST NOT** 为拒绝表达式/语句位置而添加 CallPattern catch-all（见 **D11**）。仅依赖 StmtPattern 不匹配时由 `runSyntaxCase` 返回 `no matching syntax rule`。

**理由**：与 `go-macro` `pattern_match_test.go` 及 author-guide 一致；`findAssignStmt` 由 `$lhs` 绑定替代；避免 CallPattern 误匹配 `return Try(g())` 导致 ReturnResults 贴回（违背 Stmts 完整 error handling 要求）。

**备选**：单 clause + 手写 site 检测——违背 syntax-rules 模型，未采纳。

**备选**：CallPattern + Fender 主动返回 `expression position`——保留旧错误文案，但增加 clause 且易与合法 return 路径冲突，未采纳（见 D11）。

### D5：inline — 少量 clause + Transform 分发

**选择**：约 6–10 个 clause 覆盖合法 site×桩组合；共享 `inlineTransform` 内按 pattern match 的 invoked name（`Inline0`/`Inline`/`Inline2`/`Inline3`）与 `binds.Get("inner")` 分发。unwrap 路径可单独 `SyntaxRules{Pattern: "Inline($inner)", Template: "#inner"}` + Fender 约束。

**理由**：避免 12+ clause 爆炸；保留现有 AST 代入函数（`resolveCalleeFuncDecl`、`substituteExpr` 等），仅改 binding 来源。

**桩与 n 校验**：由 pattern invoked name 限定（`Inline2` pattern 不匹配 `Inline` 调用）；callee 返回值个数仍用 `types.Info` 校验并报错。

### D6：derive — SyntaxCase + 保持 v0.6.0 语义

**选择**：

```go
Pattern: `type $item struct { Derive[$iface] $field ... }`
Transform: deriveTransform
```

- 类型实参：`binds.Get("iface")` + `ctx.Types().TypeOf`；`fmt.Stringer` 解析 MUST 用 `importer.Default()` 或 `site.(macro.FileCarrier).ExpansionFile()` imports（MUST NOT 依赖 `ctx.Package()`）
- 产出 TypeSpec 字段：`binds.Elems("field")`，每项 `Underlying()` 为 `*ast.Field`（**不含** Derive 嵌入）
- 输出：`macro.Quote` 手动注入 field 列表 → `out.ToDecls()` = `[TypeSpec']` 或 `[TypeSpec', 新生成 String()]`（仅当 `shouldGenerateString` 为真）
- **不**在 out 中复制既有 methods；引擎保留文件中未在 `out` 出现的 methods（含用户 `String()`、`Foo()` 等）

**理由**：用户要求可观测语义不变；新 API 下 `DeclExpandResult` / `TargetMethods()` / `DeclSite` 元数据槽已删除。

#### D6.1：`shouldGenerateString` 重建（迁移核心难点）

旧实现依赖三项已删除 API，新实现 MUST 用 AST + `go/types` 自行重建等价判定：

| 旧 API / 逻辑 | 用途 | 新实现路径 |
|---------------|------|------------|
| `ctx.TargetMethods()` | 文件中已有 `func (T) String()` | 扫描 `ExpansionFile()` 中 receiver 为 `T` / `*T` 的 `*ast.FuncDecl`，存在 `Name == "String"` → 不生成 |
| `site.EmbedIndex` + `otherEmbedPromotesString` | 其它匿名嵌入提升 `String()` | 从 `binds.Get("item")` 取完整 `StructType.Fields`；对每个匿名字段 `f`，若 `f == deriveEmbedAnchor`（`site.Underlying().(*ast.Field)`）则 skip；否则若 `types.NewMethodSet(ptr(f.Type)).Lookup(String) != nil` → 不生成 |
| `site.MarkerTypeName` + `stringSelectionIsDeriveStub` | method set 中 `String` 仅来自 Derive 桩提升 | 对 `*T` 的 method set 中每个 `String` selection：若存在**非** Derive 匿名嵌入路径或**非** MethodVal 提升 → 不生成；若全部 `String` 均来自 anchor Derive 字段的桩提升 → 生成 |

**Derive 嵌入身份**：以 pattern match 的 embed anchor（`site.Underlying().(*ast.Field)`）指针与 struct 匿名字段比对，**MUST NOT** 硬编码字段下标（embed 与 `$field` 书写顺序须等价）。

**禁止简化**：「method set 含 `String` 则不生成」——会把「仅 Derive 桩提升」误判为自有，导致删桩后丢失 `fmt.Stringer`。

**推荐实现顺序**（与 tasks 6.3a–d 对应）：先写纯函数 + 表驱动单测（snippet + `types.Info`），再接入 `deriveTransform`。

#### D6.2：测试与断言模型变更

| 场景 | 旧断言（`ExpandDecl` + `DeclExpandResult`） | 新断言（`Expand` + `Syntax`） |
|------|---------------------------------------------|-------------------------------|
| 正常生成 String | `result.Methods` 含 `String` | `out.ToDecls()` 含生成 `String`；Apply 后 TypeSpec 无 Derive |
| 用户自有 String | `result.Methods` 恰有 1 个 `String`（用户的） | `out.ToDecls()` **无**生成 `String`；Apply 后文件仍含用户 `String` |
| Helper 嵌入提升 String | `result.Methods` **无** `String` | `out.ToDecls()` **无**生成 `String`；Apply 后 `T` 仍满足 Stringer（Helper 提升） |
| 保留其它 method | `result.Methods` 含 `Foo` | `out` 可不含 `Foo`；Apply E2E 后文件仍含 `Foo` |

skip 场景与 method 保留 **MUST** 用 `ValidateSplice` + `Apply` 端到端验收，不可仅查 `out.ToDecls()`。

### D7：wirejson — Decl SyntaxCase

**选择**：

```go
Pattern: `type $item struct { WireJSON $field ... }`
```

Transform 设置 json tag；`omitempty` 从 `site.Underlying().(*ast.Field).Tag` + `macro.ParseMacroTag` 读取。out 仅含新 TypeSpec，不返回 methods。

### D8：共享辅助

**选择**：在 `try` 包内保留 `outerResults`/`calleePayloadCount` 等逻辑，改写为接受 `macro.Context` + `macro.Syntax`；`with` 可 import 未导出辅助 **或** 复制最小子集（优先 internal 包 `macrocontrib` 仅当重复超阈值——首版 with 复制 try 辅助以避免过早抽象）。

**理由**：最小 diff；重复可在后续 refactor 提取。

### D9：测试策略

**选择**：

- Call：`mactest.ExpandSyntax(XxxExpander, stubName, syntaxID, snippet)` → `out.ToStmts()` / `ToExpr()`，断言与现测试等价
- Decl：`mactest.Expand(DeriveExpander, syntaxID, snippet)` → `out.ToDecls()`；关键用例可加 `ValidateSplice` + `Apply` 端到端
- 删除 `fakeCallContext` 或对 error path 改用 snippet + `ExpandSyntax` 期望 error

### D10：Quote 迁移

**选择**：`quote.Stmts(tpl, args)` → `macro.Quote(tpl, map[string]macro.Syntax)`；`#` 洞绑定 `macro.WrapExpr` / `WrapStmts` 等。

**理由**：`macro/quote` 子包已删除。

### D11：非法 call site 错误语义（路径 B）

**选择**：`try` / `with` 对表达式位置（如 `_ = 1 + Try(g())`）、`with` 的独立语句位置（`With(f());`）等非法语境，**不**在 expander 内主动返回 `expression position` / `not allowed as statement`；仅配置 StmtPattern clause，由 `macro.SyntaxCase` 在全部 clause 匹配失败后返回框架统一错误 `macro: no matching syntax rule`（常附带末条原因，如 `assign stmt not found`、`unsupported call parent context`）。

**理由**：与 `go-macro` `macro-rules` spec 一致；实现最简单；避免为「友好拒绝」添加 CallPattern 导致 `return Try(...)` 走 ReturnResults 贴回降级。`inline` 不受此约束——表达式位置为合法 unwrap/内联路径，仍使用 CallPattern。

**迁移**：`try/expand_test.go`、`with/expand_errors_test.go` 等非法 site 用例 MUST 断言 `no matching syntax rule`，MUST NOT 再要求 `expression position` 子串。

## Risks / Trade-offs

- **[Risk] go-macro 未发 tag 导致 CI 失败** → 合并前与 core 发版同批次，或临时 CI 跳过直至 bump
- **[Risk] derive `shouldGenerateString` 重写复杂度高**（D6.1）→ 按 6.3a–d 分步实现并表驱动单测；禁止「method set 有 String 即 skip」简化；skip/保留场景用 Apply E2E
- **[Risk] derive 新 API 下 method 保留语义与旧 `TargetMethods` 复制不等价** → `out` 仅含增量；全量跑 derive 测试 + 对照 v0.6.0；`Foo` 与自有 `String` 用 Apply 断言
- **[Risk] inline Transform 分发遗漏 site×桩组合** → 保留现有 mactest 全量场景；缺 coverage 则补 clause
- **[Risk] `ctx.Package()` 删除影响 `io.Closer` / `fmt.Stringer` 查找** → 使用 `importer.Default()` 或从 `site.(FileCarrier).ExpansionFile()` 扫 imports（与 derive 现有 `fmtStringerType` 模式一致）
- **[Trade-off] with 与 try 代码重复** → 首版接受；后续可提取 internal 包
- **[Trade-off] 非法 site 错误文案弱化**（D11）→ 展开仍失败，但用户可见 `no matching syntax rule` 而非 `expression position`；与 core syntax-rules 一致，可接受

## Migration Plan

1. 创建本 change 制品（proposal / design / specs / tasks）
2. 本地 `go.mod` 添加 `replace => ../go-macro`
3. 按顺序迁移 try → with → inline → wirejson → derive + 测试
4. `go test ./...` 全绿
5. `go-macro` 发含 syntax-rules 的 tag → bump `require` → 移除 `replace` → 更新 README → contrib tag
6. archive change，合并 delta specs

**回滚**：恢复旧 Expander 实现并 pin `go-macro v0.4.0`（需 core 旧 tag 仍可用）。

## Open Questions

（实现前已决议，无阻塞项。）
