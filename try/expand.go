package try

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
	"github.com/arcane-craft/go-macro/macro/quote"
)

// macro: syntax-try
// TryExpand expands Try* macro calls into error-handling statement blocks.
func TryExpand(ctx macro.CallContext, call *ast.CallExpr) (macro.CallExpandResult, error) {
	if len(call.Args) != 1 {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "Try* expects one argument expression")
	}
	outer, err := outerResults(ctx)
	if err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}
	k, err := calleePayloadCount(ctx, call.Args[0])
	if err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}
	if err := checkStubMatchesK(ctx.StubName(), k); err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}

	fset := ctx.FileSet()
	expr := call.Args[0]
	errIdent := ctx.TempIdent("_err")
	valIdents := make([]*ast.Ident, k)
	for i := 0; i < k; i++ {
		valIdents[i] = ctx.TempIdent("_v")
	}

	tpl, args := tryQuoteTemplate(k, expr, errIdent, valIdents, outer)
	stmts, err := quote.Stmts(tpl, args)
	if err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
	}

	switch ctx.Site() {
	case macro.SiteAssign:
		assignStmt, ok := findAssignStmt(ctx)
		if !ok {
			return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "expected assignment context")
		}
		successTpl, successArgs, err := trySuccessAssignTemplate(assignStmt, valIdents, k)
		if err != nil {
			return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
		}
		if successTpl != "" {
			success, err := quote.Stmts(successTpl, successArgs)
			if err != nil {
				return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
			}
			stmts = append(stmts, success...)
		}
		return tryExpandResult(ctx, stmts), nil
	case macro.SiteReturn:
		if k == 0 {
			return tryExpandResult(ctx, stmts), nil
		}
		successTpl, successArgs := trySuccessReturnTemplate(k, valIdents)
		success, err := quote.Stmts(successTpl, successArgs)
		if err != nil {
			return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
		}
		stmts = append(stmts, success...)
		return tryExpandResult(ctx, stmts), nil
	case macro.SiteStmt:
		if k != 0 {
			return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "Try0 only allowed as statement for k=0")
		}
		return tryExpandResult(ctx, stmts), nil
	default:
		return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "Try* not allowed in expression position")
	}
}

func tryQuoteTemplate(k int, call ast.Expr, errIdent *ast.Ident, valIdents []*ast.Ident, outer []resultType) (string, map[string]any) {
	args := map[string]any{
		"err":  errIdent,
		"call": call,
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
			args[key] = valIdents[i]
		}
		b.WriteString(", #err := #call")
	}
	b.WriteString("\nif #err != nil {\n    return ")
	for i := 0; i < len(outer)-1; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		key := fmt.Sprintf("z%d", i)
		fmt.Fprintf(&b, "#%s", key)
		if len(outer[i].names) > 0 {
			args[key] = outer[i].names[0].Name
		} else {
			args[key] = zeroValueExpr(outer[i].typ)
		}
	}
	if len(outer) > 1 {
		b.WriteString(", ")
	}
	b.WriteString("#err\n}")
	return b.String(), args
}

func trySuccessReturnTemplate(k int, valIdents []*ast.Ident) (string, map[string]any) {
	args := map[string]any{"nil": "nil"}
	var b strings.Builder
	b.WriteString("return ")
	for i := 0; i < k; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		key := fmt.Sprintf("v%d", i)
		fmt.Fprintf(&b, "#%s", key)
		args[key] = valIdents[i]
	}
	if k > 0 {
		b.WriteString(", ")
	}
	b.WriteString("#nil")
	return b.String(), args
}

