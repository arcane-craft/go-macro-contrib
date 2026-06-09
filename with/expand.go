package with

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
	"github.com/arcane-craft/go-macro/macro/quote"
)

//macro: with
// WithExpand expands With macro calls into error handling plus defer Close.
func WithExpand(ctx macro.CallContext, call *ast.CallExpr) (macro.CallExpandResult, error) {
	if len(call.Args) != 1 {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "With expects one argument expression")
	}
	if ctx.StubName() != "With" {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "unknown with stub %q", ctx.StubName())
	}

	switch ctx.Site() {
	case macro.SiteAssign, macro.SiteReturn:
	default:
		if ctx.Site() == macro.SiteStmt {
			return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "With not allowed as statement")
		}
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "With not allowed in expression position")
	}

	outer, err := outerResults(ctx)
	if err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}
	k, err := calleePayloadCount(ctx, call.Args[0])
	if err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}
	if k != 1 {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "With requires callee with (T, error); got %d payloads before error", k)
	}
	payloadType, err := calleePayloadType(ctx, call.Args[0])
	if err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}
	if err := checkImplementsCloser(ctx, payloadType); err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(), "%s", err.Error())
	}

	fset := ctx.FileSet()
	expr := call.Args[0]
	errIdent := ctx.TempIdent("_err")
	valIdent := ctx.TempIdent("_v")

	tpl, args := withQuoteTemplate(expr, errIdent, valIdent, outer)
	stmts, err := quote.Stmts(tpl, args)
	if err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
	}
	deferStmts, err := quote.Stmts("defer func() { _ = #v0.Close() }()", map[string]any{"v0": valIdent})
	if err != nil {
		return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
	}
	stmts = append(stmts, deferStmts...)

	switch ctx.Site() {
	case macro.SiteAssign:
		assignStmt, ok := findAssignStmt(ctx)
		if !ok {
			return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "expected assignment context")
		}
		successTpl, successArgs, err := withSuccessAssignTemplate(assignStmt, valIdent)
		if err != nil {
			return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
		}
		success, err := quote.Stmts(successTpl, successArgs)
		if err != nil {
			return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
		}
		stmts = append(stmts, success...)
		return withExpandResult(ctx, stmts), nil
	case macro.SiteReturn:
		successTpl, successArgs := withSuccessReturnTemplate(valIdent)
		success, err := quote.Stmts(successTpl, successArgs)
		if err != nil {
			return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "%v", err)
		}
		stmts = append(stmts, success...)
		return withExpandResult(ctx, stmts), nil
	default:
		return macro.CallExpandResult{}, macro.ErrorAt(fset, ctx.MacroPos(), "With not allowed in expression position")
	}
}

func withQuoteTemplate(call ast.Expr, errIdent, valIdent *ast.Ident, outer []resultType) (string, map[string]any) {
	args := map[string]any{
		"err":  errIdent,
		"v0":   valIdent,
		"call": call,
	}
	var b strings.Builder
	fmt.Fprintf(&b, "#v0, #err := #call\nif #err != nil {\n    return ")
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

func withSuccessReturnTemplate(valIdent *ast.Ident) (string, map[string]any) {
	return "return #v0, #nil", map[string]any{"v0": valIdent, "nil": "nil"}
}

func withSuccessAssignTemplate(orig *ast.AssignStmt, val *ast.Ident) (string, map[string]any, error) {
	tok := ":="
	if orig.Tok.String() == "=" {
		tok = "="
	}
	args := map[string]any{"lhs0": orig.Lhs[0], "v0": val}
	tpl := fmt.Sprintf("#lhs0 %s #v0", tok)
	return tpl, args, nil
}

func withExpandResult(ctx macro.CallContext, stmts []ast.Stmt) macro.CallExpandResult {
	macro.StampStmtPos(ctx.MacroPos(), stmts)
	var target macro.SpliceTarget
	switch ctx.Site() {
	case macro.SiteAssign:
		target = macro.SpliceReplaceAssignStmt
	case macro.SiteReturn:
		target = macro.SpliceReplaceReturnStmt
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

func calleePayloadType(ctx macro.CallContext, expr ast.Expr) (types.Type, error) {
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

func checkImplementsCloser(ctx macro.CallContext, t types.Type) error {
	iface, err := ioCloserInterface(ctx)
	if err != nil {
		return err
	}
	if !types.Implements(t, iface) {
		return fmt.Errorf("payload type must implement io.Closer")
	}
	return nil
}

func ioCloserInterface(ctx macro.CallContext) (*types.Interface, error) {
	for _, imp := range ctx.Package().Imports() {
		if imp.Path() == "io" {
			obj := imp.Scope().Lookup("Closer")
			if obj == nil {
				break
			}
			iface, ok := obj.Type().Underlying().(*types.Interface)
			if ok {
				return iface, nil
			}
		}
	}
	return nil, fmt.Errorf("macro file must import io for With")
}

func isErrorType(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
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
