## MODIFIED Requirements

### Requirement: derive 语法桩

`derive` syntax MUST 提供泛型 marker 类型 `Derive[T any]`，其类型 doc MUST 含 `//macro: derive`。

`Derive[T]` MUST 提供 `func (Derive[T]) String() string` 桩方法，函数体 MUST panic，并注明勿在未展开时调用。匿名嵌入 `Derive[fmt.Stringer]` 后，Target 在宏主文件的类型检查与分析阶段 MUST 通过方法提升获得 `String()`，从而满足 `fmt.Stringer`；展开后由 `DeriveExpander` 生成 Target 自身的 `func (T) String() string` 并移除嵌入桩。

宏主文件中 Target MUST 通过 **匿名嵌入** `derive.Derive[fmt.Stringer]`（或等价 import 路径下的实例化）触发。类型实参 MUST 为 `fmt.Stringer` 接口类型。

#### Scenario: 嵌入后类型检查具备 Stringer

- **WHEN** `type Item struct { derive.Derive[fmt.Stringer]; A string }` 且宏主文件经 `go/types` 分析
- **THEN** `Item` MUST 实现 `fmt.Stringer`（经 `Derive[fmt.Stringer].String` 提升），且文件中 MUST NOT 存在 `func (Item) String() string` 声明

#### Scenario: 合法使用

- **WHEN** `type Item struct { derive.Derive[fmt.Stringer]; A string; B int }` 且已 link `DeriveExpander`
- **THEN** MUST 识别为 `derive` 站点，且 pattern 绑定 `$iface` MUST 对应 `fmt.Stringer` 类型实参

### Requirement: derive 类型实参校验

`DeriveExpander` MUST 在展开前校验 `binds.Get("iface")` 所指类型：MUST 与 `fmt.Stringer` 接口类型相同（`types.Identical`）。若不满足，MUST 返回 error，且错误信息 MUST 指明合法实参为 `fmt.Stringer`（例如含 `must be fmt.Stringer`）。

#### Scenario: 错误类型实参拒绝展开

- **WHEN** 嵌入 `derive.Derive[int]` 或 `derive.Derive[any]` 并调用 `DeriveExpander`
- **THEN** MUST 返回 error，且错误信息 MUST 指明类型实参必须为 `fmt.Stringer`

#### Scenario: 正确类型实参允许展开

- **WHEN** 嵌入 `derive.Derive[fmt.Stringer]` 且 Target 为合法 struct
- **THEN** 类型实参校验 MUST 通过并继续展开

### Requirement: derive 展开语义

`DeriveExpander`（`macro.SyntaxCase` + Transform）MUST：

1. 从 `binds.Elems("field")` 读取 struct 业务字段（pattern 已排除 `Derive` 嵌入）；
2. 产出 `out.ToDecls()[0]` 为不含 `Derive` 嵌入的 `TypeSpec`；
3. **仅当** Target 尚无自有 `String()` 时，在 `out.ToDecls()[1:]` 追加生成的 `func (T) String() string { ... }`。自有 `String()` 判定 MUST 与 v0.6.0 等价（含文件中 `func (T) String()`、非 marker 嵌入提升的 `String()`；仅 marker 桩提升不算自有）；
4. 文件中未在 `out` 出现的既有 methods（如 `func (T) Foo()`）MUST 由引擎自动保留（新 API 不要求 Transform 复制 `TargetMethods()`）。

生成的 `String()` 实现 MUST 使 `T` 满足 `fmt.Stringer`（字段名与 `%v` 拼接，逗号分隔；框架不内置其它格式）。

#### Scenario: 生成 String 并删桩

- **WHEN** 对合法 `Item` 展开并成功 Apply
- **THEN** 结果 TypeSpec MUST NOT 含 `Derive` 嵌入，且 MUST 含 `func (Item) String() string`

#### Scenario: 已有 String 则保留用户实现

- **WHEN** `Item` 已有 `func (Item) String() string`，或经非 marker 嵌入提升得到 `String()`，且展开成功
- **THEN** `out` MUST NOT 含新生成的 `func (Item) String() string`，TypeSpec MUST NOT 含 `Derive` 嵌入，且用户既有 `String()` MUST 保留

### Requirement: derive 作用域

`derive` MUST NOT 生成包级声明、其它类型或测试文件。MUST 仅通过 `out.ToDecls()` 表达 TypeSpec 变更与**新生成** methods。

#### Scenario: 无包级产物

- **WHEN** `derive` 展开成功
- **THEN** gen 文件中 MUST 仅出现 Target 类型定义与 Target 的方法，无新增 `const` 块

### Requirement: 官方宏库与引入方式

