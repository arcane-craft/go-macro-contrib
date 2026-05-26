# go-macro-contrib

Official macro libraries for [go-macro](https://github.com/arcane-craft/go-macro): `inline`, `try`, and `register` (expandtool wiring).

## Module path

```
github.com/arcane-craft/go-macro-contrib
github.com/arcane-craft/go-macro-contrib/inline
github.com/arcane-craft/go-macro-contrib/try
github.com/arcane-craft/go-macro-contrib/register
```

## Minimum go-macro version

Requires `github.com/arcane-craft/go-macro` at the version pinned in `go.mod` (currently `v0.0.0` for local development; use a released tag in production).

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

## Enable expand (blank import)

```go
import _ "github.com/arcane-craft/go-macro-contrib/register"
```

## BREAKING migration (from `go-macro/contrib`)

| Old import | New import |
|------------|------------|
| `github.com/arcane-craft/go-macro/contrib/inline` | `github.com/arcane-craft/go-macro-contrib/inline` |
| `github.com/arcane-craft/go-macro/contrib/try` | `github.com/arcane-craft/go-macro-contrib/try` |
| `github.com/arcane-craft/go-macro/contrib/register` | `github.com/arcane-craft/go-macro-contrib/register` |

```bash
go get github.com/arcane-craft/go-macro-contrib@v0.1.0
```
