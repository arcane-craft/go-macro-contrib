package with

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
)

//macro: with
// WithExpander expands With macro calls into error handling plus defer Close.
var WithExpander = macro.SyntaxCase(buildWithClauses()...)

func buildWithClauses() []macro.Clause {
	patterns := []string{
		`$lhs ... := With($inner)`,
		`$lhs ... = With($inner)`,
		`var $lhs ... = With($inner)`,
		`return $vals ... , With($inner)`,
	}
	clauses := make([]macro.Clause, len(patterns))
	for i, pat := range patterns {
		clauses[i] = macro.Clause{Pattern: pat, Transform: withTransform}
	}
	return clauses
}

func withTransform(ctx macro.Context, site macro.Syntax, binds macro.Bindings) (macro.Syntax, error) {
	call, ok := site.Underlying().(*ast.CallExpr)
	if !ok {
		return nil, fmt.Errorf("expected call site")
	}
	if len(call.Args) != 1 {
		return nil, fmt.Errorf("With expects one argument expression")
	}
	inner, ok := binds.Get("inner")
	if !ok {
		return nil, fmt.Errorf("missing inner binding")
	}
	innerExpr, ok := inner.Underlying().(ast.Expr)
	if !ok {
		return nil, fmt.Errorf("inner is not an expression")
	}

	outer, err := enclosingOuterResults(ctx, site)
	if err != nil {
		return nil, err
	}
	k, err := calleePayloadCount(ctx, innerExpr)
	if err != nil {
		return nil, err
	}
	if k != 1 {
		return nil, fmt.Errorf("With requires callee with (T, error); got %d payloads before error", k)
	}
	payloadType, err := calleePayloadType(ctx, innerExpr)
	if err != nil {
		return nil, err
	}
	if err := checkImplementsCloser(payloadType); err != nil {
		return nil, err
	}

	errIdent := ctx.TempIdent("_err")
	valIdent := ctx.TempIdent("_v")

	coreTpl, coreBinds, err := withCoreQuote(inner, errIdent, valIdent, ctx, outer)
	if err != nil {
		return nil, err
	}
	core, err := macro.Quote(coreTpl, coreBinds)
	if err != nil {
		return nil, err
	}
	coreStmts, err := core.ToStmts()
	if err != nil {
		return nil, err
	}
	stmts := coreStmts

	deferOut, err := macro.Quote("defer func() { _ = #v0.Close() }();", map[string]macro.Syntax{
		"v0": macro.WrapExpr(valIdent),
	})
	if err != nil {
		return nil, err
	}
	deferStmts, err := deferOut.ToStmts()
	if err != nil {
		return nil, err
	}
	stmts = append(stmts, deferStmts...)

	if _, ok := binds.Elems("vals"); ok {
		successTpl, successBinds := withSuccessReturnQuote(valIdent)
		success, err := macro.Quote(successTpl, successBinds)
		if err != nil {
			return nil, err
		}
		successStmts, err := success.ToStmts()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, successStmts...)
	} else if lhsElems, ok := binds.Elems("lhs"); ok {
		successTpl, successBinds, err := withSuccessAssignQuote(site, call, lhsElems, valIdent)
		if err != nil {
			return nil, err
		}
		success, err := macro.Quote(successTpl, successBinds)
		if err != nil {
			return nil, err
		}
		successStmts, err := success.ToStmts()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, successStmts...)
	}

	macro.StampStmtPos(site.MacroPos(), stmts)
	return macro.WrapStmts(stmts), nil
}

func withCoreQuote(inner macro.Syntax, errIdent, valIdent *ast.Ident, ctx macro.Context, outer *types.Tuple) (string, map[string]macro.Syntax, error) {
	binds := map[string]macro.Syntax{
		"err":  macro.WrapExpr(errIdent),
		"v0":   macro.WrapExpr(valIdent),
		"call": inner,
	}
	var b strings.Builder
	b.WriteString("#v0, #err := #call\nif #err != nil {\n    return ")
	for i := 0; i < outer.Len()-1; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		key := fmt.Sprintf("z%d", i)
		fmt.Fprintf(&b, "#%s", key)
		z, err := macro.ZeroSyntax(ctx, outer.At(i).Type())
		if err != nil {
			return "", nil, err
		}
		binds[key] = z
	}
	if outer.Len() > 1 {
		b.WriteString(", ")
	}
	b.WriteString("#err\n}")
	return b.String(), binds, nil
}

