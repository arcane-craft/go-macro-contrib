package try

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/arcane-craft/go-macro/macro"
)

//macro: syntax-try

// TryExpand expands Try* macro calls into error-handling statement blocks.
func TryExpand(ctx macro.Context, call *ast.CallExpr) (macro.ExpandResult, error) {
	if len(call.Args) != 1 {
		return macro.ExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "Try* expects one argument expression")
	}
	outer, err := outerResults(ctx)
	if err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}
	k, err := calleePayloadCount(ctx, call.Args[0])
	if err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}
	if err := checkStubMatchesK(ctx.StubName(), k); err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}

	fset := ctx.FileSet()
	expr := call.Args[0]
	errIdent := ctx.TempIdent("_err")
	valIdents := make([]*ast.Ident, k)
	for i := 0; i < k; i++ {
		valIdents[i] = ctx.TempIdent("_v")
	}

	var assignLHS []ast.Expr
	if k == 0 {
		assignLHS = []ast.Expr{errIdent}
	} else {
		assignLHS = make([]ast.Expr, k+1)
		for i := 0; i < k; i++ {
			assignLHS[i] = valIdents[i]
		}
		assignLHS[k] = errIdent
	}
	assign := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: assignLHS,
		Rhs: []ast.Expr{expr},
	}
	ifStmt := errorReturnIf(fset, errIdent, outer)
	stmts := []ast.Stmt{assign, ifStmt}

	switch ctx.Site() {
	case macro.SiteAssign:
		assignStmt, ok := findAssignStmt(ctx)
		if !ok {
			return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "expected assignment context")
		}
		// After check, append success assignment to lhs
		success := successAssign(assignStmt, valIdents, k)
		stmts = append(stmts, success)
		return tryExpandResult(ctx, stmts), nil
	case macro.SiteReturn:
		if k == 0 {
			// return Try0: only error check, success is fallthrough — replace with assign+if only
			return tryExpandResult(ctx, stmts), nil
		}
		retVals := make([]ast.Expr, len(outer))
		for i := 0; i < k; i++ {
			retVals[i] = valIdents[i]
		}
		retVals[len(outer)-1] = ast.NewIdent("nil")
		successRet := &ast.ReturnStmt{Results: retVals}
		stmts = append(stmts, successRet)
		return tryExpandResult(ctx, stmts), nil
	case macro.SiteStmt:
		if k != 0 {
			return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "Try0 only allowed as statement for k=0")
		}
		return tryExpandResult(ctx, stmts), nil
	default:
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "Try* not allowed in expression position")
	}
}

func tryExpandResult(ctx macro.Context, stmts []ast.Stmt) macro.ExpandResult {
	macro.StampStmtPos(ctx.MacroPos(), stmts)
	return macro.ExpandResult{Stmts: stmts}
}

type resultType struct {
	typ   types.Type
	names []*ast.Ident
}

func outerResults(ctx macro.Context) ([]resultType, error) {
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

func calleePayloadCount(ctx macro.Context, expr ast.Expr) (k int, err error) {
	tv, ok := ctx.Types().Types[expr]
	if !ok {
		return 0, fmt.Errorf("cannot type inner expression")
	}
	// Call or value typed as plain error (e.g. Try0(close()) where close returns error).
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

func errorReturnIf(fset *token.FileSet, errIdent *ast.Ident, outer []resultType) *ast.IfStmt {
	results := make([]ast.Expr, len(outer))
	for i := 0; i < len(outer)-1; i++ {
		if len(outer[i].names) > 0 {
			results[i] = ast.NewIdent(outer[i].names[0].Name)
		} else {
			results[i] = zeroValueExpr(outer[i].typ)
		}
	}
	results[len(outer)-1] = errIdent
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  errIdent,
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: results}}},
	}
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

func findAssignStmt(ctx macro.Context) (*ast.AssignStmt, bool) {
	call := ctx.Call()
	// Reconstruct assign from call position is done by expander before TryExpand;
	// use Call's parent via Types isn't available — inspect EnclosingFunc body.
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

func successAssign(orig *ast.AssignStmt, vals []*ast.Ident, k int) ast.Stmt {
	if k == 0 {
		return &ast.EmptyStmt{}
	}
	lhs := orig.Lhs
	if len(lhs) == 1 && k > 1 {
		// multi-value lhs
		return &ast.AssignStmt{Tok: orig.Tok, Lhs: lhs, Rhs: exprSlice(vals[:k])}
	}
	rhs := exprSlice(vals[:min(k, len(lhs))])
	return &ast.AssignStmt{Tok: orig.Tok, Lhs: lhs, Rhs: rhs}
}

func exprSlice(ids []*ast.Ident) []ast.Expr {
	out := make([]ast.Expr, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