func trySuccessAssignTemplate(orig *ast.AssignStmt, vals []*ast.Ident, k int) (string, map[string]any, error) {
	if k == 0 {
		return "", nil, nil
	}
	tok := ":="
	if orig.Tok.String() == "=" {
		tok = "="
	}
	args := map[string]any{}
	var lhs strings.Builder
	nLHS := min(k, len(orig.Lhs))
	if len(orig.Lhs) == 1 && k > 1 {
		nLHS = 1
		args["lhs0"] = orig.Lhs[0]
		lhs.WriteString("#lhs0")
	} else {
		for i := 0; i < nLHS; i++ {
			if i > 0 {
				lhs.WriteString(", ")
			}
			key := fmt.Sprintf("lhs%d", i)
			fmt.Fprintf(&lhs, "#%s", key)
			args[key] = orig.Lhs[i]
		}
	}
	var rhs strings.Builder
	nRHS := k
	if len(orig.Lhs) == 1 && k > 1 {
		nRHS = k
	} else {
		nRHS = min(k, len(orig.Lhs))
	}
	for i := 0; i < nRHS; i++ {
		if i > 0 {
			rhs.WriteString(", ")
		}
		key := fmt.Sprintf("v%d", i)
		fmt.Fprintf(&rhs, "#%s", key)
		args[key] = vals[i]
	}
	tpl := fmt.Sprintf("%s %s %s", lhs.String(), tok, rhs.String())
	return tpl, args, nil
}

func tryExpandResult(ctx macro.CallContext, stmts []ast.Stmt) macro.CallExpandResult {
	macro.StampStmtPos(ctx.MacroPos(), stmts)
	var target macro.SpliceTarget
	switch ctx.Site() {
	case macro.SiteAssign:
		target = macro.SpliceReplaceAssignStmt
	case macro.SiteReturn:
		target = macro.SpliceReplaceReturnStmt
	case macro.SiteStmt:
		target = macro.SpliceReplaceExprStmt
	default:
		target = macro.SpliceReplaceAssignStmt
	}
	return macro.CallExpandResult{Target: target, Stmts: stmts}
}

type resultType struct {
	typ   types.Type
	names []*ast.Ident
}

func outerResults(ctx macro.CallContext) ([]resultType, error) {
	var results *ast.FieldList
	switch fn := ctx.EnclosingFunc().(type) {
	case *ast.FuncDecl:
		results = fn.Type.Results
	case *ast.FuncLit:
		results = fn.Type.Results
	default:
		return nil, fmt.Errorf("invalid enclosing function")
	}
	if results == nil || len(results.List) == 0 {
		return nil, fmt.Errorf("function must return at least error as last value")
	}
	var sig *types.Signature
	if fn, ok := ctx.EnclosingFunc().(*ast.FuncDecl); ok && fn.Name != nil {
		if obj, ok := ctx.Types().Defs[fn.Name]; ok {
			if f, ok := obj.(*types.Func); ok {
				sig, _ = f.Type().(*types.Signature)
			}
		}
	}
	var out []resultType
	if sig != nil && sig.Results() != nil {
		for i := 0; i < sig.Results().Len(); i++ {
			out = append(out, resultType{typ: sig.Results().At(i).Type(), names: nil})
		}
	} else {
		for _, f := range results.List {
			t := ctx.Types().TypeOf(f.Type)
			if t == nil {
				return nil, fmt.Errorf("cannot type outer results")
			}
			out = append(out, resultType{typ: t, names: f.Names})
		}
	}
	last := out[len(out)-1]
	if !isErrorType(last.typ) {
		return nil, fmt.Errorf("outer function must end with error return")
	}
	return out, nil
}

func calleePayloadCount(ctx macro.CallContext, expr ast.Expr) (k int, err error) {
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

func zeroValueExpr(t types.Type) ast.Expr {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.String:
			return &ast.BasicLit{Kind: token.STRING, Value: `""`}
		case types.Bool:
			return &ast.Ident{Name: "false"}
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
			types.Float32, types.Float64, types.Complex64, types.Complex128:
			return &ast.BasicLit{Kind: token.INT, Value: "0"}
		}
	}
	return &ast.Ident{Name: "nil"}
}

func findAssignStmt(ctx macro.CallContext) (*ast.AssignStmt, bool) {
	call := ctx.Call()
	var found *ast.AssignStmt
	switch fn := ctx.EnclosingFunc().(type) {
	case *ast.FuncDecl:
		ast.Inspect(fn.Body, func(n ast.Node) bool {
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
	case *ast.FuncLit:
		ast.Inspect(fn.Body, func(n ast.Node) bool {
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
	}
	return found, found != nil
}
