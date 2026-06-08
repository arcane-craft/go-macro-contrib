## Why

`derivestringer.DeriveStringer` 把「派生什么 trait」隐含在包名里，扩展性差，调用点也无法自文档化。go-macro Decl 宏已支持泛型 marker 与 `MarkerTypeArgs`；将语法桩改为 `derive.Derive[fmt.Stringer]` 可在调用处显式声明派生目标，并为未来统一 `derive` 包（多种 trait 派生）奠定基础。

## What Changes

- **BREAKING**：`derivestringer/` 包重命名为 `derive/`，import 路径变为 `github.com/arcane-craft/go-macro-contrib/derive`
- **BREAKING**：marker 类型由 `DeriveStringer` 改为泛型 `Derive[T any]`，调用处匿名嵌入 `derive.Derive[fmt.Stringer]`
- **BREAKING**：syntax-id 由 `derive-stringer` 改为 `derive`
- **BREAKING**：Expander 由 `DeriveStringerExpand` 重命名为 `DeriveExpand`
- 新增展开期校验：`MarkerTypeArgs[0]` 必须是 `fmt.Stringer` 接口；否则返回明确错误（如 `derive: type argument must be fmt.Stringer`）
- 保留 `Derive[T]` 上的 `String()` panic 桩，供宏主文件展开前类型检查与方法提升
- 展开语义不变：生成字段拼接版 `String()`、移除嵌入桩、尊重用户自有 `String()` 及非 marker 嵌入提升
- 更新 `README.md`、`openspec` 规格及（联调范围内）`go-macro/docs/author-guide.md` 中的示例与路径

## Capabilities

### New Capabilities

- `syntax-derive`：定义 `derive` 包泛型 marker、`DeriveExpand` 展开语义、类型实参校验及 mactest 要求（取代 `syntax-derive-stringer`）

### Modified Capabilities

- `macro-contrib`：官方宏库路径表、syntax-id、测试路径由 `derivestringer` 更新为 `derive`
- `syntax-derive-stringer`：整份规格由 `syntax-derive` 取代（归档删除）

## Impact

- **代码**：`derivestringer/` → `derive/`（目录改名）；`derive.go`、`derive_test.go` API 与实现更新
- **规格**：新建 `openspec/specs/syntax-derive/spec.md`；删除 `syntax-derive-stringer/spec.md`；更新 `macro-contrib/spec.md`
- **下游**：所有 import `derivestringer`、`RegisterDecl("derive-stringer", ...)`、`DeriveStringerExpand` 的调用方须迁移
- **测试**：`go test ./derive/...` 替换 `./derivestringer/...`；新增错误类型实参的 expand 失败用例
- **依赖**：无新外部依赖；继续要求 go-macro Decl API（`MarkerTypeArgs`）
