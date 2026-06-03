package derivestringer

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
)

// DeriveStringer is a declaration macro marker. Embed anonymously in a struct.
//
//macro: derive-stringer
type DeriveStringer struct{}

//macro: derive-stringer

// DeriveStringerExpand generates String() and removes the embed.
func DeriveStringerExpand(ctx macro.DeclContext, site macro.DeclSite) (macro.DeclExpandResult, error) {
	target := site.Target.Name.Name
	for _, fn := range ctx.TargetMethods() {
		if fn.Name.Name == "String" {
			return macro.DeclExpandResult{}, macro.ErrorAt(ctx.FileSet(), ctx.MacroPos(),
				"%s already has String() method", target)
		}
	}
	st, ok := site.Target.Type.(*ast.StructType)
	if !ok || st.Fields == nil {
		return macro.DeclExpandResult{}, fmt.Errorf("derive-stringer: target is not a struct")
	}
	var fields []*ast.Field
	var parts []string
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue
		}
		fields = append(fields, f)
		for _, n := range f.Names {
			parts = append(parts, fmt.Sprintf("%s: %%v", n.Name))
		}
	}
	format := strings.Join(parts, ", ")
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.Ident{Name: "fmt.Sprintf"},
						Args: append(
							[]ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"` + format + `"`}},
							fieldSelectorExprs(target, fields)...,
						),
					},
				},
			},
		},
	}
	stringFn := &ast.FuncDecl{
		Name: ast.NewIdent("String"),
		Recv: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(target)}}},
		Type: &ast.FuncType{
			Params:  &ast.FieldList{},
			Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("string")}}},
		},
		Body: body,
	}
	methods := append([]*ast.FuncDecl{}, ctx.TargetMethods()...)
	methods = append(methods, stringFn)
	return macro.DeclExpandResult{Fields: fields, Methods: methods}, nil
}

func fieldSelectorExprs(recv string, fields []*ast.Field) []ast.Expr {
	var out []ast.Expr
	for _, f := range fields {
		for _, n := range f.Names {
			out = append(out, &ast.SelectorExpr{
				X:   &ast.Ident{Name: recv},
				Sel: n,
			})
		}
	}
	return out
}