func withSuccessReturnQuote(valIdent *ast.Ident) (string, map[string]macro.Syntax) {
	return "return #v0, #nil;", map[string]macro.Syntax{
		"v0":  macro.WrapExpr(valIdent),
		"nil": macro.WrapExpr(ast.NewIdent("nil")),
	}
}

func withSuccessAssignQuote(site macro.Syntax, call *ast.CallExpr, lhsElems []macro.Syntax, val *ast.Ident) (string, map[string]macro.Syntax, error) {
	if len(lhsElems) == 0 {
		return "", nil, fmt.Errorf("expected lhs binding")
	}
	tok := token.DEFINE
	if assign, ok := findAssignStmtForCall(site, call); ok && assign.Tok == token.ASSIGN {
		tok = token.ASSIGN
	}
	return fmt.Sprintf("#lhs0 %s #v0;", tok.String()), map[string]macro.Syntax{
		"lhs0": lhsElems[0],
		"v0":   macro.WrapExpr(val),
	}, nil
}

func enclosingOuterResults(ctx macro.Context, site macro.Syntax) (*types.Tuple, error) {
	results, err := macro.EnclosingResults(ctx, site)
	if err != nil {
		return nil, err
	}
	if results == nil || results.Len() == 0 {
		return nil, fmt.Errorf("function must return at least error as last value")
	}
	last := results.At(results.Len() - 1).Type()
	if !isErrorType(last) {
		return nil, fmt.Errorf("outer function must end with error return")
	}
	return results, nil
}

func calleePayloadCount(ctx macro.Context, expr ast.Expr) (k int, err error) {
	tv, ok := ctx.Types().Types[expr]
	if !ok {
		return 0, fmt.Errorf("cannot type inner expression")
	}
	if isErrorType(tv.Type) {
		return 0, nil
	}
	tuple, ok := tv.Type.(*types.Tuple)
	if !ok {
		sig, ok := tv.Type.Underlying().(*types.Signature)
		if ok && sig.Results() != nil {
			n := sig.Results().Len()
			if n == 0 {
				return 0, fmt.Errorf("inner callee must return values ending with error")
			}
			last := sig.Results().At(n - 1).Type()
			if !isErrorType(last) {
				return 0, fmt.Errorf("inner callee must end with error")
			}
			return n - 1, nil
		}
		return 0, fmt.Errorf("inner expression must be multi-value call")
	}
	n := tuple.Len()
	if n == 0 {
		return 0, fmt.Errorf("inner callee must return values ending with error")
	}
	last := tuple.At(n - 1).Type()
	if !isErrorType(last) {
		return 0, fmt.Errorf("inner callee must end with error")
	}
	return n - 1, nil
}

func calleePayloadType(ctx macro.Context, expr ast.Expr) (types.Type, error) {
	tv, ok := ctx.Types().Types[expr]
	if !ok {
		return nil, fmt.Errorf("cannot type inner expression")
	}
	if tuple, ok := tv.Type.(*types.Tuple); ok {
		if tuple.Len() < 2 {
			return nil, fmt.Errorf("inner callee must return (T, error)")
		}
		return tuple.At(0).Type(), nil
	}
	sig, ok := tv.Type.Underlying().(*types.Signature)
	if !ok || sig.Results() == nil || sig.Results().Len() < 2 {
		return nil, fmt.Errorf("inner callee must return (T, error)")
	}
	return sig.Results().At(0).Type(), nil
}

func checkImplementsCloser(t types.Type) error {
	iface, err := ioCloserInterface()
	if err != nil {
		return err
	}
	if !types.Implements(t, iface) {
		return fmt.Errorf("payload type must implement io.Closer")
	}
	return nil
}

func ioCloserInterface() (*types.Interface, error) {
	pkg, err := importer.Default().Import("io")
	if err != nil {
		return nil, fmt.Errorf("cannot import io: %w", err)
	}
	obj := pkg.Scope().Lookup("Closer")
	if obj == nil {
		return nil, fmt.Errorf("io.Closer not found")
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("io.Closer is not an interface")
	}
	return iface, nil
}

func isErrorType(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

func findAssignStmtForCall(site macro.Syntax, call *ast.CallExpr) (*ast.AssignStmt, bool) {
	fc, ok := site.(macro.FileCarrier)
	if !ok {
		return nil, false
	}
	file := fc.ExpansionFile()
	var found *ast.AssignStmt
	ast.Inspect(file, func(n ast.Node) bool {
		if a, ok := n.(*ast.AssignStmt); ok {
			for _, rhs := range a.Rhs {
				if rhs == call {
					found = a
					return false
				}
			}
		}
		return true
	})
	return found, found != nil
}
