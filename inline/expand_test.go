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

func TestInlineExpandInlinesLocalCall(t *testing.T) {
	result, err := mactest.Expand(inline.InlineExpand, "Inline", "syntax-inline", `
func Inline[T any](v T) T { panic("stub") }
func add(a, b int) int { return a + b }
func f() int { return Inline(add(1, 2)) }
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stmts) != 1 {
		t.Fatalf("want 1 return stmt, got expr=%#v stmts=%d", result.Expr, len(result.Stmts))
	}
	ret, ok := result.Stmts[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		t.Fatalf("want return, got %#v", result.Stmts[0])
	}
	be, ok := ret.Results[0].(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("want binary expr, got %#v", ret.Results[0])
	}
	litX, ok := be.X.(*ast.BasicLit)
	if !ok || litX.Value != "1" {
		t.Fatalf("want 1 on left, got %#v", be.X)
	}
	litY, ok := be.Y.(*ast.BasicLit)
	if !ok || litY.Value != "2" {
		t.Fatalf("want 2 on right, got %#v", be.Y)
	}
}

func TestInlineExpandRejectsNonInlineableCallee(t *testing.T) {
	_, err := mactest.Expand(inline.InlineExpand, "Inline", "syntax-inline", `
func Inline[T any](v T) T { panic("stub") }
func g() int { x := 3; return x }
func f() int { return Inline(g()) }
`)
	if err == nil || !strings.Contains(err.Error(), "cannot inline") {
		t.Fatalf("got %v", err)
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
	if err == nil || !strings.Contains(err.Error(), "return position") {
		t.Fatalf("got %v", err)
	}
}

func TestInline2ExpandAssignSite(t *testing.T) {
	result, err := mactest.Expand(inline.InlineExpand, "Inline2", "syntax-inline", `
func Inline2[A, B any](a A, b B) (A, B) { panic("stub") }
func split() (string, string) { return "a", "b" }
func f() (string, string) {
	a, b := Inline2(split())
	return a, b
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stmts) != 1 {
		t.Fatalf("want 1 stmt, got %d", len(result.Stmts))
	}
	assign, ok := result.Stmts[0].(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 2 || len(assign.Rhs) != 2 {
		t.Fatalf("assign: %#v", result.Stmts[0])
	}
	lit0, ok := assign.Rhs[0].(*ast.BasicLit)
	if !ok || lit0.Value != `"a"` {
		t.Fatalf("rhs[0]: %#v", assign.Rhs[0])
	}
	lit1, ok := assign.Rhs[1].(*ast.BasicLit)
	if !ok || lit1.Value != `"b"` {
		t.Fatalf("rhs[1]: %#v", assign.Rhs[1])
	}
}

func TestInline0ExpandStmtSite(t *testing.T) {
	result, err := mactest.Expand(inline.InlineExpand, "Inline0", "syntax-inline", `
func Inline0(f func()) { panic("stub") }
func cleanup() { _ = 1 }
func f() {
	Inline0(func() { cleanup() })
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stmts) != 1 {
		t.Fatalf("want 1 stmt, got %d", len(result.Stmts))
	}
	assign, ok := result.Stmts[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("want assign stmt, got %T", result.Stmts[0])
	}
	lit, ok := assign.Rhs[0].(*ast.BasicLit)
	if !ok || lit.Value != "1" {
		t.Fatalf("inlined body rhs: %#v", assign.Rhs[0])
	}
}

