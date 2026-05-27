package inline

import (
	"strings"
	"testing"
)

func TestCheckStubMatchesN(t *testing.T) {
	if err := checkStubMatchesN("Inline", 2); err == nil || !strings.Contains(err.Error(), "Inline2") {
		t.Fatalf("Inline vs n=2: %v", err)
	}
	if err := checkStubMatchesN("Inline2", 1); err == nil || !strings.Contains(err.Error(), "Inline ") {
		t.Fatalf("Inline2 vs n=1: %v", err)
	}
	if err := checkStubMatchesN("Inline0", 0); err != nil {
		t.Fatalf("Inline0 vs n=0: %v", err)
	}
}
