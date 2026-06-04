# syntax-derive-stringer Specification

## Purpose

定义官方 `derivestringer` 声明宏库的 marker、`DeriveStringerExpand` 展开语义及 mactest 要求。实现位于 `github.com/arcane-craft/go-macro-contrib/derivestringer`。

## Requirements

### Requirement: derive-stringer 语法桩

`derive-stringer` syntax MUST 提供 marker 类型 `DeriveStringer`（无类型参数），其类型 doc MUST 含 `//macro: derive-stringer`。

`DeriveStringer` MUST 提供 `func (DeriveStringer) String() string` 桩方法，函数体 MUST panic，并注明勿在未展开时调用。匿名嵌入后，Target 在宏主文件的类型检查与分析阶段 MUST 通过方法提升获得 `String()`，从而满足 `fmt.Stringer` 等依赖；展开后由 `DeriveStringerExpand` 生成 Target 自身的 `func (T) String() string` 并移除嵌入桩。

宏主文件中 Target MUST 通过 **匿名嵌入** `derivestringer.DeriveStringer`（或等价 import 路径下的类型名）触发。`DeriveStringer` MUST NOT 要求类型实参。

#### Scenario: 嵌入后类型检查具备 Stringer

- **WHEN** `type Item struct { derivestringer.DeriveStringer; A string }` 且宏主文件经 `go/types` 分析
- **THEN** `Item` MUST 实现 `fmt.Stringer`（经 `DeriveStringer.String` 提升），且文件中 MUST NOT 存在 `func (Item) String() string` 声明

#### Scenario: 合法使用

- **WHEN** `type Item struct { derivestringer.DeriveStringer; A string; B int }` 且已 link `DeriveStringerExpand`
- **THEN** MUST 识别为 `derive-stringer` 站点

### Requirement: derive-stringer 展开语义

`DeriveStringerExpand`（或等价 `DeclExpander`）MUST：

1. 从 `site.Target` 读取 struct 字段（不含已嵌入的 `DeriveStringer`）；
2. 返回全量 `Fields`：原业务字段，**不含** `DeriveStringer` 嵌入；
3. 返回全量 `Methods`：Target 在展开前已有的、receiver 为 `T` 的全部 `*ast.FuncDecl` 方法；**仅当** Target 尚无自有 `String()` 时，追加生成的 `func (T) String() string { ... }`。自有 `String()` 包括：文件中 `func (T) String() string` 声明，或由**非** `DeriveStringer` 的嵌入类型提升而来的 `String()`（须用 `go/types` 方法集判定；仅由 marker 桩提升的 `String()` 不算自有）。

生成的 `String()` 实现 MUST 使 `T` 满足 `fmt.Stringer`（由 syntax 文档描述字段拼接规则；框架不内置格式）。

#### Scenario: 生成 String 并删桩

- **WHEN** 对合法 `Item` 展开成功
- **THEN** 结果 `Fields` MUST NOT 含 `DeriveStringer`，且 `Methods` MUST 含 `func (Item) String() string`

#### Scenario: 已有 String 则保留用户实现

- **WHEN** `Item` 已有 `func (Item) String() string`，或经非 marker 嵌入提升得到 `String()`，且展开成功
- **THEN** `Methods` MUST NOT 含新生成的 `func (Item) String() string`，`Fields` MUST NOT 含 `DeriveStringer`，且用户既有 `String()` 行为 MUST 保留

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
