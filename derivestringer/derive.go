package derivestringer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
	"github.com/arcane-craft/go-macro/macro/quote"
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
		call := buildSprintfCall(format, fieldSelectorExprs(target, fields))
		decls, err := quote.Decls(`func (#recv) String() string {
	return #call
}`, map[string]any{
			"recv": target,
			"call": call,
		})
		if err != nil {
			return macro.DeclExpandResult{}, macro.ErrorAt(ctx.FileSet(), site.Target.Pos(), "%v", err)
		}
		if len(decls) != 1 {
			return macro.DeclExpandResult{}, fmt.Errorf("derive-stringer: expected one method decl, got %d", len(decls))
		}
		fd, ok := decls[0].(*ast.FuncDecl)
		if !ok {
			return macro.DeclExpandResult{}, fmt.Errorf("derive-stringer: expected *ast.FuncDecl")
		}
		methods = append(methods, fd)
	}
	return macro.DeclExpandResult{Fields: fields, Methods: methods}, nil
}

func buildSprintfCall(format string, fieldExprs []ast.Expr) ast.Expr {
	args := []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"` + format + `"`}}
	args = append(args, fieldExprs...)
	return &ast.CallExpr{
		Fun:  &ast.Ident{Name: "fmt.Sprintf"},
		Args: args,
	}
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
