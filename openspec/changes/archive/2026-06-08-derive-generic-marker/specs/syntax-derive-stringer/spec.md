## REMOVED Requirements

### Requirement: derive-stringer 语法桩

**Reason**: 由 `syntax-derive` 取代；marker 改为泛型 `derive.Derive[fmt.Stringer]`，syntax-id 改为 `derive`。

**Migration**: 将 `derivestringer.DeriveStringer` 改为 `derive.Derive[fmt.Stringer]`；import `github.com/arcane-craft/go-macro-contrib/derive`；`RegisterDecl("derive", derive.DeriveExpand)`。

### Requirement: derive-stringer 展开语义

**Reason**: 合并入 `syntax-derive` 的 `derive 展开语义` requirement；Expander 重命名为 `DeriveExpand`。

**Migration**: 链接 `DeriveExpand` 替代 `DeriveStringerExpand`；展开行为（字段拼接 String、删桩、尊重用户 String）不变。

### Requirement: derive-stringer 作用域

**Reason**: 由 `syntax-derive` 的 `derive 作用域` requirement 承接。

**Migration**: 无额外操作。

### Requirement: 可选官方宏库与引入方式

**Reason**: 包路径由 `derivestringer` 改为 `derive`，由 `syntax-derive` 定义。

**Migration**: 更新 import 与 RegisterDecl syntax-id 为 `derive`。

### Requirement: mactest 单测

**Reason**: 测试包路径与 API 变更，由 `syntax-derive` 的 mactest requirement 定义。

**Migration**: `go test ./derive/...` 替代 `./derivestringer/...`。
