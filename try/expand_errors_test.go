package try

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

// Error-path tests use a minimal typed snippet; assertions check message semantics.

func TestTryExpandRequiresOneArg(t *testing.T) {
	const src = `package p
func helper() (int, error) { return 0, nil }
func f() (int, error) { return 0, nil }
func Try[T any](v T, err error) T { panic("stub") }
`
	fset, _, fn, info, pkg := parseTrySnippet(t, src)
	call := &ast.CallExpr{Fun: ast.NewIdent("Try"), Lparen: token.NoPos}
	ctx := &fakeContext{fset: fset, info: info, pkg: pkg, call: call, stub: "Try", site: macro.SiteAssign, enclosing: fn, pos: fset.File(1).Pos(1)}
	_, err := TryExpand(ctx, call)
	if err == nil || !strings.Contains(err.Error(), "one argument") {
		t.Fatalf("got %v", err)
	}
}

func TestTryExpandWrongStubForPair(t *testing.T) {
	const src = `package p
func pair() (int, string, error) { return 0, "", nil }
func f() (int, string, error) { return 0, "", nil }
func Try[T any](v T, err error) T { panic("stub") }
`
	fset, _, fn, info, pkg := parseTrySnippet(t, src)
	pairCall := &ast.CallExpr{
		Fun:  ast.NewIdent("pair"),
		Args: nil,
	}
	tryCall := &ast.CallExpr{
		Fun:  ast.NewIdent("Try"),
		Args: []ast.Expr{pairCall},
	}
	pairResults := types.NewTuple(
		types.NewParam(0, nil, "", types.Typ[types.Int]),
		types.NewParam(0, nil, "", types.Typ[types.String]),
		types.NewParam(0, nil, "", types.Universe.Lookup("error").Type()),
	)
	info.Types[pairCall] = types.TypeAndValue{Type: pairResults}
	ctx := &fakeContext{
		fset: fset, info: info, pkg: pkg, call: tryCall, stub: "Try", site: macro.SiteAssign,
		enclosing: fn, pos: tryCall.Pos(),
	}
	_, err := TryExpand(ctx, tryCall)
	if err == nil || !strings.Contains(err.Error(), "Try2") {
		t.Fatalf("got %v", err)
	}
}

func TestCheckStubMatchesKMessages(t *testing.T) {
	if err := checkStubMatchesK("Try", 2); err == nil || !strings.Contains(err.Error(), "Try2") {
		t.Fatalf("Try vs k=2: %v", err)
	}
	if err := checkStubMatchesK("Try0", 1); err == nil || !strings.Contains(err.Error(), "Try1") {
		t.Fatalf("Try0 vs k=1: %v", err)
	}
}

func parseTrySnippet(t *testing.T, src string) (*token.FileSet, *ast.CallExpr, *ast.FuncDecl, *types.Info, *types.Package) {
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
	return fset, nil, fn, info, pkg
}
