package try

import (
	"strings"
	"testing"
)

func TestCheckStubMatchesKMessages(t *testing.T) {
	if err := checkStubMatchesK("Try", 2); err == nil || !strings.Contains(err.Error(), "Try2") {
		t.Fatalf("Try vs k=2: %v", err)
	}
	if err := checkStubMatchesK("Try0", 1); err == nil || !strings.Contains(err.Error(), "Try1") {
		t.Fatalf("Try0 vs k=1: %v", err)
	}
	if err := checkStubMatchesK("Try3", 2); err == nil || !strings.Contains(err.Error(), "Try3 requires") {
		t.Fatalf("Try3 vs k=2: %v", err)
	}
}