`derive` 包 MUST 在 `go-macro-contrib` 仓库内发布，路径为 `github.com/arcane-craft/go-macro-contrib/derive`。使用方 MUST 在宏主文件中 import 该路径，且 expand 须 **link** `derive` 的 `DeriveExpander`（`expandtool.Register`），方可展开嵌入站点。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件嵌入 `Derive[fmt.Stringer]` 但未 import `github.com/arcane-craft/go-macro-contrib/derive`
- **THEN** 展开管线 MUST NOT 注册 `derive`

### Requirement: mactest 单测

`DeriveExpander` MUST 具备不依赖 `//go:build macro` 的 `mactest.Expand` 单测，覆盖：正常展开、用户自有 `String()`、非 marker 嵌入提升 `String()`、错误类型实参失败。

#### Scenario: 纯 Expand 测试

- **WHEN** 在 `go-macro-contrib` 仓库内执行 `go test ./derive/...`
- **THEN** 测试 MUST 无需全链路 expand 即可通过

## ADDED Requirements

### Requirement: shouldGenerateString 判定（新 API）

`shouldGenerateString` MUST 在**不**使用 `TargetMethods()`、`DeclSite.EmbedIndex`、`DeclSite.MarkerTypeName` 的前提下，与 v0.6.0 可观测语义等价：

1. 若文件中存在 Target 的 `func (T) String()` 或 `func (*T) String()` 声明 → MUST 返回 false；
2. 若完整 struct（自 `binds.Get("item")`）中，除 pattern anchor Derive 嵌入外的其它匿名嵌入在 `go/types` 方法集中提升 `String()` → MUST 返回 false；
3. 若 `*T` 方法集中的 `String` **仅**来自 anchor Derive 嵌入字段的 marker 桩提升 → MUST 返回 true（MUST 生成）；
4. MUST NOT 使用「method set 含 `String` 则不生成」一类简化，以免误判仅 Derive 桩提升的情形。

Derive 嵌入身份 MUST 以 `site.Underlying().(*ast.Field)` 与 struct 匿名字段指针比对，MUST NOT 依赖固定字段下标。

#### Scenario: 仅 Derive 桩提升时应生成

- **WHEN** `type Item struct { Derive[fmt.Stringer]; A string }` 且无其它 `String` 来源
- **THEN** `shouldGenerateString` MUST 为 true

#### Scenario: 用户声明 String 时不生成

- **WHEN** 文件中已有 `func (Item) String() string` 且 struct 嵌入 `Derive[fmt.Stringer]`
- **THEN** `shouldGenerateString` MUST 为 false，且 `out.ToDecls()` MUST NOT 含新生成的 `String()`

#### Scenario: 非 marker 嵌入提升时不生成

- **WHEN** `type Item struct { Derive[fmt.Stringer]; Helper }` 且 `Helper` 提升 `String()`
- **THEN** `shouldGenerateString` MUST 为 false

### Requirement: DeriveExpander pattern

`derive` 包 MUST 使用 Decl pattern `type $item struct { Derive[$iface] $field ... }`（embed 与 `$field` 书写顺序等价）。

#### Scenario: field 绑定不含 embed

- **WHEN** `DeriveExpander` 对含 `Derive[fmt.Stringer]` 与具名字段的 struct 展开
- **THEN** `binds.Elems("field")` MUST 仅含业务字段，MUST NOT 含 Derive 嵌入

### Requirement: derive Apply 端到端断言

`derive` 包 MUST 对下列场景使用 `mactest.Expand` + `ValidateSplice` + `Apply` 端到端验收（不可仅断言 `out.ToDecls()`）：

- 用户自有 `func (T) String()`：`out` 无生成 `String`，Apply 后文件仍含用户实现；
- 非 marker 嵌入提升 `String()`：`out` 无生成 `String`，Apply 后 `T` 仍实现 `fmt.Stringer`；
- 含既有 `func (T) Foo()`：Apply 后文件仍含 `Foo`（验证引擎保留未在 `out` 中的 methods）。

#### Scenario: Apply 后保留用户 String

- **WHEN** 已有 `func (Item) String() string { return "custom" }` 且 `type Item struct { derive.Derive[fmt.Stringer]; A string }`，展开并成功 Apply
- **THEN** 文件中 MUST 仍含用户 `String()` 实现，且 MUST NOT 出现第二个生成的 `func (Item) String() string`

#### Scenario: Apply 后保留既有 method

- **WHEN** `type Item struct { derive.Derive[fmt.Stringer]; A string }` 且已有 `func (Item) Foo() int`，展开并成功 Apply
- **THEN** 文件中 MUST 仍含 `func (Item) Foo() int`，且 TypeSpec MUST NOT 含 `Derive` 嵌入
