package inline

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arcane-craft/go-macro/macro"
)

//macro: syntax-inline
// InlineExpand inlines resolvable same-file function calls or unwraps Inline(expr) at SiteExpr.
func InlineExpand(ctx macro.Context, call *ast.CallExpr) (macro.ExpandResult, error) {
	fset := ctx.FileSet()
	stubN, err := stubExpectedN(ctx.StubName())
	if err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
	}

	site := ctx.Site()
	if err := checkSite(stubN, site); err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
	}

	innerCall, bareExpr, err := innerCallFromArgs(call, stubN)
	if err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
	}

	if stubN == 1 && site == macro.SiteExpr && innerCall == nil {
		if len(call.Args) != 1 {
			return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "Inline expects exactly one argument")
		}
		return macro.ExpandResult{Target: macro.SpliceReplaceCallExpr, Expr: bareExpr}, nil
	}

	if innerCall == nil {
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "expected function call argument")
	}

	n, err := calleeResultCount(ctx, innerCall)
	if err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
	}
	if err := checkStubMatchesN(ctx.StubName(), n); err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
	}

	fn, err := resolveCalleeFuncDecl(ctx, innerCall)
	if err != nil {
		if stubN == 1 && site == macro.SiteExpr {
			return macro.ExpandResult{Target: macro.SpliceReplaceCallExpr, Expr: bareExpr}, nil
		}
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
	}

	retExprs, bodyStmts, err := inlineableBody(fn, n)
	if err != nil {
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "cannot inline %s: %s", fn.Name.Name, err.Error())
	}

	switch {
	case n == 0:
		stmts, err := substituteStmts(ctx, fn, innerCall, bodyStmts)
		if err != nil {
			return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
		}
		macro.StampStmtPos(ctx.MacroPos(), stmts)
		return macro.ExpandResult{Target: macro.SpliceReplaceExprStmt, Stmts: stmts}, nil
	case n == 1 && (site == macro.SiteExpr || site == macro.SiteReturn):
		expr, err := substituteExpr(ctx, fn, innerCall, retExprs[0])
		if err != nil {
			return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
		}
		if site == macro.SiteExpr {
			return macro.ExpandResult{Target: macro.SpliceReplaceCallExpr, Expr: expr}, nil
		}
		stmts := []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{expr}}}
		macro.StampStmtPos(ctx.MacroPos(), stmts)
		return macro.ExpandResult{Target: macro.SpliceReplaceReturnStmt, Stmts: stmts}, nil
	case n >= 2 && (site == macro.SiteAssign || site == macro.SiteReturn):
		rhs, err := substituteExprs(ctx, fn, innerCall, retExprs)
		if err != nil {
			return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%s", err.Error())
		}
		if site == macro.SiteAssign {
			assign, ok := findAssignStmt(ctx)
			if !ok {
				return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "expected assignment context")
			}
			stmt := &ast.AssignStmt{Tok: assign.Tok, Lhs: assign.Lhs, Rhs: rhs}
			stmts := []ast.Stmt{stmt}
			macro.StampStmtPos(ctx.MacroPos(), stmts)
			return macro.ExpandResult{Target: macro.SpliceReplaceAssignStmt, Stmts: stmts}, nil
		}
		stmt := &ast.ReturnStmt{Results: rhs}
		stmts := []ast.Stmt{stmt}
		macro.StampStmtPos(ctx.MacroPos(), stmts)
		return macro.ExpandResult{Target: macro.SpliceReplaceReturnStmt, Stmts: stmts}, nil
	default:
		return macro.ExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "inline: unexpected site/arity combination")
	}
}

func stubExpectedN(stub string) (int, error) {
	switch stub {
	case "Inline0":
		return 0, nil
	case "Inline":
		return 1, nil
	case "Inline2":
		return 2, nil
	case "Inline3":
		return 3, nil
	default:
		return 0, fmt.Errorf("unknown inline stub %q", stub)
	}
}

