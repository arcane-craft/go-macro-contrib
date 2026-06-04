package derivestringer_test

import (
	"fmt"
	"testing"

	"github.com/arcane-craft/go-macro-contrib/derivestringer"
	"github.com/arcane-craft/go-macro/macro/mactest"
)

func TestDeriveStringerStubPromotesStringer(t *testing.T) {
	var _ fmt.Stringer = struct {
		derivestringer.DeriveStringer
		A string
	}{}
}

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

func TestDeriveStringerSkipsWhenTargetDeclaresString(t *testing.T) {
	result, err := mactest.ExpandDecl(derivestringer.DeriveStringerExpand, "derive-stringer", `
type DeriveStringer struct{}

func (Item) String() string { return "custom" }

type Item struct {
	DeriveStringer
	A string
}
`)
	if err != nil {
		t.Fatal(err)
	}
	var stringMethods int
	for _, m := range result.Methods {
		if m.Name.Name == "String" {
			stringMethods++
		}
	}
	if stringMethods != 1 {
		t.Fatalf("expected exactly one String method, got %d", stringMethods)
	}
}

func TestDeriveStringerSkipsWhenOtherEmbedPromotesString(t *testing.T) {
	result, err := mactest.ExpandDecl(derivestringer.DeriveStringerExpand, "derive-stringer", `
type DeriveStringer struct{}

type Helper struct{}

func (Helper) String() string { return "from-helper" }

type Item struct {
	DeriveStringer
	Helper
}
`)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range result.Methods {
		if m.Name.Name == "String" {
			t.Fatal("expected no generated String method when Helper promotes String")
		}
	}
}
