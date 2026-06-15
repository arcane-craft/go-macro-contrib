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
	out, err := mactest.ExpandSyntax(with.WithExpander, "With", "with", `
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
	stmts, err := out.ToStmts()
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 4 {
		t.Fatalf("want 4 stmts, got %d", len(stmts))
	}
	assign0, ok := stmts[0].(*ast.AssignStmt)
	if !ok || assign0.Tok != token.DEFINE || len(assign0.Lhs) != 2 {
		t.Fatalf("first stmt: %T", stmts[0])
	}
	if _, ok := stmts[1].(*ast.IfStmt); !ok {
		t.Fatal("second stmt not if")
	}
	if _, ok := stmts[2].(*ast.DeferStmt); !ok {
		t.Fatal("third stmt not defer")
	}
	assign1, ok := stmts[3].(*ast.AssignStmt)
	if !ok || mactest.IdentName(assign1.Lhs[0]) != "r" {
		t.Fatalf("success assign lhs: %#v", assign1.Lhs)
	}
	if mactest.IdentName(assign1.Rhs[0]) != mactest.IdentName(assign0.Lhs[0]) {
		t.Fatalf("success value %q != temp %q", mactest.IdentName(assign1.Rhs[0]), mactest.IdentName(assign0.Lhs[0]))
	}
}

func TestWithExpandReturn(t *testing.T) {
	out, err := mactest.ExpandSyntax(with.WithExpander, "With", "with", `
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
	stmts, err := out.ToStmts()
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) < 4 {
		t.Fatalf("want assign+if+defer+return, got %d", len(stmts))
	}
	if _, ok := stmts[2].(*ast.DeferStmt); !ok {
		t.Fatal("missing defer")
	}
	ret, ok := stmts[len(stmts)-1].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 {
		t.Fatalf("success return: %#v", stmts[len(stmts)-1])
	}
	if mactest.IdentName(ret.Results[1]) != "nil" {
		t.Fatalf("success err: %#v", ret.Results[1])
	}
}

func TestWithExpandNamedReturn(t *testing.T) {
	out, err := mactest.ExpandSyntax(with.WithExpander, "With", "with", `
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
	stmts, err := out.ToStmts()
	if err != nil {
		t.Fatal(err)
	}
	ifStmt := stmts[1].(*ast.IfStmt)
	ret := ifStmt.Body.List[0].(*ast.ReturnStmt)
	if lit, ok := ret.Results[0].(*ast.BasicLit); !ok || lit.Value != "0" {
		t.Fatalf("named zero: %#v", ret.Results[0])
	}
}

func TestWithExpandRejectNoErrorReturn(t *testing.T) {
	_, err := mactest.ExpandSyntax(with.WithExpander, "With", "with", `
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

func TestWithExpandRejectExprSite(t *testing.T) {
	_, err := mactest.ExpandSyntax(with.WithExpander, "With", "with", `
import "io"
func With[T io.Closer](v T, err error) T { panic("stub") }
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func use(_ any) {}
func f() (resource, error) {
	use(With(open()))
	return resource{}, nil
}
`)
	if err == nil || !strings.Contains(err.Error(), "no matching syntax rule") {
		t.Fatalf("got %v", err)
	}
}

func TestWithExpandRejectStmtSite(t *testing.T) {
	_, err := mactest.ExpandSyntax(with.WithExpander, "With", "with", `
import "io"
func With[T io.Closer](v T, err error) T { panic("stub") }
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func f() (resource, error) {
	With(open());
	return resource{}, nil
}
`)
	if err == nil || !strings.Contains(err.Error(), "no matching syntax rule") {
		t.Fatalf("got %v", err)
	}
}
