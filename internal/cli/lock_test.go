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
	"github.com/sentiolabs/open-events/internal/schemair"
)

// buildLockRegistry builds a minimal directory-form registry for lock tests.
func buildLockRegistry(t *testing.T) string {
	t.Helper()
	return testfx.New().
		Namespace("com.example.product").
		Package("github.com/example/product/events", "example_product.events").
		Owner("data-platform", "data-platform@example.com").
		Domain("user").
		Owner("data-platform").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signed_up").Version(1).Status("active").Description("User signed up.").Done().
		Done().
		Write(t)
}

func TestLockUpdateWritesLockFile(t *testing.T) {
	registryPath := buildLockRegistry(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"lock", "update", registryPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
	}
	lockPath := filepath.Join(registryPath, "openevents.lock.yaml")
	lock, err := readLockFile(lockPath)
	if err != nil {
		t.Fatalf("readLockFile(%q) error = %v", lockPath, err)
	}
	if got, want := len(lock.Events), 1; got != want {
		t.Fatalf("len(lock.Events) = %d, want %d", got, want)
	}
	if got, want := len(lock.Domains), 1; got != want {
		t.Fatalf("len(lock.Domains) = %d, want %d", got, want)
	}
}

func TestLockCheckRejectsStaleLock(t *testing.T) {
	registryPath := buildLockRegistry(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	update := NewRootCommand(&stdout, &stderr)
	update.SetArgs([]string{"lock", "update", registryPath})
	if err := update.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v, stderr = %s", err, stderr.String())
	}

	lockPath := filepath.Join(registryPath, "openevents.lock.yaml")
	lock, err := readLockFile(lockPath)
	if err != nil {
		t.Fatalf("readLockFile(%q) error = %v", lockPath, err)
	}
	// Tamper: clear all events to simulate a stale lock.
	lock.Events = map[string]schemair.LockedEvent{}
	if err := writeLockFile(lockPath, lock); err != nil {
		t.Fatalf("writeLockFile(%q) error = %v", lockPath, err)
	}

	stdout.Reset()
	stderr.Reset()
	check := NewRootCommand(&stdout, &stderr)
	check.SetArgs([]string{"lock", "check", registryPath})

	err = check.Execute()
	if !errors.Is(err, errLockFailed) {
		t.Fatalf("Execute() error = %v, want errLockFailed", err)
	}
	if got := stderr.String(); !strings.Contains(got, "schema lock is stale") {
		t.Fatalf("stderr = %q, want stale lock message", got)
	}
}

func TestLockCheckRejectsNonCanonicalLockFile(t *testing.T) {
	registryPath := buildLockRegistry(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	update := NewRootCommand(&stdout, &stderr)
	update.SetArgs([]string{"lock", "update", registryPath})
	if err := update.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v, stderr = %s", err, stderr.String())
	}

	lockPath := filepath.Join(registryPath, "openevents.lock.yaml")
	canonical, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", lockPath, err)
	}
	nonCanonical := append([]byte("# non-canonical\n"), canonical...)
	if err := os.WriteFile(lockPath, nonCanonical, 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	check := NewRootCommand(&stdout, &stderr)
	check.SetArgs([]string{"lock", "check", registryPath})

	err = check.Execute()
	if !errors.Is(err, errLockFailed) {
		t.Fatalf("Execute() error = %v, want errLockFailed", err)
	}
	if got := stderr.String(); !strings.Contains(got, "schema lock is not canonical") {
		t.Fatalf("stderr = %q, want non-canonical lock message", got)
	}
}

func TestLockUpdateLoadsSplitDirectoryRegistry(t *testing.T) {
	registryPath := testfx.New().
		Namespace("com.example.product").
		Package("github.com/example/product/events", "example_product.events").
		Owner("data-platform", "data-platform@example.com").
		Domain("search").
		Owner("data-platform").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"query"}, "query_submitted").
		Version(1).Status("active").Description("User submitted a search query.").
		Property("query_text", registry.FieldTypeString, true, registry.PIIPersonal).
		Done().
		Done().
		Write(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"lock", "update", registryPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
	}

	lock, err := readLockFile(filepath.Join(registryPath, "openevents.lock.yaml"))
	if err != nil {
		t.Fatalf("readLockFile() error = %v", err)
	}
	if got, want := len(lock.Events), 1; got != want {
		t.Fatalf("len(lock.Events) = %d, want %d", got, want)
	}
	// Domain context should also be serialized.
	if got, want := len(lock.Domains), 1; got != want {
		t.Fatalf("len(lock.Domains) = %d, want %d", got, want)
	}
}

func TestLockFilePathForDirectoryPath(t *testing.T) {
	registryPath := t.TempDir()
	got := lockFilePath(registryPath)
	want := filepath.Join(registryPath, "openevents.lock.yaml")
	if got != want {
		t.Fatalf("lockFilePath() = %q, want %q", got, want)
	}
}
