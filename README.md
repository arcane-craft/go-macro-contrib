# go-macro-contrib

Official macro libraries for [go-macro](https://github.com/arcane-craft/go-macro): `inline` and `try`.

## Module path

```
github.com/arcane-craft/go-macro-contrib
github.com/arcane-craft/go-macro-contrib/inline
github.com/arcane-craft/go-macro-contrib/try
```

Each stub and Expander function is annotated with `//macro: <syntax-id>`. Consumers run:

```bash
go run github.com/arcane-craft/go-macro/cmd/macro@latest expand .
```

No `register` package is required.

## Minimum go-macro version

Requires `github.com/arcane-craft/go-macro` at the version pinned in `go.mod`.

## Local development (sibling checkout)

Clone this repo next to `go-macro`:

```
go-macro-work/
  go-macro/
  go-macro-contrib/
```

In `go-macro-contrib/go.mod`:

```go
replace github.com/arcane-craft/go-macro => ../go-macro
```

Run tests:

```bash
go test ./...
```

## BREAKING migration (from `go-macro/contrib`)

| Old import | New import |
|------------|------------|
| `github.com/arcane-craft/go-macro/contrib/inline` | `github.com/arcane-craft/go-macro-contrib/inline` |
| `github.com/arcane-craft/go-macro/contrib/try` | `github.com/arcane-craft/go-macro-contrib/try` |

Remove blank import of `contrib/register` or `go-macro-contrib/register`; use `cmd/macro expand` instead.

```bash
go get github.com/arcane-craft/go-macro-contrib@v0.1.0
```
