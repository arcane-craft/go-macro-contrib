package inline_test

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/arcane-craft/go-macro-contrib/inline"
	"github.com/arcane-craft/go-macro/macro/mactest"
)

func TestInlineExpandExprUnwrapsArgument(t *testing.T) {
	result, err := mactest.Expand(inline.InlineExpand, "Inline", "syntax-inline", `
func Inline[T any](v T) T { panic("stub") }
func f() int {
	return 1 + Inline(2)
}
`)
	if err != nil {
		t.Fatal(err)
	}
	lit, ok := result.Expr.(*ast.BasicLit)
	if !ok || lit.Value != "2" {
		t.Fatalf("want literal 2, got %#v", result.Expr)
	}
}

func TestInlineExpandPreservesComplexArg(t *testing.T) {
	result, err := mactest.Expand(inline.InlineExpand, "Inline", "syntax-inline", `
func Inline[T any](v T) T { panic("stub") }
func g() int { return 3 }
func f() int { return 1 + Inline(g()) }
`)
	if err != nil {
		t.Fatal(err)
	}
	call, ok := result.Expr.(*ast.CallExpr)
	if !ok || mactest.IdentName(call.Fun) != "g" {
		t.Fatalf("want g(), got %#v", result.Expr)
	}
}

func TestInlineExpandRejectAssign(t *testing.T) {
	_, err := mactest.Expand(inline.InlineExpand, "Inline", "syntax-inline", `
func Inline[T any](v T) T { panic("stub") }
func f() int {
	x := Inline(1)
	return x
}
`)
	if err == nil || !strings.Contains(err.Error(), "expression position") {
		t.Fatalf("got %v", err)
	}
}

