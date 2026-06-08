## Context

`derivestringer` 当前提供无类型参数的 marker `DeriveStringer` 与 syntax-id `derive-stringer`。宏主文件通过匿名嵌入触发 `DeriveStringerExpand`，展开后生成 `String()` 并移除桩。

go-macro v0.4.0 的 `DeclSite` 已含 `MarkerTypeArgs []types.Type`，引擎对 `Derive[fmt.Stringer]` 类嵌入会解析基础名 `Derive` 与实参列表。`try`、`inline` 已采用泛型桩模式，Decl 宏侧尚无先例。

用户已确认：幻影类型参数 `T any`、展开期严格校验 `fmt.Stringer`、包改名为 `derive`、syntax-id 改为 `derive`、Expander 改名为 `DeriveExpand`。

## Goals / Non-Goals

**Goals:**

- 提供 `derive.Derive[fmt.Stringer]` 作为 Decl 宏 marker，调用点自文档化
- 展开语义与现 `derivestringer` 一致（字段拼接 `String()`、stub 判定、跳过用户自有 `String()`）
- 错误类型实参在 expand 时返回含 `must be fmt.Stringer` 的明确错误
- 统一 `derive` 包命名，为未来多 trait 派生预留包结构

**Non-Goals:**

- 本次不实现 `Derive[json.Marshaler]` 等其他 trait
- 不保留 `derivestringer` 兼容层或 deprecated alias
- 不修改 go-macro 框架（`DeclSite`、registry 已够用）
- 不改变 `String()` 生成格式（仍为 `fmt.Sprintf` 字段拼接）

## Decisions

### 1. 幻影类型参数 `Derive[T any]`

**选择**：`type Derive[T any] struct{}`，不在类型参数上加 `interface{ String() string }` 约束。

**理由**：`Derive[fmt.Stringer]` 的 T 表示「要派生的 trait 类型本身」，而非「实现该 trait 的具体类型」。幻影参数 + expander 校验与 Rust trait marker 语义更接近。

**备选**：`Derive[T interface{ String() string }]` — 拒绝，因 `fmt.Stringer` 作为类型实参时与约束直觉不一致，且无法表达未来 `json.Marshaler` 等异构 trait。

### 2. 展开期校验类型实参

**选择**：`DeriveExpand` 入口检查 `len(site.MarkerTypeArgs) == 1` 且 `site.MarkerTypeArgs[0]` 与 `fmt.Stringer` 接口类型一致（`types.Identical` 或比较 `*types.Interface` 方法集）。

**理由**：`Derive[int]`、`Derive[any]` 等可编译（桩 `String()` 仍提升），必须在 expand 阶段拒绝并提示合法实参。

### 3. stub 判定适配泛型 receiver

**选择**：`stringSelectionIsDeriveStub`（自 `stringSelectionIsDeriveStringerStub` 重命名）继续比较 `site.MarkerTypeName == "Derive"` 与 `site.EmbedIndex`；`methodReceiverNamedType` 对泛型实例化 receiver 仍返回基础名 `Derive`。

**理由**：框架 `embeddedMarker` 已剥离类型实参，仅上报 `baseName`。需在实现后跑 mactest 验证 `go/types` 方法集路径无回归。

### 4. 包与 API 破坏性迁移

**选择**：`derivestringer/` 目录直接改名为 `derive/`，删除旧包名；syntax-id `derive`；`DeriveExpand` 注册名与 `//macro: derive` 对齐。

**理由**：用户明确不要 re-export 兼容层；一次性迁移成本可控（contrib 仓库内自洽，下游文档同步更新）。

### 5. 规格归档策略

**选择**：新增 `syntax-derive` 规格，归档时删除 `syntax-derive-stringer`；`macro-contrib` 用 delta 更新路径表。

## Risks / Trade-offs

- **[Risk] 泛型 stub 的 method set 判定与现逻辑不兼容** → 保留并扩展 mactest：正常展开、跳过自有 String、Helper 嵌入、错误实参四类用例
- **[Risk] 下游未迁移 import / RegisterDecl** → README、author-guide、macro-contrib spec 明确 **BREAKING** 与迁移对照表
- **[Risk] `fmt.Stringer` 接口类型比较因 import 路径别名失败** → 用 `types.Info` 解析 `fmt.Stringer` 的 `*types.Interface`，与 `MarkerTypeArgs[0]` 做 `types.Identical`，不比较 AST 字符串
- **[Trade-off] 错误实参可编译** → 接受；与 `T any` 幻影参数设计一致，靠 expand 报错兜底

## Migration Plan

1. 实现 `derive` 包并删除 `derivestringer`
2. 更新 contrib 内所有引用（README、openspec、测试）
3. 联调时同步 `go-macro/docs/author-guide.md` 示例
4. 发布说明列出对照：

   | 旧 | 新 |
   |----|-----|
   | `.../derivestringer` | `.../derive` |
   | `DeriveStringer` | `Derive[fmt.Stringer]` |
   | `derive-stringer` | `derive` |
   | `DeriveStringerExpand` | `DeriveExpand` |

5. 无运行时 rollback；git revert 即可回退

## Open Questions

（无——探索阶段决策已闭合。）
