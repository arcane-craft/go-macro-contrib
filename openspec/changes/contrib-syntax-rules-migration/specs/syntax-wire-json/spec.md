## MODIFIED Requirements

### Requirement: wire-json 语法桩

`wire-json` syntax MUST 提供 marker 类型 `WireJSON`（无类型参数），其类型 doc MUST 含 `//macro: wire-json`。

宏主文件中 Target MUST 通过 **匿名嵌入** `wirejson.WireJSON` 触发。可选参数 MUST 仅通过嵌入字段 `` `macro:"..."` `` 传递（如 `omitempty=Role`）；引擎 MUST NOT 读取 `WireJSON` 类型内 struct 字段。站点 MUST 由 Decl pattern `type $item struct { WireJSON $field ... }` 识别。

#### Scenario: 带 tag 可选参数

- **WHEN** `type Profile struct` 含匿名嵌入 `WireJSON` 且嵌入字段带 `` `macro:"omitempty=Role"` ``
- **THEN** MUST 识别为 `wire-json` 站点，且 Transform MUST 经 `macro.ParseMacroTag` 从 embed 字段 tag 读取 `omitempty=Role`

### Requirement: wire-json 展开语义

`WireJSONExpander`（`macro.SyntaxCase` + Transform）MUST：

1. 产出 `out.ToDecls()[0]` 为不含 `WireJSON` 嵌入的 `TypeSpec`；为各业务字段设置或合并 `json` struct tag（tag 键名为字段名小写；`omitempty` 由 embed 字段 `macro` tag 指定字段名列表）；
2. **不**在 `out` 中返回既有 methods；文件中未 match 的 Target methods MUST 由引擎自动保留。

若字段已含 `json` tag 且与 wire-json 规则冲突，Expander MUST 按 syntax 文档策略处理（默认 MUST 返回 error）。

#### Scenario: 补全 json tag

- **WHEN** 对 `type User struct { wirejson.WireJSON; ID int64; Name string }` 展开并成功 Apply
- **THEN** TypeSpec 中 `ID`、`Name` MUST 含 `json` tag，且 MUST NOT 含 `WireJSON` 嵌入

#### Scenario: 全量 Methods 保留

- **WHEN** `User` 在展开前已有 `func (User) Validate() error`
- **THEN** Apply 后文件中 MUST 仍含 `Validate` 方法

### Requirement: 可选官方宏库与引入方式

`wirejson` 包 MUST 在 `go-macro-contrib` 仓库内发布，路径为 `github.com/arcane-craft/go-macro-contrib/wirejson`。使用方 MUST import 该路径，且 expand 须 link `wire-json` 的 `WireJSONExpander`，方可展开嵌入站点。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件嵌入 `WireJSON` 但未 import `github.com/arcane-craft/go-macro-contrib/wirejson`
- **THEN** 展开管线 MUST NOT 注册 `wire-json`

### Requirement: mactest 单测

`WireJSONExpander` MUST 具备不依赖 `//go:build macro` 的 `mactest.Expand` 单测，覆盖默认 tag、`omitempty` 与冲突 tag 错误路径。

#### Scenario: 纯 Expand 测试

- **WHEN** 在 `go-macro-contrib` 仓库内执行 `go test ./wirejson/...`
- **THEN** 测试 MUST 无需全链路 expand 即可通过

## ADDED Requirements

### Requirement: WireJSONExpander pattern

`wirejson` 包 MUST 使用 Decl pattern `type $item struct { WireJSON $field ... }`。可选 `macro` struct tag MUST 从 `site.Underlying().(*ast.Field).Tag` 经 `macro.ParseMacroTag` 读取。

#### Scenario: omitempty 来自 embed tag

- **WHEN** embed 字段为 `` WireJSON `macro:"omitempty=Role"` ``
- **THEN** Transform MUST 为 `Role` 字段应用 `json` omitempty
