## 1. 包迁移与桩 API

- [x] 1.1 将 `derivestringer/` 目录重命名为 `derive/`，包声明改为 `package derive`
- [x] 1.2 定义泛型 marker：`type Derive[T any] struct{}`，doc 含 `//macro: derive`
- [x] 1.3 将桩方法改为 `func (Derive[T]) String() string { panic(...) }`
- [x] 1.4 将 `DeriveStringerExpand` 重命名为 `DeriveExpand`，更新 `//macro: derive` 注释

## 2. Expander 逻辑

- [x] 2.1 在 `DeriveExpand` 入口添加类型实参校验（`len == 1` 且 `types.Identical` 于 `fmt.Stringer`），错误信息含 `must be fmt.Stringer`
- [x] 2.2 将 `stringSelectionIsDeriveStringerStub` 重命名并适配 `MarkerTypeName == "Derive"` 的泛型 stub 判定
- [x] 2.3 确认展开语义不变：字段遍历、`shouldGenerateString`、`quote.Decls` 生成 `String()`、全量 `Fields`/`Methods` 返回

## 3. 测试

- [x] 3.1 更新 `derive_test.go`：嵌入改为 `Derive[fmt.Stringer]`，syntax-id 改为 `derive`，调用 `DeriveExpand`
- [x] 3.2 保留并通过既有场景：正常展开、用户自有 `String()`、Helper 嵌入提升 `String()`
- [x] 3.3 新增错误类型实参用例（如 `Derive[int]`、`Derive[any]`）断言 expand 失败且错误信息含 `fmt.Stringer`
- [x] 3.4 执行 `go test ./derive/...` 确认通过

## 4. 文档与规格落地

- [x] 4.1 更新 `README.md`：derive 包用法、`Derive[fmt.Stringer]` 示例、syntax-id `derive`、迁移对照表
- [x] 4.2 联调范围内更新 `go-macro/docs/author-guide.md` 中的 derive 示例（若存在 `derivestringer` 引用）
- [x] 4.3 归档变更后确认 `openspec/specs/syntax-derive/spec.md` 生效、`syntax-derive-stringer` 已移除

## 5. 收尾验证

- [x] 5.1 全仓库检索并清除 `derivestringer`、`derive-stringer`、`DeriveStringer` 残留引用
- [x] 5.2 执行 `go test ./...` 确认 contrib 全量测试通过
