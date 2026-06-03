package wirejson_test

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/arcane-craft/go-macro-contrib/wirejson"
	"github.com/arcane-craft/go-macro/macro/mactest"
)

func TestWireJSONExpand(t *testing.T) {
	result, err := mactest.ExpandDecl(wirejson.WireJSONExpand, "wire-json", `
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
	if len(result.Fields) != 2 {
		t.Fatalf("fields: %d", len(result.Fields))
	}
	if tag := fieldTag(result.Fields[0]); tag != `json:"id"` {
		t.Fatalf("ID tag: %q", tag)
	}
	if tag := fieldTag(result.Fields[1]); tag != `json:"name"` {
		t.Fatalf("Name tag: %q", tag)
	}
}

func TestWireJSONExpandOmitEmpty(t *testing.T) {
	result, err := mactest.ExpandDecl(wirejson.WireJSONExpand, "wire-json", `
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
	if tag := fieldTag(result.Fields[0]); tag != `json:"id"` {
		t.Fatalf("ID tag: %q", tag)
	}
	if tag := fieldTag(result.Fields[1]); tag != `json:"role,omitempty"` {
		t.Fatalf("Role tag: %q", tag)
	}
}

func TestWireJSONExpandPreservesMethods(t *testing.T) {
	result, err := mactest.ExpandDecl(wirejson.WireJSONExpand, "wire-json", `
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
	found := false
	for _, m := range result.Methods {
		if m.Name.Name == "Validate" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected Validate method preserved")
	}
}

func TestWireJSONExpandConflictingTag(t *testing.T) {
	_, err := mactest.ExpandDecl(wirejson.WireJSONExpand, "wire-json", `
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
