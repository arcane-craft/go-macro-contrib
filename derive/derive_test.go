package derive_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/arcane-craft/go-macro-contrib/derive"
	"github.com/arcane-craft/go-macro/macro/mactest"
)

const deriveMarkerStub = `
import "fmt"

type Derive[T any] struct{}

func (Derive[T]) String() string { panic("stub") }
`

func TestDeriveStubPromotesStringer(t *testing.T) {
	var _ fmt.Stringer = struct {
		derive.Derive[fmt.Stringer]
		A string
	}{}
}

func TestDeriveExpand(t *testing.T) {
	result, err := mactest.ExpandDecl(derive.DeriveExpand, "derive", deriveMarkerStub+`
type Item struct {
	Derive[fmt.Stringer]
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

func TestDeriveSkipsWhenTargetDeclaresString(t *testing.T) {
	result, err := mactest.ExpandDecl(derive.DeriveExpand, "derive", deriveMarkerStub+`
func (Item) String() string { return "custom" }

type Item struct {
	Derive[fmt.Stringer]
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

func TestDeriveSkipsWhenOtherEmbedPromotesString(t *testing.T) {
	result, err := mactest.ExpandDecl(derive.DeriveExpand, "derive", deriveMarkerStub+`
type Helper struct{}

func (Helper) String() string { return "from-helper" }

type Item struct {
	Derive[fmt.Stringer]
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

func TestDeriveRejectsInvalidTypeArg(t *testing.T) {
	cases := []struct {
		name   string
		snippet string
	}{
		{
			name: "int",
			snippet: `
type Derive[T any] struct{}

func (Derive[T]) String() string { panic("stub") }

type Item struct {
	Derive[int]
	A string
}
`,
		},
		{
			name: "any",
			snippet: `
type Derive[T any] struct{}

func (Derive[T]) String() string { panic("stub") }

type Item struct {
	Derive[any]
	A string
}
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mactest.ExpandDecl(derive.DeriveExpand, "derive", tc.snippet)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "fmt.Stringer") {
				t.Fatalf("error %q should mention fmt.Stringer", err)
			}
		})
	}
}