func checkStubMatchesN(stub string, n int) error {
	want, err := stubExpectedN(stub)
	if err != nil {
		return err
	}
	if want != n {
		switch n {
		case 0:
			return fmt.Errorf("use Inline0 for void callee")
		case 1:
			return fmt.Errorf("use Inline for single-value callee")
		case 2:
			return fmt.Errorf("use Inline2 for 2-value callee")
		case 3:
			return fmt.Errorf("use Inline3 for 3-value callee")
		default:
			return fmt.Errorf("stub %q does not match %d result callee", stub, n)
		}
	}
	return nil
}

func checkSite(stubN int, site macro.CallSiteKind) error {
	switch stubN {
	case 0:
		if site != macro.SiteStmt {
			return fmt.Errorf("Inline0 only allowed as statement")
		}
	case 1:
		if site != macro.SiteExpr && site != macro.SiteReturn {
			return fmt.Errorf("Inline only allowed in expression or return position")
		}
	case 2, 3:
		if site != macro.SiteAssign && site != macro.SiteReturn {
			return fmt.Errorf("Inline%d only allowed in assignment or return position", stubN)
		}
	}
	return nil
}

func innerCallFromArgs(call *ast.CallExpr, stubN int) (*ast.CallExpr, ast.Expr, error) {
	switch stubN {
	case 0:
		if len(call.Args) != 1 {
			return nil, nil, fmt.Errorf("Inline0 expects one func literal argument")
		}
		fl, ok := call.Args[0].(*ast.FuncLit)
		if !ok {
			return nil, nil, fmt.Errorf("Inline0 expects func literal argument")
		}
		inner := callFromFuncLitBody(fl)
		if inner == nil {
			return nil, nil, fmt.Errorf("Inline0 func literal must contain a single call statement")
		}
		return inner, call.Args[0], nil
	default:
		if len(call.Args) != 1 {
			return nil, nil, fmt.Errorf("Inline* expects one argument expression")
		}
		arg := call.Args[0]
		inner, ok := arg.(*ast.CallExpr)
		if !ok {
			return nil, arg, nil
		}
		return inner, arg, nil
	}
}

func callFromFuncLitBody(fl *ast.FuncLit) *ast.CallExpr {
	if fl.Body == nil || len(fl.Body.List) != 1 {
		return nil
	}
	es, ok := fl.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return nil
	}
	c, ok := es.X.(*ast.CallExpr)
	if !ok {
		return nil
	}
	return c
}

func calleeResultCount(ctx macro.Context, inner *ast.CallExpr) (int, error) {
	tv, ok := ctx.Types().Types[inner]
	if !ok {
		return 0, fmt.Errorf("cannot type inner call")
	}
	if tuple, ok := tv.Type.(*types.Tuple); ok {
		return tuple.Len(), nil
	}
	if sig, ok := tv.Type.Underlying().(*types.Signature); ok {
		if sig.Results() == nil {
			return 0, nil
		}
		return sig.Results().Len(), nil
	}
	return 1, nil
}

func astFileFor(ctx macro.Context) *ast.File {
	pos := ctx.EnclosingFunc().Pos()
	for node := range ctx.Types().Scopes {
		f, ok := node.(*ast.File)
		if !ok {
			continue
		}
		if pos >= f.Pos() && pos <= f.End() {
			return f
		}
	}
	return nil
}

func resolveCalleeFuncDecl(ctx macro.Context, inner *ast.CallExpr) (*ast.FuncDecl, error) {
	ident, ok := inner.Fun.(*ast.Ident)
	if !ok {
		return nil, fmt.Errorf("only direct identifier calls can be inlined")
	}
	obj, ok := ctx.Types().Uses[ident]
	if !ok {
		return nil, fmt.Errorf("cannot resolve callee %s", ident.Name)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("callee %s is not a function", ident.Name)
	}
	file := astFileFor(ctx)
	if file == nil {
		return nil, fmt.Errorf("cannot locate source file")
	}
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if def, ok := ctx.Types().Defs[fd.Name]; ok && def == fn {
			return fd, nil
		}
	}
	return nil, fmt.Errorf("function %s not found in file", ident.Name)
}

