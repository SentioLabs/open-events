package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/sentiolabs/open-events/internal/registry/testfx"
)

// buildGenerateRegistry creates a directory-form registry with two domains and
// a codegen block for the given languages.
func buildGenerateRegistry(t *testing.T, languages []string) string {
	t.Helper()
	b := testfx.New().
		Namespace("com.example.test").
		Package("github.com/example/test/events", "example_test.events").
		Owner("eng", "eng@example.com")
	for _, lang := range languages {
		b = b.Language(lang)
	}
	return b.
		Domain("user").
		Owner("eng").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("signup").Done().
		Done().
		Domain("search").
		Owner("eng").
		Context("session_id", registry.FieldTypeUUID, false, registry.PIIPseudonymous).
		Action([]string{"query"}, "submitted").Version(1).Status("active").Description("query submitted").Done().
		Done().
		Write(t)
}

// TestGenerateUnifiedHappyPathGo tests that `generate <registry>` with go in
// codegen.languages emits Go bindings under <registry>/.openevents/proto/ and gen/go/.
func TestGenerateUnifiedHappyPathGo(t *testing.T) {
	registryPath := buildGenerateRegistry(t, []string{"go"})

	// Lock must exist before generate can run.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	lockCmd := NewRootCommand(&stdout, &stderr)
	lockCmd.SetArgs([]string{"lock", "update", registryPath})
	if err := lockCmd.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v, stderr = %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", registryPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "ok: generated bindings for") {
		t.Fatalf("stdout = %q, want containing \"ok: generated bindings for\"", out)
	}
	if !strings.Contains(out, "go") {
		t.Fatalf("stdout = %q, want containing \"go\"", out)
	}

	// Verify Go output files exist.
	genDir := filepath.Join(registryPath, "gen", "go")
	for _, domain := range []string{"user", "search"} {
		domDir := filepath.Join(genDir, domain)
		for _, file := range []string{"event_names.go", "context.go", "events.go"} {
			if _, err := os.Stat(filepath.Join(domDir, file)); err != nil {
				t.Errorf("expected file %s/%s to exist: %v", domDir, file, err)
			}
		}
	}
}

// TestGenerateUnifiedHappyPathPython tests that `generate <registry>` with python in
// codegen.languages emits Python bindings.
func TestGenerateUnifiedHappyPathPython(t *testing.T) {
	registryPath := buildGenerateRegistry(t, []string{"python"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	lockCmd := NewRootCommand(&stdout, &stderr)
	lockCmd.SetArgs([]string{"lock", "update", registryPath})
	if err := lockCmd.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v, stderr = %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", registryPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "ok: generated bindings for") {
		t.Fatalf("stdout = %q, want containing \"ok: generated bindings for\"", out)
	}
	if !strings.Contains(out, "python") {
		t.Fatalf("stdout = %q, want containing \"python\"", out)
	}

	// Verify Python output files exist.
	genDir := filepath.Join(registryPath, "gen", "python")
	for _, domain := range []string{"user", "search"} {
		for _, file := range []string{
			filepath.Join("event_names", domain+".py"),
			filepath.Join("context", domain+".py"),
		} {
			if _, err := os.Stat(filepath.Join(genDir, file)); err != nil {
				t.Errorf("expected file %s to exist: %v", filepath.Join(genDir, file), err)
			}
		}
	}
}

// TestGenerateMissingLanguageConfigUsesDefaults verifies that when a language is
// listed in codegen.languages but has no entry in codegen.configs, defaults are used.
func TestGenerateMissingLanguageConfigUsesDefaults(t *testing.T) {
	// Build a registry with "go" in languages but NO configs block.
	registryPath := buildGenerateRegistry(t, []string{"go"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	lockCmd := NewRootCommand(&stdout, &stderr)
	lockCmd.SetArgs([]string{"lock", "update", registryPath})
	if err := lockCmd.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", registryPath})

	// Must succeed — missing config block means use defaults.
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
	}

	// Default Go out is ./gen/go, so files should appear there.
	genDir := filepath.Join(registryPath, "gen", "go")
	if _, err := os.Stat(genDir); err != nil {
		t.Fatalf("expected default gen/go dir to exist: %v", err)
	}
}

// TestGenerateUnknownLanguageReturnsError verifies that an unknown language name
// in codegen.languages causes generate to return an error.
func TestGenerateUnknownLanguageReturnsError(t *testing.T) {
	registryPath := buildGenerateRegistry(t, []string{"ruby"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	lockCmd := NewRootCommand(&stdout, &stderr)
	lockCmd.SetArgs([]string{"lock", "update", registryPath})
	if err := lockCmd.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", registryPath})

	err := cmd.Execute()
	if !errors.Is(err, errGenerationFailed) {
		t.Fatalf("Execute() error = %v, want errGenerationFailed", err)
	}
	if got := stderr.String(); !strings.Contains(got, "ruby") {
		t.Fatalf("stderr = %q, want containing unsupported language mention", got)
	}
}

// TestGenerateEmptyLanguagesSkipsCodegen verifies that when codegen.languages is
// empty, generate succeeds (no codegen runs) and still emits proto.
func TestGenerateEmptyLanguagesSkipsCodegen(t *testing.T) {
	registryPath := buildGenerateRegistry(t, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	lockCmd := NewRootCommand(&stdout, &stderr)
	lockCmd.SetArgs([]string{"lock", "update", registryPath})
	if err := lockCmd.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", registryPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
	}
}

// TestGenerateProtoUnsupportedWithoutLockWritesErrOut verifies that generate
// (top-level) fails gracefully when no lock file exists.
func TestGenerateProtoUnsupportedWithoutLockWritesErrOut(t *testing.T) {
	registryPath := buildGenerateRegistry(t, []string{"go"})
	// No lock update — lock file is missing.

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", registryPath})

	err := cmd.Execute()
	if !errors.Is(err, errGenerationFailed) {
		t.Fatalf("Execute() error = %v, want errGenerationFailed", err)
	}
	if got := stderr.String(); !strings.Contains(got, "openevents.lock.yaml") {
		t.Fatalf("stderr = %q, want lock file error", got)
	}
}

// TestGenerateHiddenProtoSubcommandStillWorks verifies that the hidden `generate proto`
// subcommand is still registered and functional.
func TestGenerateHiddenProtoSubcommandStillWorks(t *testing.T) {
	registryPath := buildGenerateRegistry(t, []string{"go"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	lockCmd := NewRootCommand(&stdout, &stderr)
	lockCmd.SetArgs([]string{"lock", "update", registryPath})
	if err := lockCmd.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v", err)
	}

	outDir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", "proto", registryPath, outDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "ok: generated proto schema in") {
		t.Fatalf("stdout = %q, want proto success message", got)
	}
}
