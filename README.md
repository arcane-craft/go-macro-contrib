# go-macro-contrib

[go-macro](https://github.com/arcane-craft/go-macro) 的官方宏库，提供 `inline`、`try` 两个语法包。

## 阅读指引

| 你想做什么 | 从这里开始 |
|------------|------------|
| 在项目里**用** `inline` / `try` | [快速上手](#快速上手) |
| 了解每个宏做什么 | [宏一览](#宏一览) |
| 本地改本仓库、联调核心 | [维护者参考](#维护者参考) |

写宏库、框架接入细节见 [go-macro 宏作者指南](https://github.com/arcane-craft/go-macro/blob/master/docs/author-guide.md)。

## 快速上手

下面用 `inline` 走通一次「写宏 → 展开 → 日常构建」。`try` 只需把 import 与调用换成 `try` 包即可。

### 1. 安装本仓库

在你的**业务 module** 根目录执行：

```bash
go get github.com/arcane-craft/go-macro-contrib@latest
```

展开由 [go-macro](https://github.com/arcane-craft/go-macro) 的 CLI 完成；`go run .../cmd/macro@latest` 会在本次命令里拉取 CLI，无需单独安装二进制。

### 2. 准备宏主文件

在要写宏的包里新建（或修改）源文件，例如 `demo.go`：

```go
//go:build macro

//go:generate go run github.com/arcane-craft/go-macro/cmd/macro@latest expand .

package demo

import "github.com/arcane-craft/go-macro-contrib/inline"

func Answer() int {
	return inline.Inline(42)
}
```

说明：

- `//go:build macro` 让这份源码只在「宏编辑」语境下参与类型检查；日常 `go build` 走展开后生成的 `demo_macro_gen.go`，一般**不用**给日常构建加 `-tags macro`。
- `//go:generate` 告诉 Go：你在该包目录执行 `go generate` 时，应调用上面的 `expand` 命令。
- 你需要 **import** 用到的宏包；展开器只会处理已 import 的宏库。

### 3. 运行展开

在 `demo.go` **所在包目录**执行：

```bash
go generate .
```

CLI 会扫描该包的 import、生成/更新 `.gomacro/expand_runner`（建议加入 `.gitignore`），并写出 `demo_macro_gen.go`。

也可以直接调用 expand（整模块可用 `./...`）：

```bash
go run github.com/arcane-craft/go-macro/cmd/macro@latest expand .
```

### 4. 日常构建

若仓库会被他人 `go get`，建议把宏主文件与 `*_macro_gen.go` 一并提交。之后：

```bash
go build ./...
go test ./...
```

读到此处，你应已完成第一次成功展开；下文是常用命令与参考信息。

## 常用命令

| 命令 | 何时用 |
|------|--------|
| `go generate .` | 在宏主文件所在包触发展开（依赖文件里的 `//go:generate`） |
| `go run github.com/arcane-craft/go-macro/cmd/macro@latest expand .` | 手动展开当前 module（可换为 `./...` 等 pattern） |
| `go test ./...` | 在本仓库根目录验证 `inline` / `try` 实现（维护者） |

## 编辑器（gopls）

宏主文件带 `//go:build macro` 时，若 IDE 同时分析宏主文件和 `*_macro_gen.go`，可能出现重复定义提示。你可以在 gopls 里加上 `-tags=macro`，让编辑器按宏版本源码做补全与类型检查：

```json
"gopls": {
  "buildFlags": ["-tags=macro"]
}
```

更多背景见 [go-macro README · gopls](https://github.com/arcane-craft/go-macro#gopls)。

## 宏一览

两个包都是**语法桩**：你在宏主文件里调用它们，展开后得到普通 Go；运行时不要直接调用桩函数（会 panic）。

### inline

`inline.Inline(expr)` 在**表达式位置**展开为 `expr` 本身，相当于去掉一层包装。

```go
x := inline.Inline(1 + g()) // 展开后等价于 x := 1 + g()
```

只用于表达式语境（例如 `return` 右侧、赋值右侧）。写在语句等其它位置时，展开器会报错。

### try

`try` 族把 `(T, error)`、`(A, B, error)` 等签名展开成带 `if err != nil` 的控制流。按 callee 在 **error 之前有几个返回值** 选桩：

| 桩 | 典型 callee |
|----|-------------|
| `Try0` | `(error)` |
| `Try` | `(T, error)` |
| `Try2` | `(A, B, error)` |
| `Try3` | `(A, B, C, error)` |

例如 `try.Try(os.Open(...))` 对应 `( *os.File, error)`；若是 `(A, B, error)`，应写 `try.Try2(...)`。

## 维护者参考

### 模块路径

| 包 | import |
|----|--------|
| inline | `github.com/arcane-craft/go-macro-contrib/inline` |
| try | `github.com/arcane-craft/go-macro-contrib/try` |

### 与 go-macro 并列联调

同时开发核心与本仓库时，建议目录布局：

```text
go-macro-work/
  go-macro/
  go-macro-contrib/
```

仅在**本地** `go-macro-contrib/go.mod` 临时添加（勿作为发布/tag 的提交内容）：

```go
replace github.com/arcane-craft/go-macro => ../go-macro
```

然后在本仓库根目录执行 `go test ./...`。

### 进一步阅读

- [go-macro](https://github.com/arcane-craft/go-macro) — 框架、示例与宏作者指南
