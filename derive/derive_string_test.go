package derive

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/arcane-craft/go-macro/macro"
)

func TestShouldGenerateStringPaths(t *testing.T) {
	const base = `package p
import "fmt"
type Derive[T any] struct{}
func (Derive[T]) String() string { panic("stub") }
`
	cases := []struct {
		name   string
		suffix string
		want   bool
	}{
		{
			name: "generate",
			suffix: `
type Item struct {
	Derive[fmt.Stringer]
	A string
}`,
			want: true,
		},
		{
			name: "user string",
			suffix: `
func (Item) String() string { return "custom" }
type Item struct {
	Derive[fmt.Stringer]
	A string
}`,
			want: false,
		},
		{
			name: "helper embed",
			suffix: `
type Helper struct{}
func (Helper) String() string { return "h" }
type Item struct {
	Derive[fmt.Stringer]
	Helper
}`,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "p.go", base+tc.suffix, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Defs:  make(map[*ast.Ident]types.Object),
				Uses:  make(map[*ast.Ident]types.Object),
			}
			cfg := &types.Config{Importer: importer.Default()}
			if _, err := cfg.Check("p", fset, []*ast.File{f}, info); err != nil {
				t.Fatal(err)
			}
			var embed *ast.Field
			var itemTS *ast.TypeSpec
			ast.Inspect(f, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == "Item" {
					itemTS = ts
				}
				if fl, ok := n.(*ast.Field); ok && fl.Names == nil {
					if ie, ok := fl.Type.(*ast.IndexExpr); ok {
						if id, ok := ie.X.(*ast.Ident); ok && id.Name == "Derive" {
							embed = fl
						}
					}
				}
				return true
			})
			if embed == nil || itemTS == nil {
				t.Fatal("missing embed or Item")
			}
			binds := macroQuoteBindings(itemTS, embed, itemTS.Type.(*ast.StructType).Fields.List)
			site := &deriveTestSite{file: f, anchor: embed}
			got := shouldGenerateString(ctxStub{info: info}, site, binds)
			if got != tc.want {
				t.Fatalf("shouldGenerateString=%v want %v", got, tc.want)
			}
		})
	}
}

type ctxStub struct {
	macro.Context
	info *types.Info
}

func (c ctxStub) Types() *types.Info { return c.info }

type deriveTestSite struct {
	file   *ast.File
	anchor *ast.Field
}

func (s *deriveTestSite) Match(string) (macro.Bindings, error) { return nil, nil }
func (s *deriveTestSite) ToExpr() (ast.Expr, error)            { return nil, nil }
func (s *deriveTestSite) ToExprs() ([]ast.Expr, error)         { return nil, nil }
func (s *deriveTestSite) ToStmt() (ast.Stmt, error)            { return nil, nil }
func (s *deriveTestSite) ToStmts() ([]ast.Stmt, error)         { return nil, nil }
func (s *deriveTestSite) ToDecl() (ast.Decl, error)            { return nil, nil }
func (s *deriveTestSite) ToDecls() ([]ast.Decl, error)         { return nil, nil }
func (s *deriveTestSite) Underlying() ast.Node                 { return s.anchor }
func (s *deriveTestSite) MacroPos() token.Pos                  { return s.anchor.Pos() }
func (s *deriveTestSite) ExpansionFile() *ast.File             { return s.file }
func (s *deriveTestSite) ClearExpansionMeta()                  {}

func macroQuoteBindings(itemTS *ast.TypeSpec, embed *ast.Field, allFields []*ast.Field) macro.Bindings {
	b := newMapBindings()
	b.singles["item"] = macro.WrapNode(itemTS)
	var fieldSyns []macro.Syntax
	for _, f := range allFields {
		if f == embed {
			continue
		}
		if len(f.Names) == 0 {
			continue
		}
		fieldSyns = append(fieldSyns, macro.WrapNode(f))
	}
	b.lists["field"] = fieldSyns
	return b
}

type mapBindings struct {
	singles map[string]macro.Syntax
	lists   map[string][]macro.Syntax
}

func newMapBindings() *mapBindings {
	return &mapBindings{
		singles: make(map[string]macro.Syntax),
		lists:   make(map[string][]macro.Syntax),
	}
}

func (b *mapBindings) Get(name string) (macro.Syntax, bool) {
	v, ok := b.singles[name]
	return v, ok
}

func (b *mapBindings) Elems(name string) ([]macro.Syntax, bool) {
	v, ok := b.lists[name]
	return v, ok
}
