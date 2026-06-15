package wirejson_test

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/arcane-craft/go-macro/macro/mactest"
	"github.com/arcane-craft/go-macro-contrib/wirejson"
)

func TestWireJSONExpand(t *testing.T) {
	out, err := mactest.Expand(wirejson.WireJSONExpander, "wire-json", `
type WireJSON struct{}

type User struct {
	WireJSON
	ID int64
	Name string
}
`)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := out.ToDecls()
	if err != nil {
		t.Fatal(err)
	}
	if len(decls) != 1 {
		t.Fatalf("decls: %d", len(decls))
	}
	ts, ok := decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	if !ok {
		t.Fatalf("decl: %T", decls[0])
	}
	st := ts.Type.(*ast.StructType)
	if len(st.Fields.List) != 2 {
		t.Fatalf("fields: %d", len(st.Fields.List))
	}
	if tag := fieldTag(st.Fields.List[0]); tag != `json:"id"` {
		t.Fatalf("ID tag: %q", tag)
	}
	if tag := fieldTag(st.Fields.List[1]); tag != `json:"name"` {
		t.Fatalf("Name tag: %q", tag)
	}
}

func TestWireJSONExpandOmitEmpty(t *testing.T) {
	out, err := mactest.Expand(wirejson.WireJSONExpander, "wire-json", `
type WireJSON struct{}

type Profile struct {
	WireJSON ` + "`macro:\"omitempty=Role\"`" + `
	ID int64
	Role string
}
`)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := out.ToDecls()
	if err != nil {
		t.Fatal(err)
	}
	ts := decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	st := ts.Type.(*ast.StructType)
	if tag := fieldTag(st.Fields.List[0]); tag != `json:"id"` {
		t.Fatalf("ID tag: %q", tag)
	}
	if tag := fieldTag(st.Fields.List[1]); tag != `json:"role,omitempty"` {
		t.Fatalf("Role tag: %q", tag)
	}
}

func TestWireJSONExpandOutIsTypeSpecOnly(t *testing.T) {
	out, err := mactest.Expand(wirejson.WireJSONExpander, "wire-json", `
type WireJSON struct{}

type User struct {
	WireJSON
	Name string
}

func (User) Validate() error { return nil }
`)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := out.ToDecls()
	if err != nil {
		t.Fatal(err)
	}
	if len(decls) != 1 {
		t.Fatalf("expected single TypeSpec decl, got %d", len(decls))
	}
	if _, ok := decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec); !ok {
		t.Fatalf("expected TypeSpec, got %T", decls[0])
	}
}

func TestWireJSONExpandConflictingTag(t *testing.T) {
	_, err := mactest.Expand(wirejson.WireJSONExpander, "wire-json", `
type WireJSON struct{}

type User struct {
	WireJSON
	Name string ` + "`json:\"display_name\"`" + `
}
`)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "conflicting json tag") {
		t.Fatalf("err: %v", err)
	}
}

func fieldTag(f *ast.Field) string {
	if f == nil || f.Tag == nil {
		return ""
	}
	return strings.Trim(f.Tag.Value, "`")
}
