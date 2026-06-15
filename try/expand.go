package try

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
)

//macro: try
// TryExpander expands Try* macro calls into error-handling statement blocks.
var TryExpander = macro.SyntaxCase(buildTryClauses()...)

func buildTryClauses() []macro.Clause {
	stubs := []string{"Try0", "Try", "Try2", "Try3"}
	patterns := []string{
		`$lhs ... := %s($inner)`,
		`$lhs ... = %s($inner)`,
		`var $lhs ... = %s($inner)`,
		`return $vals ... , %s($inner)`,
	}
	var clauses []macro.Clause
	for _, stub := range stubs {
		for _, pat := range patterns {
			clauses = append(clauses, macro.Clause{
				Pattern:   fmt.Sprintf(pat, stub),
				Transform: tryTransform,
			})
		}
		if stub == "Try0" {
			clauses = append(clauses, macro.Clause{
				Pattern:   `Try0($inner);`,
				Transform: tryTransform,
			})
		}
	}
	return clauses
}

func tryTransform(ctx macro.Context, site macro.Syntax, binds macro.Bindings) (macro.Syntax, error) {
	call, ok := site.Underlying().(*ast.CallExpr)
	if !ok {
		return nil, fmt.Errorf("expected call site")
	}
	if len(call.Args) != 1 {
		return nil, fmt.Errorf("Try* expects one argument expression")
	}
	inner, ok := binds.Get("inner")
	if !ok {
		return nil, fmt.Errorf("missing inner binding")
	}
	innerExpr, ok := inner.Underlying().(ast.Expr)
	if !ok {
		return nil, fmt.Errorf("inner is not an expression")
	}

	stubName := invokedName(call.Fun)
	outer, err := enclosingOuterResults(ctx, site)
	if err != nil {
		return nil, err
	}
	k, err := calleePayloadCount(ctx, innerExpr)
	if err != nil {
		return nil, err
	}
	if err := checkStubMatchesK(stubName, k); err != nil {
		return nil, err
	}

	errIdent := ctx.TempIdent("_err")
	valIdents := make([]*ast.Ident, k)
	for i := 0; i < k; i++ {
		valIdents[i] = ctx.TempIdent("_v")
	}

	coreTpl, coreBinds, err := tryCoreQuote(k, inner, errIdent, valIdents, ctx, outer)
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

	if _, ok := binds.Elems("vals"); ok {
		if k > 0 {
			successTpl, successBinds := trySuccessReturnQuote(k, valIdents)
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
	} else if lhsElems, ok := binds.Elems("lhs"); ok {
		if k > 0 {
			successTpl, successBinds, err := trySuccessAssignQuote(site, call, lhsElems, valIdents, k)
			if err != nil {
				return nil, err
			}
			if successTpl != "" {
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
		}
	} else {
		if k != 0 {
			return nil, fmt.Errorf("Try0 only allowed as statement for k=0")
		}
	}

	macro.StampStmtPos(site.MacroPos(), stmts)
	return macro.WrapStmts(stmts), nil
}

func tryCoreQuote(k int, inner macro.Syntax, errIdent *ast.Ident, valIdents []*ast.Ident, ctx macro.Context, outer *types.Tuple) (string, map[string]macro.Syntax, error) {
	binds := map[string]macro.Syntax{
		"err":  macro.WrapExpr(errIdent),
		"call": inner,
	}
	var b strings.Builder
	if k == 0 {
		b.WriteString("#err := #call")
	} else {
		for i := 0; i < k; i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			key := fmt.Sprintf("v%d", i)
			fmt.Fprintf(&b, "#%s", key)
			binds[key] = macro.WrapExpr(valIdents[i])
		}
		b.WriteString(", #err := #call")
	}
	b.WriteString("\nif #err != nil {\n    return ")
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

func trySuccessReturnQuote(k int, valIdents []*ast.Ident) (string, map[string]macro.Syntax) {
	binds := map[string]macro.Syntax{"nil": macro.WrapExpr(ast.NewIdent("nil"))}
	var b strings.Builder
	b.WriteString("return ")
	for i := 0; i < k; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		key := fmt.Sprintf("v%d", i)
		fmt.Fprintf(&b, "#%s", key)
		binds[key] = macro.WrapExpr(valIdents[i])
	}
	if k > 0 {
		b.WriteString(", ")
	}
	b.WriteString("#nil")
	return b.String(), binds
}

func trySuccessAssignQuote(site macro.Syntax, call *ast.CallExpr, lhsElems []macro.Syntax, vals []*ast.Ident, k int) (string, map[string]macro.Syntax, error) {
	if k == 0 {
		return "", nil, nil
	}
	tok := token.DEFINE
	if assign, ok := findAssignStmtForCall(site, call); ok && assign.Tok == token.ASSIGN {
		tok = token.ASSIGN
	}
	binds := map[string]macro.Syntax{}
	var lhs strings.Builder
	nLHS := min(k, len(lhsElems))
	if len(lhsElems) == 1 && k > 1 {
		nLHS = 1
		binds["lhs0"] = lhsElems[0]
		lhs.WriteString("#lhs0")
	} else {
		for i := 0; i < nLHS; i++ {
			if i > 0 {
				lhs.WriteString(", ")
			}
			key := fmt.Sprintf("lhs%d", i)
			fmt.Fprintf(&lhs, "#%s", key)
			binds[key] = lhsElems[i]
		}
	}
	var rhs strings.Builder
	nRHS := k
	if len(lhsElems) == 1 && k > 1 {
		nRHS = k
	} else {
		nRHS = min(k, len(lhsElems))
	}
	for i := 0; i < nRHS; i++ {
		if i > 0 {
			rhs.WriteString(", ")
		}
		key := fmt.Sprintf("v%d", i)
		fmt.Fprintf(&rhs, "#%s", key)
		binds[key] = macro.WrapExpr(vals[i])
	}
	tpl := fmt.Sprintf("%s %s %s;", lhs.String(), tok.String(), rhs.String())
	return tpl, binds, nil
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

func isErrorType(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

func checkStubMatchesK(stub string, k int) error {
	switch stub {
	case "Try0":
		if k != 0 {
			return fmt.Errorf("use Try%d for %d payloads", k, k)
		}
	case "Try":
		if k != 1 {
			if k == 0 {
				return fmt.Errorf("use Try0 for error-only callee")
			}
			return fmt.Errorf("use Try%d for %d payloads", k, k)
		}
	case "Try2":
		if k != 2 {
			return fmt.Errorf("Try2 requires callee with 2 payloads + error, got k=%d", k)
		}
	case "Try3":
		if k != 3 {
			return fmt.Errorf("Try3 requires callee with 3 payloads + error, got k=%d", k)
		}
	default:
		if k >= 2 && stub != fmt.Sprintf("Try%d", k) {
			return fmt.Errorf("stub %q does not match payload count %d", stub, k)
		}
	}
	return nil
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

func invokedName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if id, ok := f.X.(*ast.Ident); ok {
			return id.Name + "." + f.Sel.Name
		}
		return f.Sel.Name
	default:
		return ""
	}
}
