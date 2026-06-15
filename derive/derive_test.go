package derive_test

import (
	"fmt"
	"go/ast"
	"strings"
	"testing"

	"github.com/arcane-craft/go-macro/macro/mactest"
	"github.com/arcane-craft/go-macro-contrib/derive"
)

const deriveMarkerStub = `
import "fmt"

type Derive[T any] struct{}

func (Derive[T]) String() string { panic("stub") }
`

func TestDeriveStubPromotesStringer(t *testing.T) {
	var _ fmt.Stringer = struct {
		derive.Derive[fmt.Stringer]
		A string
	}{}
}

func TestDeriveExpand(t *testing.T) {
	out, err := mactest.Expand(derive.DeriveExpander, "derive", deriveMarkerStub+`
type Item struct {
	Derive[fmt.Stringer]
	A string
	B int
}
`)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := out.ToDecls()
	if err != nil {
		t.Fatal(err)
	}
	ts := typeSpecFromDecls(t, decls)
	if len(ts.Type.(*ast.StructType).Fields.List) != 2 {
		t.Fatalf("fields: %d", len(ts.Type.(*ast.StructType).Fields.List))
	}
	if !hasStringMethodDecl(decls) {
		t.Fatal("expected String method")
	}
}

func TestDeriveSkipsWhenTargetDeclaresString(t *testing.T) {
	out, err := mactest.Expand(derive.DeriveExpander, "derive", deriveMarkerStub+`
func (Item) String() string { return "custom" }

type Item struct {
	Derive[fmt.Stringer]
	A string
}
`)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := out.ToDecls()
	if err != nil {
		t.Fatal(err)
	}
	if hasStringMethodDecl(decls) {
		t.Fatal("expected no generated String method")
	}
}

func TestDeriveSkipsWhenOtherEmbedPromotesString(t *testing.T) {
	out, err := mactest.Expand(derive.DeriveExpander, "derive", deriveMarkerStub+`
type Helper struct{}

func (Helper) String() string { return "from-helper" }

type Item struct {
	Derive[fmt.Stringer]
	Helper
}
`)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := out.ToDecls()
	if err != nil {
		t.Fatal(err)
	}
	if hasStringMethodDecl(decls) {
		t.Fatal("expected no generated String method when Helper promotes String")
	}
}

func TestDeriveRejectsInvalidTypeArg(t *testing.T) {
	cases := []struct {
		name    string
		snippet string
	}{
		{
			name: "int",
			snippet: `
type Derive[T any] struct{}

func (Derive[T]) String() string { panic("stub") }

type Item struct {
	Derive[int]
	A string
}
`,
		},
		{
			name: "any",
			snippet: `
type Derive[T any] struct{}

func (Derive[T]) String() string { panic("stub") }

type Item struct {
	Derive[any]
	A string
}
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mactest.Expand(derive.DeriveExpander, "derive", tc.snippet)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "fmt.Stringer") {
				t.Fatalf("error %q should mention fmt.Stringer", err)
			}
		})
	}
}

func typeSpecFromDecls(t *testing.T, decls []ast.Decl) *ast.TypeSpec {
	t.Helper()
	for _, d := range decls {
		if gd, ok := d.(*ast.GenDecl); ok {
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					return ts
				}
			}
		}
	}
	t.Fatal("no TypeSpec in decls")
	return nil
}

func hasStringMethodDecl(decls []ast.Decl) bool {
	for _, d := range decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "String" {
			return true
		}
	}
	return false
}
