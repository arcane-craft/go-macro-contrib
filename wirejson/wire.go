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

// WireJSONExpand sets json tags and removes the embed.
func WireJSONExpand(ctx macro.DeclContext, site macro.DeclSite) (macro.DeclExpandResult, error) {
	st, ok := site.Target.Type.(*ast.StructType)
	if !ok || st.Fields == nil {
		return macro.DeclExpandResult{}, fmt.Errorf("wire-json: target is not a struct")
	}
	omit := parseOmitEmpty(site.MacroTag)
	var fields []*ast.Field
	for _, f := range st.Fields.List {
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
			return macro.DeclExpandResult{}, macro.ErrorAt(ctx.FileSet(), f.Pos(),
				"wire-json: field %s already has conflicting json tag %q", name, existing)
		}
		clone.Tag = &ast.BasicLit{Kind: token.STRING, Value: "`" + tag + "`"}
		fields = append(fields, &clone)
	}
	methods := append([]*ast.FuncDecl{}, ctx.TargetMethods()...)
	return macro.DeclExpandResult{Fields: fields, Methods: methods}, nil
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
