package with

import (
	"strings"
	"testing"

	"go/types"
)

func TestCheckImplementsCloserRejectsInt(t *testing.T) {
	if err := checkImplementsCloser(types.Typ[types.Int]); err == nil || !strings.Contains(err.Error(), "io.Closer") {
		t.Fatalf("got %v", err)
	}
}

func TestIOCloserInterface(t *testing.T) {
	iface, err := ioCloserInterface()
	if err != nil {
		t.Fatal(err)
	}
	if iface.NumMethods() != 1 || iface.Method(0).Name() != "Close" {
		t.Fatalf("unexpected io.Closer: %v", iface)
	}
}
