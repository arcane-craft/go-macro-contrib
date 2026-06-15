package derive

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
)

// Derive is a declaration macro marker. Embed Derive[fmt.Stringer] anonymously in a struct.
//
//macro: derive
type Derive[T any] struct{}

// String is a type-check stub promoted to embedders before expand. Do not call at runtime.
func (Derive[T]) String() string {
	panic("Derive.String is a macro stub and must not be called at runtime")
}

//macro: derive
// DeriveExpander removes the embed and generates String() unless the target already has one.
var DeriveExpander = macro.SyntaxCase(macro.Clause{
	Pattern:   `type $item struct { Derive[$iface] $field ... }`,
	Transform: deriveTransform,
})

func deriveTransform(ctx macro.Context, site macro.Syntax, binds macro.Bindings) (macro.Syntax, error) {
	if err := validateStringerTypeArg(ctx, site, binds); err != nil {
		return nil, err
	}
	itemSyn, ok := binds.Get("item")
	if !ok {
		return nil, fmt.Errorf("derive: missing item binding")
	}
	itemTS, ok := itemSyn.Underlying().(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("derive: item is not TypeSpec")
	}
	fieldElems, ok := binds.Elems("field")
	if !ok {
		return nil, fmt.Errorf("derive: missing field bindings")
	}
	fields := make([]*ast.Field, 0, len(fieldElems))
	var parts []string
	for _, fe := range fieldElems {
		f, ok := fe.Underlying().(*ast.Field)
		if !ok {
			return nil, fmt.Errorf("derive: field binding is not *ast.Field")
		}
		if len(f.Names) == 0 {
			continue
		}
		fields = append(fields, f)
		for _, n := range f.Names {
			parts = append(parts, fmt.Sprintf("%s: %%v", n.Name))
		}
	}
	target := itemTS.Name.Name
	newTS := &ast.TypeSpec{
		Name: itemTS.Name,
		Type: &ast.StructType{Fields: &ast.FieldList{List: fields}},
	}
	decls := []ast.Decl{&ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{newTS}}}

	if shouldGenerateString(ctx, site, binds) {
		format := strings.Join(parts, ", ")
		call := buildSprintfCall(format, fieldSelectorExprs(target, fields))
		fd := &ast.FuncDecl{
			Name: ast.NewIdent("String"),
			Recv: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(target)}}},
			Type: &ast.FuncType{
				Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("string")}}},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}},
		}
		decls = append(decls, fd)
	}
	return macro.WrapDecls(decls), nil
}

func validateStringerTypeArg(ctx macro.Context, site macro.Syntax, binds macro.Bindings) error {
	ifaceSyn, ok := binds.Get("iface")
	if !ok {
		return fmt.Errorf("derive: type argument must be fmt.Stringer")
	}
	ifaceExpr, ok := ifaceSyn.Underlying().(ast.Expr)
	if !ok {
		return fmt.Errorf("derive: iface is not an expression")
	}
	got := types.Unalias(ctx.Types().TypeOf(ifaceExpr))
	if got == nil {
		return fmt.Errorf("derive: cannot resolve type argument")
	}
	want := fmtStringerType()
	if want == nil {
		return fmt.Errorf("derive: cannot resolve fmt.Stringer")
	}
	if !types.Identical(got, want) {
		if named, ok := got.(*types.Named); !ok || named.Obj().Pkg() == nil ||
			named.Obj().Pkg().Path() != "fmt" || named.Obj().Name() != "Stringer" {
			return fmt.Errorf("derive: type argument must be fmt.Stringer")
		}
	}
	return nil
}

func fmtStringerType() types.Type {
	pkg, err := importer.Default().Import("fmt")
	if err != nil {
		return nil
	}
	if obj := pkg.Scope().Lookup("Stringer"); obj != nil {
		return obj.Type()
	}
	return nil
}

func buildSprintfCall(format string, fieldExprs []ast.Expr) ast.Expr {
	args := []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"` + format + `"`}}
	args = append(args, fieldExprs...)
	return &ast.CallExpr{
		Fun:  &ast.Ident{Name: "fmt.Sprintf"},
		Args: args,
	}
}

func shouldGenerateString(ctx macro.Context, site macro.Syntax, binds macro.Bindings) bool {
	itemSyn, ok := binds.Get("item")
	if !ok {
		return true
	}
	itemTS, ok := itemSyn.Underlying().(*ast.TypeSpec)
	if !ok {
		return true
	}
	typeName := itemTS.Name.Name
	deriveAnchor, ok := site.Underlying().(*ast.Field)
	if !ok {
		return true
	}
	fc, ok := site.(macro.FileCarrier)
	if !ok {
		return true
	}
	file := fc.ExpansionFile()
	if targetDeclaresString(file, typeName) {
		return false
	}
	st, ok := itemTS.Type.(*ast.StructType)
	if !ok || st.Fields == nil {
		return true
	}
	if otherEmbedPromotesString(st.Fields.List, deriveAnchor, ctx.Types()) {
		return false
	}
	named := targetNamedType(ctx, itemTS)
	if named == nil {
		return true
	}
	return deriveStubOnlyPromotesString(named, deriveAnchor, st, ctx.Types())
}

func targetDeclaresString(file *ast.File, typeName string) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "String" || fd.Recv == nil || len(fd.Recv.List) == 0 {
			return true
		}
		if recvMatchesTypeName(fd.Recv.List[0].Type, typeName) {
			found = true
			return false
		}
		return true
	})
	return found
}

func recvMatchesTypeName(recvType ast.Expr, typeName string) bool {
	switch t := recvType.(type) {
	case *ast.Ident:
		return t.Name == typeName
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name == typeName
		}
	}
	return false
}

func otherEmbedPromotesString(structFields []*ast.Field, deriveAnchor *ast.Field, info *types.Info) bool {
	for _, f := range structFields {
		if len(f.Names) > 0 {
			continue
		}
		if f == deriveAnchor {
			continue
		}
		typ := info.TypeOf(f.Type)
		if typ == nil {
			continue
		}
		if types.NewMethodSet(types.NewPointer(typ)).Lookup(nil, "String") != nil {
			return true
		}
	}
	return false
}

func deriveStubOnlyPromotesString(namedType types.Type, deriveAnchor *ast.Field, st *ast.StructType, info *types.Info) bool {
	ms := types.NewMethodSet(types.NewPointer(namedType))
	for i := 0; i < ms.Len(); i++ {
		sel := ms.At(i)
		if sel.Obj().Name() != "String" {
			continue
		}
		if !stringSelectionIsDeriveStub(deriveAnchor, st, sel) {
			return false
		}
	}
	return true
}

func stringSelectionIsDeriveStub(deriveAnchor *ast.Field, st *ast.StructType, sel *types.Selection) bool {
	if sel.Kind() != types.MethodVal {
		return false
	}
	fn, ok := sel.Obj().(*types.Func)
	if !ok {
		return false
	}
	if methodReceiverNamedType(fn) != "Derive" {
		return false
	}
	idx := sel.Index()
	if len(idx) == 0 || st.Fields == nil {
		return false
	}
	if idx[0] < 0 || idx[0] >= len(st.Fields.List) {
		return false
	}
	return st.Fields.List[idx[0]] == deriveAnchor
}

func targetNamedType(ctx macro.Context, target *ast.TypeSpec) types.Type {
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
