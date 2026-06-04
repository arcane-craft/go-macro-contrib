package derivestringer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
)

// DeriveStringer is a declaration macro marker. Embed anonymously in a struct.
//
//macro: derive-stringer
type DeriveStringer struct{}

// String is a type-check stub promoted to embedders before expand. Do not call at runtime.
func (DeriveStringer) String() string {
	panic("DeriveStringer.String is a macro stub and must not be called at runtime")
}

//macro: derive-stringer

// DeriveStringerExpand removes the embed and generates String() unless Target already has one.
func DeriveStringerExpand(ctx macro.DeclContext, site macro.DeclSite) (macro.DeclExpandResult, error) {
	target := site.Target.Name.Name
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
	methods := append([]*ast.FuncDecl{}, ctx.TargetMethods()...)
	if shouldGenerateString(ctx, site) {
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
		methods = append(methods, &ast.FuncDecl{
			Name: ast.NewIdent("String"),
			Recv: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(target)}}},
			Type: &ast.FuncType{
				Params:  &ast.FieldList{},
				Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("string")}}},
			},
			Body: body,
		})
	}
	return macro.DeclExpandResult{Fields: fields, Methods: methods}, nil
}

// shouldGenerateString reports whether to synthesize func (T) String().
// User-declared (T) String() or String() promoted from a non-marker embed wins.
func shouldGenerateString(ctx macro.DeclContext, site macro.DeclSite) bool {
	for _, fn := range ctx.TargetMethods() {
		if fn.Name.Name == "String" {
			return false
		}
	}
	typ := targetNamedType(ctx, site.Target)
	if typ == nil {
		return true
	}
	sel := types.NewMethodSet(types.NewPointer(typ)).Lookup(nil, "String")
	if sel == nil {
		return true
	}
	return stringSelectionIsDeriveStringerStub(site, sel)
}

func targetNamedType(ctx macro.DeclContext, target *ast.TypeSpec) types.Type {
	obj := ctx.Types().Defs[target.Name]
	if obj == nil {
		return nil
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil
	}
	return tn.Type()
}

func stringSelectionIsDeriveStringerStub(site macro.DeclSite, sel *types.Selection) bool {
	if sel.Kind() != types.MethodVal {
		return false
	}
	fn, ok := sel.Obj().(*types.Func)
	if !ok {
		return false
	}
	if methodReceiverNamedType(fn) != site.MarkerTypeName {
		return false
	}
	idx := sel.Index()
	return len(idx) > 0 && idx[0] == site.EmbedIndex
}

func methodReceiverNamedType(fn *types.Func) string {
	recv := fn.Type().(*types.Signature).Recv().Type()
	for {
		switch t := recv.(type) {
		case *types.Pointer:
			recv = t.Elem()
		case *types.Named:
			return t.Obj().Name()
		default:
			return ""
		}
	}
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
