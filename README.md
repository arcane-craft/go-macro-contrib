# go-macro-contrib

[go-macro](https://github.com/arcane-craft/go-macro) 的官方宏语法包：在源码里写「宏调用」，用 CLI 展开成普通 Go，再照常 `go build` / `go test`。

| 宏 | 用途（一句话） |
|----|----------------|
| `inline` | 把同文件里可内联的小函数展开进调用处 |
| `try` | 把 `(T, error)` 等签名展开成带 `if err != nil` 的控制流 |
| `with` | 在 `try` 式 error 处理基础上自动 `defer Close()`（`io.Closer`） |
| `derive` | 为 struct 派生 `fmt.Stringer`（生成 `String()`） |
| `wirejson` | 为 struct 字段补全 `json` tag |

---

## 你想做什么？

| 目标 | 跳转到 |
|------|--------|
| 第一次在业务项目里用起来 | [5 分钟上手](#5-分钟上手) |
| 查某个宏怎么写、有哪些桩 | [宏参考](#宏参考) |
| 改本仓库、和 go-macro 联调 | [维护者](#维护者) |

要自己写宏或做框架接入，请看 [go-macro 宏作者指南](https://github.com/arcane-craft/go-macro/blob/master/docs/author-guide.md)。

---

## 5 分钟上手

下面用 `inline` 走通一遍：**安装 → 写宏主文件 → 展开 → 日常构建**。若你用的是 `try`、`derive` 或 `wirejson`，步骤相同，只需换掉 import 与宏调用（写法见 [宏参考](#宏参考)）。

### 你需要什么

- 一个已有的 Go module（业务项目根目录）
- 本机可执行 `go generate`（会顺带拉取 [go-macro](https://github.com/arcane-craft/go-macro) 的 `macro` CLI，不必单独安装二进制）

### 1. 安装语法包

在**业务 module 根目录**执行：

```bash
go get github.com/arcane-craft/go-macro-contrib@latest
```

### 2. 写宏主文件

在要用宏的包里新建或修改源文件，例如 `demo.go`：

```go
//go:build macro

//go:generate go run github.com/arcane-craft/go-macro/cmd/macro@latest expand .

package demo

import "github.com/arcane-craft/go-macro-contrib/inline"

func Answer() int {
	return inline.Inline(42)
}
```

这三行注释各自负责一件事：

| 注释 | 作用 |
|------|------|
| `//go:build macro` | 这份源码只在「宏编辑」时参与类型检查；日常构建读展开后的 `demo_macro_gen.go`，一般**不必**给 `go build` 加 `-tags macro` |
| `//go:generate ... expand .` | 在本包执行 `go generate` 时，自动调用上面的展开命令 |
| `import ".../inline"` | 展开器只处理你已 import 的宏包 |

### 3. 展开

在 `demo.go` **所在包目录**执行：

```bash
go generate .
```

CLI 会扫描该包的 import，生成/更新 `.gomacro/expand_runner`（建议加入 `.gitignore`），并写出 `demo_macro_gen.go`。

也可以手动展开（整模块可用 `./...`）：

```bash
go run github.com/arcane-craft/go-macro/cmd/macro@latest expand .
```

### 4. 提交与日常构建

若别人会通过 `go get` 拉你的模块，建议把宏主文件和 `*_macro_gen.go` 一并提交。之后照常：

```bash
go build ./...
go test ./...
```

走到这里，你应已经完成第一次成功展开。

### 常用命令

| 命令 | 什么时候用 |
|------|------------|
| `go generate .` | 在宏主文件所在包触发展开（依赖文件里的 `//go:generate`） |
| `go run github.com/arcane-craft/go-macro/cmd/macro@latest expand .` | 不经过 generate，直接展开当前 module（pattern 可写成 `./...`） |
| `go test ./...` | 在本仓库根目录验证各语法包（维护者） |

---

## 编辑器（gopls）

宏主文件带 `//go:build macro` 时，若 IDE 同时分析宏主文件和 `*_macro_gen.go`，可能出现「重复定义」之类的提示。

你可以在 gopls 里加上 `-tags=macro`，让编辑器**只按宏版本**做补全与类型检查：

```json
"gopls": {
  "buildFlags": ["-tags=macro"]
}
```

更多说明见 [go-macro README · gopls](https://github.com/arcane-craft/go-macro#gopls)。

---

## 宏参考

以下包都是**语法桩**：你在宏主文件里调用它们，展开后得到普通 Go。**运行时不要直接调用桩函数**（会 panic）。

### inline — 内联同文件小函数

`inline` 按被调函数的**返回值个数**（0～3）选桩，在展开阶段把同文件内可内联的调用替换为函数体（形参代入实参），语义接近编译器内联。

| 桩 | 返回值个数 | 典型写法 |
|----|------------|----------|
| `Inline0` | 0 | `inline.Inline0(func() { cleanup() })`（语句位置） |
| `Inline` | 1 | `return inline.Inline(add(1, 2))` |
| `Inline2` | 2 | `a, b := inline.Inline2(split())` |
| `Inline3` | 3 | 三值赋值或 `return` |

**首版可内联的函数需满足：** 同文件、直接标识符调用、函数体为单条 `return`（或 `n=0` 的无返回值语句体）、非 variadic。

`Inline` 对非函数调用的实参仍会 **unwrap**（去掉包装），例如 `inline.Inline(1 + g())` 在表达式位置等价于 `1 + g()`。

`Inline0` 因 Go 类型限制，须用 `func() { ... }` 包装无返回值调用；`Inline2` / `Inline3` 须在赋值或 `return` 语境使用。

### try — 展开 error 处理

`try` 族把 `(T, error)`、`(A, B, error)` 等签名展开成带 `if err != nil` 的控制流。按 callee 在 **error 之前有几个返回值** 选桩：

| 桩 | 典型 callee 签名 |
|----|------------------|
| `Try0` | `(error)` |
| `Try` | `(T, error)` |
| `Try2` | `(A, B, error)` |
| `Try3` | `(A, B, C, error)` |

例如 `try.Try(os.Open(...))` 对应 `(*os.File, error)`；若是 `(A, B, error)`，应写 `try.Try2(...)`。

### with — 获取资源并自动 defer Close

`with` 将 `(T, error)` 获取（`T` 须为 `io.Closer`）展开为：`if err != nil` 错误传播 + `defer func() { _ = res.Close() }()` + 赋值。不含块级语法；函数体其余语句由你照常编写。

| 桩 | 典型 callee 签名 |
|----|------------------|
| `With` | `(T, error)`，且 `T` 实现 `io.Closer` |

```go
f := with.With(os.Open(path))
return io.ReadAll(f)
```

**与 `try` 如何选：** 需要自动 `Close` 时用 `with.With`；不需清理或自行 `defer` 时用 `try.Try`。宏主文件须 `import "io"`（桩泛型约束 `io.Closer`）。

`With` 须在赋值或 `return` 语境使用；v1 仅支持单载荷 `(T, error)`。

### derive — 派生 fmt.Stringer

在 struct 中**匿名嵌入** `derive.Derive[fmt.Stringer]`（需 `import "fmt"`）。marker 自带桩 `String()`（经提升供宏主文件类型检查）；展开后移除嵌入桩，并在 Target 尚无自有 `String()` 时生成字段拼接版 `String()`。若已手写 `func (T) String() string`，或由其它嵌入类型提升得到 `String()`，则不再生成，保留用户实现。

**自 `derivestringer` 迁移：**

| 旧 | 新 |
|----|-----|
| `.../derivestringer` | `.../derive` |
| `DeriveStringer` | `Derive[fmt.Stringer]` |
| `derive-stringer` | `derive` |
| `DeriveStringerExpand` | `DeriveExpand` |

### wirejson — 补全 json tag

匿名嵌入 `wirejson.WireJSON`，为字段补全 `json` struct tag 并移除嵌入桩。可在嵌入字段上写可选属性，例如 `` `macro:"omitempty=FieldName"` ``。

---

## 维护者

### 包与 syntax-id

| 包 | syntax-id | import |
|----|-----------|--------|
| inline | `syntax-inline` | `github.com/arcane-craft/go-macro-contrib/inline` |
| try | `syntax-try` | `github.com/arcane-craft/go-macro-contrib/try` |
| with | `syntax-with` | `github.com/arcane-craft/go-macro-contrib/with` |
| derive | `derive` | `github.com/arcane-craft/go-macro-contrib/derive` |
| wirejson | `wire-json` | `github.com/arcane-craft/go-macro-contrib/wirejson` |

`derive` / `wirejson` 需要带 Decl 宏能力的 go-macro；本地联调时请 `replace` 到含该能力的 `../go-macro`。

### 与 go-macro 并列联调

建议目录布局：

```text
go-macro-work/
  go-macro/
  go-macro-contrib/
```

仅在**本地** `go-macro-contrib/go.mod` 临时添加（勿随发布 tag 提交）：

```go
replace github.com/arcane-craft/go-macro => ../go-macro
```

然后在本仓库根目录执行 `go test ./...`。

### 延伸阅读

- [go-macro](https://github.com/arcane-craft/go-macro) — 框架、示例与宏作者指南
