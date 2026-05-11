package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestGenerateCommandUnsupportedTargetWritesErrOut(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", "ruby", "../../examples/basic", t.TempDir()})

	err := cmd.Execute()
	if !errors.Is(err, errGenerationFailed) {
		t.Fatalf("Execute() error = %v, want errGenerationFailed", err)
	}
	if got := stderr.String(); !strings.Contains(got, "unsupported generation target \"ruby\"") {
		t.Fatalf("stderr = %q, want unsupported generation target message", got)
	}
}
