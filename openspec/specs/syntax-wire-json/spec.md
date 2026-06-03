# syntax-wire-json Specification

## Purpose

定义官方 `wirejson` 声明宏库的 marker、`WireJSONExpand` 展开语义及 mactest 要求。实现位于 `github.com/arcane-craft/go-macro-contrib/wirejson`。

## Requirements

### Requirement: wire-json 语法桩

`wire-json` syntax MUST 提供 marker 类型 `WireJSON`（无类型参数），其类型 doc MUST 含 `//macro: wire-json`。

宏主文件中 Target MUST 通过 **匿名嵌入** `wirejson.WireJSON` 触发。可选参数 MUST 仅通过嵌入字段 `` `macro:"..."` `` 传递（如 `omitempty=Role`）；引擎 MUST NOT 读取 `WireJSON` 类型内 struct 字段。

#### Scenario: 带 tag 可选参数

- **WHEN** `type Profile struct` 含匿名嵌入 `WireJSON` 且嵌入字段带 `` `macro:"omitempty=Role"` ``
- **THEN** MUST 识别为 `wire-json` 站点且 `MacroTag` 含 `omitempty=Role`

### Requirement: wire-json 展开语义

`WireJSONExpand`（或等价 `DeclExpander`）MUST：

1. 返回全量 `Fields`：不含 `WireJSON` 嵌入；为各业务字段设置或合并 `json` struct tag（tag 键名为字段名小写；`omitempty` 由 `macro` tag 指定字段名列表）；
2. 返回全量 `Methods`：Target 展开前已有的、receiver 为 Target 的全部方法（原样带回，除非 syntax 明确替换）。

若字段已含 `json` tag 且与 wire-json 规则冲突，Expander MUST 按 syntax 文档策略处理（默认 MUST 返回 error）。

#### Scenario: 补全 json tag

- **WHEN** 对 `type User struct { wirejson.WireJSON; ID int64; Name string }` 展开成功
- **THEN** `Fields` 中 `ID`、`Name` MUST 含 `json` tag，且 `Fields` MUST NOT 含 `WireJSON`

#### Scenario: 全量 Methods 带回

- **WHEN** `User` 在展开前已有 `func (User) Validate() error`
- **THEN** 成功结果的 `Methods` MUST 仍含 `Validate` 方法

### Requirement: wire-json 作用域

`wire-json` MUST NOT 生成 `MarshalJSON`/`UnmarshalJSON`，除非未来 syntax 修订显式添加；本 requirement 下 MUST 仅修改 struct 字段 tag 与删除嵌入桩，不生成包级声明或其它类型。

#### Scenario: 仅改 tag

- **WHEN** `wire-json` 展开成功
- **THEN** MUST NOT 新增包级 `const`/`var` 或非 Target 方法

### Requirement: 可选官方宏库与引入方式

`wirejson` 包 MUST 在 `go-macro-contrib` 仓库内发布，路径为 `github.com/arcane-craft/go-macro-contrib/wirejson`。使用方 MUST import 该路径，且 expand 须 link `wire-json` 的 `WireJSONExpand`，方可展开嵌入站点。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件嵌入 `WireJSON` 但未 import `github.com/arcane-craft/go-macro-contrib/wirejson`
- **THEN** 展开管线 MUST NOT 注册 `wire-json`

### Requirement: mactest 单测

`WireJSONExpand` MUST 具备不依赖 `//go:build macro` 的 `mactest.ExpandDecl` 单测，覆盖默认 tag、`omitempty` 与冲突 tag 错误路径。

#### Scenario: 纯 Expand 测试

- **WHEN** 在 `go-macro-contrib` 仓库内执行 `go test ./wirejson/...`
- **THEN** 测试 MUST 无需全链路 expand 即可通过
