package inline

import (
	"go/ast"

	"github.com/arcane-craft/go-macro/macro"
)

//macro: syntax-inline

// InlineExpand expands Inline(...) to its sole argument expression.
func InlineExpand(ctx macro.Context, call *ast.CallExpr) (macro.ExpandResult, error) {
	switch ctx.Site() {
	case macro.SiteExpr:
	default:
		return macro.ExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(),
			"Inline only allowed in expression position")
	}
	if len(call.Args) != 1 {
		return macro.ExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(),
			"Inline expects exactly one argument")
	}
	return macro.ExpandResult{Expr: call.Args[0]}, nil
}