func inlineableBody(fn *ast.FuncDecl, n int) ([]ast.Expr, []ast.Stmt, error) {
	if fn.Body == nil {
		return nil, nil, fmt.Errorf("missing body")
	}
	if fn.Type.Params != nil {
		for _, f := range fn.Type.Params.List {
			if _, ok := f.Type.(*ast.Ellipsis); ok {
				return nil, nil, fmt.Errorf("variadic parameters not supported")
			}
		}
	}
	if n == 0 {
		for _, s := range fn.Body.List {
			if ret, ok := s.(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
				return nil, nil, fmt.Errorf("void function must not return values")
			}
		}
		return nil, fn.Body.List, nil
	}
	if len(fn.Body.List) != 1 {
		return nil, nil, fmt.Errorf("body must be a single return statement")
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		return nil, nil, fmt.Errorf("body must be a single return statement")
	}
	if len(ret.Results) != n {
		return nil, nil, fmt.Errorf("return arity %d, want %d", len(ret.Results), n)
	}
	return ret.Results, nil, nil
}

func paramSubstMap(ctx macro.Context, fn *ast.FuncDecl, inner *ast.CallExpr) (map[*types.Var]ast.Expr, error) {
	if fn.Type.Params == nil {
		if len(inner.Args) != 0 {
			return nil, fmt.Errorf("argument count mismatch")
		}
		return map[*types.Var]ast.Expr{}, nil
	}
	subst := make(map[*types.Var]ast.Expr)
	argIdx := 0
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			if argIdx >= len(inner.Args) {
				return nil, fmt.Errorf("argument count mismatch")
			}
			argIdx++
			continue
		}
		for _, name := range field.Names {
			if argIdx >= len(inner.Args) {
				return nil, fmt.Errorf("argument count mismatch")
			}
			obj := ctx.Types().Defs[name]
			v, ok := obj.(*types.Var)
			if !ok {
				return nil, fmt.Errorf("parameter %s not found", name.Name)
			}
			subst[v] = cloneExpr(inner.Args[argIdx])
			argIdx++
		}
	}
	if argIdx != len(inner.Args) {
		return nil, fmt.Errorf("argument count mismatch")
	}
	return subst, nil
}

func substituteExpr(ctx macro.Context, fn *ast.FuncDecl, inner *ast.CallExpr, expr ast.Expr) (ast.Expr, error) {
	subst, err := paramSubstMap(ctx, fn, inner)
	if err != nil {
		return nil, err
	}
	return applySubstExpr(ctx, expr, subst), nil
}

func substituteExprs(ctx macro.Context, fn *ast.FuncDecl, inner *ast.CallExpr, exprs []ast.Expr) ([]ast.Expr, error) {
	subst, err := paramSubstMap(ctx, fn, inner)
	if err != nil {
		return nil, err
	}
	out := make([]ast.Expr, len(exprs))
	for i, e := range exprs {
		out[i] = applySubstExpr(ctx, e, subst)
	}
	return out, nil
}

func substituteStmts(ctx macro.Context, fn *ast.FuncDecl, inner *ast.CallExpr, stmts []ast.Stmt) ([]ast.Stmt, error) {
	subst, err := paramSubstMap(ctx, fn, inner)
	if err != nil {
		return nil, err
	}
	out := make([]ast.Stmt, len(stmts))
	for i, s := range stmts {
		out[i] = applySubstStmt(ctx, s, subst)
	}
	return out, nil
}

