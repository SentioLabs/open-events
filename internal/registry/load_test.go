package registry

import (
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
	if got, want := len(loaded.Events), 1; got != want {
		t.Fatalf("len(loaded.Events) = %d, want %d", got, want)
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
