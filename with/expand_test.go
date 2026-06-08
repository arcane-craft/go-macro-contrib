package with_test

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"

	"github.com/arcane-craft/go-macro/macro/mactest"
	"github.com/arcane-craft/go-macro-contrib/with"
)

func TestWithExpandAssign(t *testing.T) {
	result, err := mactest.ExpandCall(with.WithExpand, "With", "syntax-with", `
import "io"
func With[T io.Closer](v T, err error) T { panic("stub") }
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func f() ([]byte, error) {
	r := With(open())
	_ = r
	return nil, nil
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stmts) != 4 {
		t.Fatalf("want 4 stmts, got %d", len(result.Stmts))
	}
	assign0, ok := result.Stmts[0].(*ast.AssignStmt)
	if !ok || assign0.Tok != token.DEFINE || len(assign0.Lhs) != 2 {
		t.Fatalf("first stmt: %T", result.Stmts[0])
	}
	if _, ok := result.Stmts[1].(*ast.IfStmt); !ok {
		t.Fatal("second stmt not if")
	}
	if _, ok := result.Stmts[2].(*ast.DeferStmt); !ok {
		t.Fatal("third stmt not defer")
	}
	assign1, ok := result.Stmts[3].(*ast.AssignStmt)
	if !ok || mactest.IdentName(assign1.Lhs[0]) != "r" {
		t.Fatalf("success assign lhs: %#v", assign1.Lhs)
	}
	if mactest.IdentName(assign1.Rhs[0]) != mactest.IdentName(assign0.Lhs[0]) {
		t.Fatalf("success value %q != temp %q", mactest.IdentName(assign1.Rhs[0]), mactest.IdentName(assign0.Lhs[0]))
	}
}

func TestWithExpandReturn(t *testing.T) {
	result, err := mactest.ExpandCall(with.WithExpand, "With", "syntax-with", `
import "io"
func With[T io.Closer](v T, err error) (T, error) { panic("stub") }
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func f() (resource, error) {
	return With(open())
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stmts) < 4 {
		t.Fatalf("want assign+if+defer+return, got %d", len(result.Stmts))
	}
	if _, ok := result.Stmts[2].(*ast.DeferStmt); !ok {
		t.Fatal("missing defer")
	}
	ret, ok := result.Stmts[len(result.Stmts)-1].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 {
		t.Fatalf("success return: %#v", result.Stmts[len(result.Stmts)-1])
	}
	if mactest.IdentName(ret.Results[1]) != "nil" {
		t.Fatalf("success err: %#v", ret.Results[1])
	}
}

func TestWithExpandNamedReturn(t *testing.T) {
	result, err := mactest.ExpandCall(with.WithExpand, "With", "syntax-with", `
import "io"
func With[T io.Closer](v T, err error) T { panic("stub") }
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func f() (r resource, err error) {
	r = With(open())
	return
}
`)
	if err != nil {
		t.Fatal(err)
	}
	ifStmt := result.Stmts[1].(*ast.IfStmt)
	ret := ifStmt.Body.List[0].(*ast.ReturnStmt)
	if mactest.IdentName(ret.Results[0]) != "nil" {
		t.Fatalf("named zero: %#v", ret.Results[0])
	}
}

func TestWithExpandRejectNoErrorReturn(t *testing.T) {
	_, err := mactest.ExpandCall(with.WithExpand, "With", "syntax-with", `
import "io"
func With[T io.Closer](v T, err error) T { panic("stub") }
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func f() int {
	_ = With(open())
	return 0
}
`)
	if err == nil || !strings.Contains(err.Error(), "error return") {
		t.Fatalf("got %v", err)
	}
}

