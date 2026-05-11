package integration_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestValidateDemoRegistry(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/openevents", "validate", "../../examples/demo")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate demo failed: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := "ok: registry valid (3 events, 4 context fields)"
	if got != want {
		t.Fatalf("validate output = %q, want %q", got, want)
	}
}
