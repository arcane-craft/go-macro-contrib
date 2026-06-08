## ADDED Requirements

### Requirement: derive 语法桩

`derive` syntax MUST 提供泛型 marker 类型 `Derive[T any]`，其类型 doc MUST 含 `//macro: derive`。

`Derive[T]` MUST 提供 `func (Derive[T]) String() string` 桩方法，函数体 MUST panic，并注明勿在未展开时调用。匿名嵌入 `Derive[fmt.Stringer]` 后，Target 在宏主文件的类型检查与分析阶段 MUST 通过方法提升获得 `String()`，从而满足 `fmt.Stringer`；展开后由 `DeriveExpand` 生成 Target 自身的 `func (T) String() string` 并移除嵌入桩。

宏主文件中 Target MUST 通过 **匿名嵌入** `derive.Derive[fmt.Stringer]`（或等价 import 路径下的实例化）触发。类型实参 MUST 为 `fmt.Stringer` 接口类型。

#### Scenario: 嵌入后类型检查具备 Stringer

- **WHEN** `type Item struct { derive.Derive[fmt.Stringer]; A string }` 且宏主文件经 `go/types` 分析
- **THEN** `Item` MUST 实现 `fmt.Stringer`（经 `Derive[fmt.Stringer].String` 提升），且文件中 MUST NOT 存在 `func (Item) String() string` 声明

#### Scenario: 合法使用

- **WHEN** `type Item struct { derive.Derive[fmt.Stringer]; A string; B int }` 且已 link `DeriveExpand`
- **THEN** MUST 识别为 `derive` 站点，且 `DeclSite.MarkerTypeName` MUST 为 `Derive`、`MarkerTypeArgs` MUST 含 `fmt.Stringer`

### Requirement: derive 类型实参校验

`DeriveExpand` MUST 在展开前校验 `site.MarkerTypeArgs`：长度 MUST 为 1，且唯一实参 MUST 与 `fmt.Stringer` 接口类型相同（`types.Identical`）。若不满足，MUST 返回 error，且错误信息 MUST 指明合法实参为 `fmt.Stringer`（例如含 `must be fmt.Stringer`）。

#### Scenario: 错误类型实参拒绝展开

- **WHEN** 嵌入 `derive.Derive[int]` 或 `derive.Derive[any]` 并调用 `DeriveExpand`
- **THEN** MUST 返回 error，且错误信息 MUST 指明类型实参必须为 `fmt.Stringer`

#### Scenario: 正确类型实参允许展开

- **WHEN** 嵌入 `derive.Derive[fmt.Stringer]` 且 Target 为合法 struct
- **THEN** 类型实参校验 MUST 通过并继续展开

### Requirement: derive 展开语义

`DeriveExpand`（或等价 `DeclExpander`）MUST：

1. 从 `site.Target` 读取 struct 字段（不含已嵌入的 `Derive[fmt.Stringer]`）；
2. 返回全量 `Fields`：原业务字段，**不含** `Derive` 嵌入；
3. 返回全量 `Methods`：Target 在展开前已有的、receiver 为 `T` 的全部 `*ast.FuncDecl` 方法；**仅当** Target 尚无自有 `String()` 时，追加生成的 `func (T) String() string { ... }`。自有 `String()` 包括：文件中 `func (T) String() string` 声明，或由**非** `Derive` marker 的嵌入类型提升而来的 `String()`（须用 `go/types` 方法集判定；仅由 marker 桩提升的 `String()` 不算自有）。

生成的 `String()` 实现 MUST 使 `T` 满足 `fmt.Stringer`（字段名与 `%v` 拼接，逗号分隔；框架不内置其它格式）。

#### Scenario: 生成 String 并删桩

- **WHEN** 对合法 `Item` 展开成功
- **THEN** 结果 `Fields` MUST NOT 含 `Derive` 嵌入，且 `Methods` MUST 含 `func (Item) String() string`

#### Scenario: 已有 String 则保留用户实现

- **WHEN** `Item` 已有 `func (Item) String() string`，或经非 marker 嵌入提升得到 `String()`，且展开成功
- **THEN** `Methods` MUST NOT 含新生成的 `func (Item) String() string`，`Fields` MUST NOT 含 `Derive` 嵌入，且用户既有 `String()` 行为 MUST 保留

### Requirement: derive 作用域

`derive` MUST NOT 生成包级声明、其它类型或测试文件。MUST 仅通过 `DeclExpandResult` 的 `Fields` 与 `Methods` 表达结果。

#### Scenario: 无包级产物

- **WHEN** `derive` 展开成功
- **THEN** gen 文件中 MUST 仅出现 Target 类型定义与 Target 的方法，无新增 `const` 块

### Requirement: 官方宏库与引入方式

`derive` 包 MUST 在 `go-macro-contrib` 仓库内发布，路径为 `github.com/arcane-craft/go-macro-contrib/derive`。使用方 MUST 在宏主文件中 import 该路径，且 expand 须 **link** `derive` 的 `DeriveExpand`（`expandtool.RegisterDecl`），方可展开嵌入站点。

#### Scenario: 未 import 时不展开

- **WHEN** 宏主文件嵌入 `Derive[fmt.Stringer]` 但未 import `github.com/arcane-craft/go-macro-contrib/derive`
- **THEN** 展开管线 MUST NOT 注册 `derive`

### Requirement: mactest 单测

`DeriveExpand` MUST 具备不依赖 `//go:build macro` 的 `mactest.ExpandDecl` 单测，覆盖：正常展开、用户自有 `String()`、非 marker 嵌入提升 `String()`、错误类型实参失败。

#### Scenario: 纯 Expand 测试

- **WHEN** 在 `go-macro-contrib` 仓库内执行 `go test ./derive/...`
- **THEN** 测试 MUST 无需全链路 expand 即可通过
