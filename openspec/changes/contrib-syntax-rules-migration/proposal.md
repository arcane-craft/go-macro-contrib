## Why

`go-macro` 的 `feature-synatx-rule` 分支已完成 **syntax-rules** 重构：统一 `Expander(ctx, site) (Syntax, error)`，引入 `macro.SyntaxRules` / `macro.SyntaxCase`，并 **删除** `CallContext`、`DeclContext`、`CallExpandResult`、`DeclExpandResult`、`macro/quote` 等旧 API。`go-macro-contrib` 五个官方宏（`inline`、`try`、`with`、`derive`、`wirejson`）仍基于 v0.4.0 旧 Expander 实现，无法与新版核心联调或发布。

本次变更将 contrib 全部宏实现迁移至 `SyntaxCase`（及必要的 `SyntaxRules`）模型，对齐 `go-macro` author-guide，并在 **用户可见展开语义不变** 的前提下完成测试与 OpenSpec 更新。

**行为基线**：文中 **v0.6.0** 均指 **go-macro-contrib 当前已发布版本的可观测展开行为**（mactest 与合法调用点展开后 AST），**非** `go-macro` 核心版本号。实现期 `go.mod` 仍 pin 旧 core tag（如 v0.4.0），迁移验收以 contrib 现有测试场景等价覆盖为准。

## What Changes

- **BREAKING（provider 实现面）**：五个包的 Expander 由 `*Expand(ctx, ...) (CallExpandResult|DeclExpandResult, error)` 改为 `var XxxExpander = macro.SyntaxCase(...)`（统一 `macro.Expander` 签名）
- **BREAKING（expand link 名）**：`cmd/macro expand` 自动 link 的符号由 `TryExpand` 等改为 `TryExpander` 等（`//macro:` 标注同步更新）
- `try`、`with`：多 clause `SyntaxCase` + `macro.Quote`；`macro/quote` import 移除
- `inline`：少量 clause + Transform 内分发；保留 AST 代入核心逻辑
- `derive`、`wirejson`：Decl `SyntaxCase` + pattern `type $item struct { ... }`；`derive` 重写 `shouldGenerateString`（无 `TargetMethods`/`EmbedIndex`，见 design D6.1）并保持 v0.6.0 可观测语义；skip/保留场景测试改用 Apply E2E
- 测试：`mactest.ExpandCall` → `mactest.ExpandSyntax`；`mactest.ExpandDecl` → `mactest.Expand`；删除或改写 `fakeCallContext` 测试
- `go.mod`：开发期本地 `replace github.com/arcane-craft/go-macro => ../go-macro`；**已提交** `go.mod` 在 `go-macro` 发含 syntax-rules 的 tag 前保持旧 `require`，发版后 bump 并移除 `replace`
- OpenSpec：`macro-contrib` 与六个 `syntax-*` spec 更新为新 API 表述
- **不变**：宏桩签名、syntax-id、用户调用语法、合法调用点的展开后 AST 语义（mactest 可观测行为与 v0.6.0 对齐）
- **微变（仅非法 call site）**：`try`/`with` 在表达式或语句等非法位置的展开失败错误，由 `expression position` 等改为框架 `no matching syntax rule`（design D11 / 路径 B）；展开仍失败，不影响合法用法

## Capabilities

### New Capabilities

（无——本次为框架 API 迁移，不引入新的用户可见宏能力。）

### Modified Capabilities

- `macro-contrib`：Expander 注册/link 名、依赖的 `go-macro` 最低版本、`mactest` 辅助函数、`macro.Quote` 替代 `macro/quote`
- `syntax-try`：Expander 实现形态与贴回描述（`Syntax.ToStmts()` 替代 `CallExpandResult`）
- `syntax-with`：同上
- `syntax-inline`：同上；桩识别由 pattern invoked name 替代 `ctx.StubName()`
- `syntax-derive`：Decl Expander 形态、`Syntax.ToDecls()` 替代 `DeclExpandResult`；引擎自动保留未 match methods
- `syntax-wire-json`：Decl Expander 形态与 mactest API

## Impact

- **代码**：`inline/`、`try/`、`with/`、`derive/`、`wirejson/` 的 `expand.go`（或等价）、测试与 `fake_test.go`
- **依赖**：最低 `go-macro` 版本升至含 syntax-rules 的 semver tag（实现期 `replace` 联调 `../go-macro`）
- **测试**：全部 `*_test.go` 迁移至新 `mactest` API；`go test ./...` MUST 通过
- **文档**：`README.md` 最低兼容核心版本（发布时与 `go.mod` bump 同步）
- **外部**：使用方宏调用语法 **无** BREAKING；仅直接 import Expander 符号的测试/工具需改用 `XxxExpander`
