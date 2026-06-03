# syntax-derive-stringer Specification

## Purpose

定义官方 `derivestringer` 声明宏库的 marker、`DeriveStringerExpand` 展开语义及 mactest 要求。实现位于 `github.com/arcane-craft/go-macro-contrib/derivestringer`。

## Requirements

### Requirement: derive-stringer 语法桩

`derive-stringer` syntax MUST 提供 marker 类型 `DeriveStringer`（无类型参数），其类型 doc MUST 含 `//macro: derive-stringer`。

宏主文件中 Target MUST 通过 **匿名嵌入** `derivestringer.DeriveStringer`（或等价 import 路径下的类型名）触发。`DeriveStringer` MUST NOT 要求类型实参。

#### Scenario: 合法使用

- **WHEN** `type Item struct { derivestringer.DeriveStringer; A string; B int }` 且已 link `DeriveStringerExpand`
- **THEN** MUST 识别为 `derive-stringer` 站点

### Requirement: derive-stringer 展开语义

`DeriveStringerExpand`（或等价 `DeclExpander`）MUST：

1. 从 `site.Target` 读取 struct 字段（不含已嵌入的 `DeriveStringer`）；
2. 返回全量 `Fields`：原业务字段，**不含** `DeriveStringer` 嵌入；
3. 返回全量 `Methods`：包含为 Target 生成的 `func (T) String() string { ... }`，以及 Target 在展开前已有的、receiver 为 `T` 的全部方法（若作者策略为保留）。

`String()` 实现 MUST 使 `T` 满足 `fmt.Stringer`（由 syntax 文档描述字段拼接规则；框架不内置格式）。

若 Target 在展开前已存在 `String()` 方法，Expander MUST 按 syntax 文档策略处理（默认 MUST 返回错误，冲突由宏作者在此 syntax 内规定）。

#### Scenario: 生成 String 并删桩

- **WHEN** 对合法 `Item` 展开成功
- **THEN** 结果 `Fields` MUST NOT 含 `DeriveStringer`，且 `Methods` MUST 含 `func (Item) String() string`

#### Scenario: 已有 String 冲突

- **WHEN** `Item` 已有 `func (Item) String() string` 且 `DeriveStringerExpand` 策略为冲突报错
- **THEN** MUST 返回 error

### Requirement: derive-stringer 作用域

`derive-stringer` MUST NOT 生成包级声明、其它类型或测试文件。MUST 仅通过 `DeclExpandResult` 的 `Fields` 与 `Methods` 表达结果。

#### Scenario: 无包级产物

- **WHEN** `derive-stringer` 展开成功
- **THEN** gen 文件中 MUST 仅出现 `Item` 类型定义与 `Item` 的方法，无新增 `const` 块

### Requirement: 可选官方宏库与引入方式

`derivestringer` 包 MUST 在 `go-macro-contrib` 仓库内发布，路径为 `github.com/arcane-craft/go-macro-contrib/derivestringer`。使用方 MUST 在宏主文件中 import 该路径，且 expand 须 **link** `derive-stringer` 的 `DeriveStringerExpand`（`expandtool.RegisterDecl`），方可展开嵌入站点。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件嵌入 `DeriveStringer` 但未 import `github.com/arcane-craft/go-macro-contrib/derivestringer`
- **THEN** 展开管线 MUST NOT 注册 `derive-stringer`

### Requirement: mactest 单测

`DeriveStringerExpand` MUST 具备不依赖 `//go:build macro` 的 `mactest.ExpandDecl` 单测。

#### Scenario: 纯 Expand 测试

- **WHEN** 在 `go-macro-contrib` 仓库内执行 `go test ./derivestringer/...`
- **THEN** 测试 MUST 无需全链路 expand 即可通过
