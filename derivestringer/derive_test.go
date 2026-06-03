package derivestringer_test

import (
	"testing"

	"github.com/arcane-craft/go-macro-contrib/derivestringer"
	"github.com/arcane-craft/go-macro/macro/mactest"
)

func TestDeriveStringerExpand(t *testing.T) {
	result, err := mactest.ExpandDecl(derivestringer.DeriveStringerExpand, "derive-stringer", `
type DeriveStringer struct{}

type Item struct {
	DeriveStringer
	A string
	B int
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fields) != 2 {
		t.Fatalf("fields: %d", len(result.Fields))
	}
	foundString := false
	for _, m := range result.Methods {
		if m.Name.Name == "String" {
			foundString = true
		}
	}
	if !foundString {
		t.Fatal("expected String method")
	}
}
