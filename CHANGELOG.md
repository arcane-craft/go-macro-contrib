# Changelog

## v0.1.0

Initial release as independent module `github.com/arcane-craft/go-macro-contrib` (migrated from `github.com/arcane-craft/go-macro/contrib`).

### BREAKING migration

| Old import | New import |
|------------|------------|
| `github.com/arcane-craft/go-macro/contrib/inline` | `github.com/arcane-craft/go-macro-contrib/inline` |
| `github.com/arcane-craft/go-macro/contrib/try` | `github.com/arcane-craft/go-macro-contrib/try` |
| `github.com/arcane-craft/go-macro/contrib/register` | `github.com/arcane-craft/go-macro-contrib/register` |

```bash
go get github.com/arcane-craft/go-macro-contrib@v0.1.0
```

Requires compatible `github.com/arcane-craft/go-macro` (see `go.mod`).
