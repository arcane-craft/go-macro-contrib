package wirejson

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/arcane-craft/go-macro/macro"
)

// WireJSON is a declaration macro marker for json struct tags.
//
//macro: wire-json
type WireJSON struct{}

//macro: wire-json
// WireJSONExpander sets json tags and removes the embed.
var WireJSONExpander = macro.SyntaxCase(macro.Clause{
	Pattern:   `type $item struct { WireJSON $field ... }`,
	Transform: wireJSONTransform,
})

func wireJSONTransform(ctx macro.Context, site macro.Syntax, binds macro.Bindings) (macro.Syntax, error) {
	itemSyn, ok := binds.Get("item")
	if !ok {
		return nil, fmt.Errorf("wire-json: missing item binding")
	}
	itemTS, ok := itemSyn.Underlying().(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("wire-json: item is not TypeSpec")
	}
	embedField, ok := site.Underlying().(*ast.Field)
	if !ok {
		return nil, fmt.Errorf("wire-json: expected embed field site")
	}
	omit := parseOmitEmpty(macro.ParseMacroTag(embedField.Tag))

	fieldElems, ok := binds.Elems("field")
	if !ok {
		return nil, fmt.Errorf("wire-json: missing field bindings")
	}
	fields := make([]*ast.Field, 0, len(fieldElems))
	for _, fe := range fieldElems {
		f, ok := fe.Underlying().(*ast.Field)
		if !ok {
			return nil, fmt.Errorf("wire-json: field binding is not *ast.Field")
		}
		if len(f.Names) == 0 {
			continue
		}
		clone := *f
		name := f.Names[0].Name
		tag := jsonTagForField(name, omit[name])
		if tag == "" {
			fields = append(fields, f)
			continue
		}
		if existing, _ := parseStructTag(f.Tag); existing != "" && existing != tag {
			return nil, fmt.Errorf("wire-json: field %s already has conflicting json tag %q", name, existing)
		}
		clone.Tag = &ast.BasicLit{Kind: token.STRING, Value: "`" + tag + "`"}
		fields = append(fields, &clone)
	}
	newTS := &ast.TypeSpec{
		Name: itemTS.Name,
		Type: &ast.StructType{Fields: &ast.FieldList{List: fields}},
	}
	return macro.WrapDecls([]ast.Decl{&ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{newTS},
	}}), nil
}

func parseOmitEmpty(tag macro.MacroTag) map[string]bool {
	out := make(map[string]bool)
	if tag == nil {
		return out
	}
	v, ok := tag["omitempty"]
	if !ok || v == "" {
		return out
	}
	for _, name := range strings.Split(v, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = true
		}
	}
	return out
}

func jsonTagForField(name string, omit bool) string {
	key := strings.ToLower(name)
	if omit {
		return fmt.Sprintf(`json:"%s,omitempty"`, key)
	}
	return fmt.Sprintf(`json:"%s"`, key)
}

func parseStructTag(lit *ast.BasicLit) (jsonVal string, ok bool) {
	if lit == nil {
		return "", false
	}
	s := strings.Trim(lit.Value, "`")
	for _, part := range strings.Fields(s) {
		if strings.HasPrefix(part, `json:"`) {
			return part, true
		}
	}
	return "", false
}
