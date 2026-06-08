package with

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/arcane-craft/go-macro/macro"
)

func TestWithExpandRequiresOneArg(t *testing.T) {
	const src = `package p
import "io"
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func f() (resource, error) { return resource{}, nil }
func With[T io.Closer](v T, err error) T { panic("stub") }
`
	fset, f, fn, info, pkg := parseWithSnippet(t, src)
	call := &ast.CallExpr{Fun: ast.NewIdent("With"), Lparen: token.NoPos}
	ctx := &fakeCallContext{fset: fset, file: f, info: info, pkg: pkg, call: call, stub: "With", site: macro.SiteAssign, enclosing: fn, pos: fset.File(1).Pos(1)}
	_, err := WithExpand(ctx, call)
	if err == nil || !strings.Contains(err.Error(), "one argument") {
		t.Fatalf("got %v", err)
	}
}

func TestWithExpandRejectNonCloser(t *testing.T) {
	const src = `package p
import "io"
func helper() (int, error) { return 0, nil }
func f() (int, error) { return 0, nil }
func With[T io.Closer](v T, err error) T { panic("stub") }
`
	fset, f, fn, info, pkg := parseWithSnippet(t, src)
	helperCall := &ast.CallExpr{Fun: ast.NewIdent("helper")}
	withCall := &ast.CallExpr{Fun: ast.NewIdent("With"), Args: []ast.Expr{helperCall}}
	helperResults := types.NewTuple(
		types.NewParam(0, nil, "", types.Typ[types.Int]),
		types.NewParam(0, nil, "", types.Universe.Lookup("error").Type()),
	)
	info.Types[helperCall] = types.TypeAndValue{Type: helperResults}
	ctx := &fakeCallContext{
		fset: fset, file: f, info: info, pkg: pkg, call: withCall, stub: "With", site: macro.SiteAssign,
		enclosing: fn, pos: withCall.Pos(),
	}
	_, err := WithExpand(ctx, withCall)
	if err == nil || !strings.Contains(err.Error(), "io.Closer") {
		t.Fatalf("got %v", err)
	}
}

func TestWithExpandRejectExprSite(t *testing.T) {
	const src = `package p
import "io"
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func f() (resource, error) { return resource{}, nil }
func With[T io.Closer](v T, err error) T { panic("stub") }
`
	fset, f, fn, info, pkg := parseWithSnippet(t, src)
	openCall := &ast.CallExpr{Fun: ast.NewIdent("open")}
	withCall := &ast.CallExpr{Fun: ast.NewIdent("With"), Args: []ast.Expr{openCall}}
	resType := pkg.Scope().Lookup("resource").Type()
	info.Types[openCall] = types.TypeAndValue{Type: types.NewTuple(
		types.NewParam(0, nil, "", resType),
		types.NewParam(0, nil, "", types.Universe.Lookup("error").Type()),
	)}
	ctx := &fakeCallContext{
		fset: fset, file: f, info: info, pkg: pkg, call: withCall, stub: "With", site: macro.SiteExpr,
		enclosing: fn, pos: withCall.Pos(),
	}
	_, err := WithExpand(ctx, withCall)
	if err == nil || !strings.Contains(err.Error(), "expression position") {
		t.Fatalf("got %v", err)
	}
}

func TestWithExpandRejectMultiPayload(t *testing.T) {
	const src = `package p
import "io"
type resource struct{}
func (resource) Close() error { return nil }
func pair() (resource, int, error) { return resource{}, 0, nil }
func f() (resource, error) { return resource{}, nil }
func With[T io.Closer](v T, err error) T { panic("stub") }
`
	fset, f, fn, info, pkg := parseWithSnippet(t, src)
	pairCall := &ast.CallExpr{Fun: ast.NewIdent("pair")}
	withCall := &ast.CallExpr{Fun: ast.NewIdent("With"), Args: []ast.Expr{pairCall}}
	resType := pkg.Scope().Lookup("resource").Type()
	info.Types[pairCall] = types.TypeAndValue{Type: types.NewTuple(
		types.NewParam(0, nil, "", resType),
		types.NewParam(0, nil, "", types.Typ[types.Int]),
		types.NewParam(0, nil, "", types.Universe.Lookup("error").Type()),
	)}
	ctx := &fakeCallContext{
		fset: fset, file: f, info: info, pkg: pkg, call: withCall, stub: "With", site: macro.SiteAssign,
		enclosing: fn, pos: withCall.Pos(),
	}
	_, err := WithExpand(ctx, withCall)
	if err == nil || !strings.Contains(err.Error(), "(T, error)") {
		t.Fatalf("got %v", err)
	}
}

func TestWithExpandRejectStmtSite(t *testing.T) {
	const src = `package p
import "io"
type resource struct{}
func (resource) Close() error { return nil }
func open() (resource, error) { return resource{}, nil }
func f() (resource, error) { return resource{}, nil }
func With[T io.Closer](v T, err error) T { panic("stub") }
`
	fset, f, fn, info, pkg := parseWithSnippet(t, src)
	openCall := &ast.CallExpr{Fun: ast.NewIdent("open")}
	withCall := &ast.CallExpr{Fun: ast.NewIdent("With"), Args: []ast.Expr{openCall}}
	resType := pkg.Scope().Lookup("resource").Type()
	info.Types[openCall] = types.TypeAndValue{Type: types.NewTuple(
		types.NewParam(0, nil, "", resType),
		types.NewParam(0, nil, "", types.Universe.Lookup("error").Type()),
	)}
	ctx := &fakeCallContext{
		fset: fset, file: f, info: info, pkg: pkg, call: withCall, stub: "With", site: macro.SiteStmt,
		enclosing: fn, pos: withCall.Pos(),
	}
	_, err := WithExpand(ctx, withCall)
	if err == nil || !strings.Contains(err.Error(), "statement") {
		t.Fatalf("got %v", err)
	}
}

func parseWithSnippet(t *testing.T, src string) (*token.FileSet, *ast.File, *ast.FuncDecl, *types.Info, *types.Package) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var fn *ast.FuncDecl
	ast.Inspect(f, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name.Name == "f" {
			fn = fd
		}
		return true
	})
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	cfg := &types.Config{Importer: importer.Default()}
	pkg, err := cfg.Check("p", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(err)
	}
	return fset, f, fn, info, pkg
}
