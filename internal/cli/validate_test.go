package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateCommandWithValidExample(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"validate", "../../examples/basic"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}

	want := "ok: registry valid (2 events, 3 context fields)"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want containing %q", got, want)
	}
}

func TestValidateCommandWithInvalidPath(t *testing.T) {
	badPath := "../../examples/does-not-exist"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"validate", badPath})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want non-nil")
	}

	if got := stderr.String(); !strings.Contains(got, badPath) {
		t.Fatalf("stderr = %q, want containing %q", got, badPath)
	}
}