func applySubstExpr(ctx macro.Context, expr ast.Expr, subst map[*types.Var]ast.Expr) ast.Expr {
	var sub func(ast.Expr) ast.Expr
	sub = func(e ast.Expr) ast.Expr {
		if e == nil {
			return nil
		}
		switch x := e.(type) {
		case *ast.Ident:
			if obj, ok := ctx.Types().Uses[x]; ok {
				if v, ok := obj.(*types.Var); ok {
					if rep, ok := subst[v]; ok {
						return cloneExpr(rep)
					}
				}
			}
			return cloneExpr(x)
		case *ast.BinaryExpr:
			return &ast.BinaryExpr{X: sub(x.X), Op: x.Op, Y: sub(x.Y)}
		case *ast.UnaryExpr:
			return &ast.UnaryExpr{Op: x.Op, X: sub(x.X)}
		case *ast.ParenExpr:
			return &ast.ParenExpr{X: sub(x.X)}
		case *ast.CallExpr:
			out := &ast.CallExpr{Fun: sub(x.Fun)}
			for _, a := range x.Args {
				out.Args = append(out.Args, sub(a))
			}
			return out
		case *ast.BasicLit:
			return &ast.BasicLit{Kind: x.Kind, Value: x.Value}
		case *ast.SelectorExpr:
			return &ast.SelectorExpr{X: sub(x.X), Sel: x.Sel}
		default:
			return x
		}
	}
	return sub(expr)
}

func applySubstStmt(ctx macro.Context, stmt ast.Stmt, subst map[*types.Var]ast.Expr) ast.Stmt {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return &ast.ExprStmt{X: applySubstExpr(ctx, s.X, subst)}
	case *ast.AssignStmt:
		out := &ast.AssignStmt{Tok: s.Tok}
		for _, lhs := range s.Lhs {
			out.Lhs = append(out.Lhs, applySubstExpr(ctx, lhs, subst))
		}
		for _, rhs := range s.Rhs {
			out.Rhs = append(out.Rhs, applySubstExpr(ctx, rhs, subst))
		}
		return out
	case *ast.ReturnStmt:
		out := &ast.ReturnStmt{}
		for _, r := range s.Results {
			out.Results = append(out.Results, applySubstExpr(ctx, r, subst))
		}
		return out
	default:
		return s
	}
}

func findAssignStmt(ctx macro.Context) (*ast.AssignStmt, bool) {
	call := ctx.Call()
	var found *ast.AssignStmt
	inspect := func(body *ast.BlockStmt) {
		if body == nil {
			return
		}
		ast.Inspect(body, func(n ast.Node) bool {
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
	switch fn := ctx.EnclosingFunc().(type) {
	case *ast.FuncDecl:
		inspect(fn.Body)
	case *ast.FuncLit:
		inspect(fn.Body)
	}
	return found, found != nil
}

func cloneExpr(e ast.Expr) ast.Expr {
	if e == nil {
		return nil
	}
	switch x := e.(type) {
	case *ast.Ident:
		return ast.NewIdent(x.Name)
	case *ast.BasicLit:
		return &ast.BasicLit{Kind: x.Kind, Value: x.Value}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{X: cloneExpr(x.X), Op: x.Op, Y: cloneExpr(x.Y)}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{Op: x.Op, X: cloneExpr(x.X)}
	case *ast.ParenExpr:
		return &ast.ParenExpr{X: cloneExpr(x.X)}
	case *ast.CallExpr:
		out := &ast.CallExpr{Fun: cloneExpr(x.Fun)}
		for _, a := range x.Args {
			out.Args = append(out.Args, cloneExpr(a))
		}
		return out
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: cloneExpr(x.X), Sel: ast.NewIdent(x.Sel.Name)}
	default:
		return x
	}
}

func cloneStmt(s ast.Stmt) ast.Stmt {
	switch x := s.(type) {
	case *ast.ExprStmt:
		return &ast.ExprStmt{X: cloneExpr(x.X)}
	case *ast.AssignStmt:
		out := &ast.AssignStmt{Tok: x.Tok}
		for _, e := range x.Lhs {
			out.Lhs = append(out.Lhs, cloneExpr(e))
		}
		for _, e := range x.Rhs {
			out.Rhs = append(out.Rhs, cloneExpr(e))
		}
		return out
	case *ast.ReturnStmt:
		out := &ast.ReturnStmt{}
		for _, e := range x.Results {
			out.Results = append(out.Results, cloneExpr(e))
		}
		return out
	default:
		return x
	}
}
