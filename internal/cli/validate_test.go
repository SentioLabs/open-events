package cli

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestValidateCommandIgnoresLockFileInDirectory(t *testing.T) {
	registryPath := t.TempDir()
	content, err := os.ReadFile("../../examples/basic/openevents.yaml")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryPath, "openevents.yaml"), content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	update := NewRootCommand(&stdout, &stderr)
	update.SetArgs([]string{"lock", "update", registryPath})
	if err := update.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	validate := NewRootCommand(&stdout, &stderr)
	validate.SetArgs([]string{"validate", registryPath})
	if err := validate.Execute(); err != nil {
		t.Fatalf("validate Execute() error = %v", err)
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
