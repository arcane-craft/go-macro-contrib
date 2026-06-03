package try_test

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"

	"github.com/arcane-craft/go-macro/macro/mactest"
	"github.com/arcane-craft/go-macro-contrib/try"
)

func TestTryExpandAssignInlinesCallee(t *testing.T) {
	result, err := mactest.ExpandCall(try.TryExpand, "Try", "syntax-try", `
func Try[T any](v T, err error) T { panic("stub") }
func helper() (int, error) { return 0, nil }
func f() (int, error) {
	x := Try(helper())
	return x, nil
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stmts) != 3 {
		t.Fatalf("want 3 stmts, got %d", len(result.Stmts))
	}
	assign0, ok := result.Stmts[0].(*ast.AssignStmt)
	if !ok || assign0.Tok != token.DEFINE || len(assign0.Lhs) != 2 || len(assign0.Rhs) != 1 {
		t.Fatalf("first stmt: %T", result.Stmts[0])
	}
	call, ok := assign0.Rhs[0].(*ast.CallExpr)
	if !ok || mactest.IdentName(call.Fun) != "helper" {
		t.Fatalf("rhs callee: %s", mactest.FormatExpr(nil, assign0.Rhs[0]))
	}
	ifStmt, ok := result.Stmts[1].(*ast.IfStmt)
	if !ok {
		t.Fatal("second stmt not if")
	}
	cond, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.NEQ || mactest.IdentName(cond.Y) != "nil" {
		t.Fatalf("if cond: %#v", ifStmt.Cond)
	}
	ret := ifStmt.Body.List[0].(*ast.ReturnStmt)
	if len(ret.Results) != 2 {
		t.Fatalf("error return arity: %d", len(ret.Results))
	}
	if lit, ok := ret.Results[0].(*ast.BasicLit); !ok || lit.Value != "0" {
		t.Fatalf("zero value: %#v", ret.Results[0])
	}
	if mactest.IdentName(ret.Results[1]) != mactest.IdentName(cond.X) {
		t.Fatalf("returned err %q != cond %q", mactest.IdentName(ret.Results[1]), mactest.IdentName(cond.X))
	}
	assign1, ok := result.Stmts[2].(*ast.AssignStmt)
	if !ok || len(assign1.Lhs) != 1 || mactest.IdentName(assign1.Lhs[0]) != "x" {
		t.Fatalf("success assign lhs: %#v", assign1.Lhs)
	}
	if mactest.IdentName(assign1.Rhs[0]) != mactest.IdentName(assign0.Lhs[0]) {
		t.Fatalf("success value %q != temp %q", mactest.IdentName(assign1.Rhs[0]), mactest.IdentName(assign0.Lhs[0]))
	}
}

func TestTryExpandReturnReplacesWithErrorHandling(t *testing.T) {
	result, err := mactest.ExpandCall(try.TryExpand, "Try", "syntax-try", `
func Try[T any](v T, err error) (T, error) { panic("stub") }
func helper() (int, error) { return 1, nil }
func f() (int, error) {
	return Try(helper())
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stmts) < 3 {
		t.Fatalf("want assign+if+return, got %d stmts", len(result.Stmts))
	}
	ret, ok := result.Stmts[len(result.Stmts)-1].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 {
		t.Fatalf("success return: %#v", result.Stmts[len(result.Stmts)-1])
	}
	if mactest.IdentName(ret.Results[0]) == "" {
		t.Fatalf("payload return should be temp ident, got %#v", ret.Results[0])
	}
	if mactest.IdentName(ret.Results[1]) != "nil" {
		t.Fatalf("success err: %#v", ret.Results[1])
	}
}

func TestTry0ExpandStmtSite(t *testing.T) {
	result, err := mactest.ExpandCall(try.TryExpand, "Try0", "syntax-try", `
func Try0(err error) { panic("stub") }
func closer() error { return nil }
func f() error {
	Try0(closer())
	return nil
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stmts) != 2 {
		t.Fatalf("Try0: want assign+if, got %d", len(result.Stmts))
	}
	assign := result.Stmts[0].(*ast.AssignStmt)
	if len(assign.Lhs) != 1 || mactest.IdentName(assign.Rhs[0].(*ast.CallExpr).Fun) != "closer" {
		t.Fatalf("Try0 assign: lhs=%d rhs=%s", len(assign.Lhs), mactest.FormatExpr(nil, assign.Rhs[0]))
	}
	if _, ok := result.Stmts[1].(*ast.IfStmt); !ok {
		t.Fatal("Try0 missing if err")
	}
}

func TestTry2ExpandAssign(t *testing.T) {
	result, err := mactest.ExpandCall(try.TryExpand, "Try2", "syntax-try", `
func Try2[A, B any](a A, b B, err error) (A, B) { panic("stub") }
func pair() (int, string, error) { return 0, "", nil }
func f() (int, string, error) {
	a, b := Try2(pair())
	return a, b, nil
}
`)
	if err != nil {
		t.Fatal(err)
	}
	assign0 := result.Stmts[0].(*ast.AssignStmt)
	if len(assign0.Lhs) != 3 {
		t.Fatalf("Try2 assign lhs count: %d", len(assign0.Lhs))
	}
	call, ok := assign0.Rhs[0].(*ast.CallExpr)
	if !ok || mactest.IdentName(call.Fun) != "pair" {
		t.Fatal("Try2 rhs not pair()")
	}
	assign1 := result.Stmts[2].(*ast.AssignStmt)
	if len(assign1.Lhs) != 2 || len(assign1.Rhs) != 2 {
		t.Fatalf("Try2 success assign: lhs=%d rhs=%d", len(assign1.Lhs), len(assign1.Rhs))
	}
}

func TestTryExpandNamedReturnUsesStringZero(t *testing.T) {
	result, err := mactest.ExpandCall(try.TryExpand, "Try", "syntax-try", `
func Try[T any](v T, err error) T { panic("stub") }
func helper() (string, error) { return "", nil }
func f() (msg string, err error) {
	msg = Try(helper())
	return
}
`)
	if err != nil {
		t.Fatal(err)
	}
	ifStmt := result.Stmts[1].(*ast.IfStmt)
	ret := ifStmt.Body.List[0].(*ast.ReturnStmt)
	if lit, ok := ret.Results[0].(*ast.BasicLit); !ok || lit.Value != `""` {
		t.Fatalf("named string zero: %#v", ret.Results[0])
	}
}

func TestTryExpandRejectNoErrorReturn(t *testing.T) {
	_, err := mactest.ExpandCall(try.TryExpand, "Try", "syntax-try", `
func Try[T any](v T, err error) T { panic("stub") }
func helper() (int, error) { return 0, nil }
func f() int {
	_ = Try(helper())
	return 0
}
`)
	if err == nil || !strings.Contains(err.Error(), "error return") {
		t.Fatalf("got %v", err)
	}
}

func TestTryExpandRejectExprSite(t *testing.T) {
	_, err := mactest.ExpandCall(try.TryExpand, "Try", "syntax-try", `
func Try[T any](v T, err error) T { panic("stub") }
func helper() (int, error) { return 0, nil }
func f() (int, error) {
	_ = 1 + Try(helper())
	return 0, nil
}
`)
	if err == nil || !strings.Contains(err.Error(), "expression position") {
		t.Fatalf("got %v", err)
	}
}

