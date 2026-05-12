package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSingleFile(t *testing.T) {
	registryPath := filepath.Join("testdata", "load", "valid-single.yaml")

	loaded, diags := Load(registryPath)
	if diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = %v", registryPath, diags)
	}

	if got, want := len(loaded.Context), 3; got != want {
		t.Fatalf("len(loaded.Context) = %d, want %d", got, want)
	}
	if got, want := len(loaded.Events), 2; got != want {
		t.Fatalf("len(loaded.Events) = %d, want %d", got, want)
	}
}

func TestLoadDirectorySortsAndMerges(t *testing.T) {
	registryPath := filepath.Join("testdata", "load", "valid-dir")

	loaded, diags := Load(registryPath)
	if diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = %v", registryPath, diags)
	}

	if got, want := len(loaded.Context), 3; got != want {
		t.Fatalf("len(loaded.Context) = %d, want %d", got, want)
	}
	if got, want := len(loaded.Events), 2; got != want {
		t.Fatalf("len(loaded.Events) = %d, want %d", got, want)
	}
	if got, want := loaded.Events[0].Name, "search.query_submitted"; got != want {
		t.Fatalf("loaded.Events[0].Name = %q, want %q", got, want)
	}
	if got, want := loaded.Events[1].Name, "user.signed_up"; got != want {
		t.Fatalf("loaded.Events[1].Name = %q, want %q", got, want)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	registryPath := filepath.Join("testdata", "load", "unknown-field.yaml")

	_, diags := Load(registryPath)
	if !diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = none, want errors", registryPath)
	}
	if got, want := diags.Error(), "field unknown_field not found"; !strings.Contains(got, want) {
		t.Fatalf("diags = %q, want substring %q", got, want)
	}
}

func TestLoadRejectsDuplicateContextAcrossFiles(t *testing.T) {
	registryPath := filepath.Join("testdata", "load", "duplicate-context")

	_, diags := Load(registryPath)
	if !diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = none, want errors", registryPath)
	}
	if got, want := diags.Error(), "duplicate context field"; !strings.Contains(got, want) {
		t.Fatalf("diags = %q, want substring %q", got, want)
	}
}

func TestLoadRejectsDuplicateEventVersionAcrossFiles(t *testing.T) {
	registryPath := filepath.Join("testdata", "load", "duplicate-event")

	_, diags := Load(registryPath)
	if !diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = none, want errors", registryPath)
	}
	if got, want := diags.Error(), "duplicate event version"; !strings.Contains(got, want) {
		t.Fatalf("diags = %q, want substring %q", got, want)
	}
}

func TestLoadRejectsAdditionalYAMLDocuments(t *testing.T) {
	tempDir := t.TempDir()
	registryPath := filepath.Join(tempDir, "registry.yaml")
	if err := os.WriteFile(registryPath, []byte("openevents: 0.1.0\nnamespace: com.example.product\n---\nnamespace: com.example.other\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", registryPath, err)
	}

	_, diags := Load(registryPath)
	if !diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = none, want errors", registryPath)
	}
	if got, want := diags.Error(), "additional YAML documents are not supported"; !strings.Contains(got, want) {
		t.Fatalf("diags = %q, want substring %q", got, want)
	}
}

func TestLoadDirectoryIgnoresOpenEventsLockFile(t *testing.T) {
	registryPath := filepath.Join("..", "..", "examples", "basic", "openevents.yaml")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", registryPath, err)
	}

	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "openevents.yaml"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(openevents.yaml): %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "openevents.lock.yaml"), []byte("version: 1\ncontext: {}\nevents: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(openevents.lock.yaml): %v", err)
	}

	withLock, diags := Load(tempDir)
	if diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = %v", tempDir, diags)
	}
	if got, want := len(withLock.Events), 2; got != want {
		t.Fatalf("len(withLock.Events) = %d, want %d", got, want)
	}
}

func TestLoadDirectoryLoadsNestedOpenEventsLockFile(t *testing.T) {
	tempDir := t.TempDir()

	root := "openevents: 0.1.0\nnamespace: com.example.product\n"
	if err := os.WriteFile(filepath.Join(tempDir, "openevents.yaml"), []byte(root), 0o644); err != nil {
		t.Fatalf("WriteFile(openevents.yaml): %v", err)
	}

	eventsDir := filepath.Join(tempDir, "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", eventsDir, err)
	}

	nested := "events:\n  nested.lock_event:\n    version: 1\n    status: active\n    owner: eng\n    producer: app\n    destination:\n      queue: analytics\n"
	if err := os.WriteFile(filepath.Join(eventsDir, "openevents.lock.yaml"), []byte(nested), 0o644); err != nil {
		t.Fatalf("WriteFile(events/openevents.lock.yaml): %v", err)
	}

	loaded, diags := Load(tempDir)
	if diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = %v", tempDir, diags)
	}

	if got, want := len(loaded.Events), 1; got != want {
		t.Fatalf("len(loaded.Events) = %d, want %d", got, want)
	}
	if got, want := loaded.Events[0].Name, "nested.lock_event"; got != want {
		t.Fatalf("loaded.Events[0].Name = %q, want %q", got, want)
	}
}

func TestLoadConflictingSingletonDiagnosticLocation(t *testing.T) {
	tempDir := t.TempDir()
	aPath := filepath.Join(tempDir, "a.yaml")
	bPath := filepath.Join(tempDir, "b.yaml")

	if err := os.WriteFile(aPath, []byte("openevents: 0.1.0\nnamespace: old\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", aPath, err)
	}
	if err := os.WriteFile(bPath, []byte("openevents: 0.1.0\nnamespace: new\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", bPath, err)
	}

	_, diags := Load(tempDir)
	if !diags.HasErrors() {
		t.Fatalf("Load(%q) diagnostics = none, want errors", tempDir)
	}

	if got, want := diags[0].Location, bPath+": namespace"; got != want {
		t.Fatalf("diags[0].Location = %q, want %q", got, want)
	}
	if got, want := diags[0].Message, "conflicting value \"new\"; already set to \"old\""; got != want {
		t.Fatalf("diags[0].Message = %q, want %q", got, want)
	}
}
