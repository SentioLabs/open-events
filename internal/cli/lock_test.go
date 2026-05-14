package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/schemair"
)

func TestLockUpdateWritesLockFile(t *testing.T) {
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
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"lock", "update", registryPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	lockPath := filepath.Join(registryPath, "openevents.lock.yaml")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("Stat(%q) error = %v", lockPath, err)
	}
}

func TestLockCheckRejectsStaleLock(t *testing.T) {
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

	lockPath := filepath.Join(registryPath, "openevents.lock.yaml")
	lock, err := readLockFile(lockPath)
	if err != nil {
		t.Fatalf("readLockFile(%q) error = %v", lockPath, err)
	}
	// T3: Lock.Context replaced by per-domain Lock.Domains; T6 will update this
	// test to tamper with domains. Clear all events to simulate a stale lock.
	lock.Events = map[string]schemair.LockedEvent{}
	_ = lock.Domains // Domains now holds per-domain context; tampering deferred to T6
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

	lockPath := filepath.Join(registryPath, "openevents.lock.yaml")
	canonical, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", lockPath, err)
	}
	nonCanonical := append([]byte("# non-canonical\n"), canonical...)
	if err := os.WriteFile(lockPath, nonCanonical, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
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
	registryPath := t.TempDir()
	root := []byte(`openevents: 0.1.0
namespace: com.example.product
package:
  go: github.com/example/product/events
  python: example_product.events
defaults:
  queue: product-events
  snowflake:
    database: ANALYTICS
    schema: EVENTS
owners:
  - team: data-platform
    email: data-platform@example.com
context:
  tenant_id:
    type: string
    required: true
    pii: none
`)
	fragment := []byte(`events:
  search.query_submitted:
    version: 1
    status: active
    description: User submitted a search query.
    owner: data-platform
    producer: api
    sources: [ios]
    destination:
      queue: product-events
      snowflake_table: fact_search_query_submitted
    properties:
      query_text:
        type: string
        required: true
        pii: personal
`)
	if err := os.WriteFile(filepath.Join(registryPath, "openevents.yaml"), root, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryPath, "events.yaml"), fragment, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"lock", "update", registryPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	lock, err := readLockFile(filepath.Join(registryPath, "openevents.lock.yaml"))
	if err != nil {
		t.Fatalf("readLockFile() error = %v", err)
	}
	if got, want := len(lock.Events), 1; got != want {
		t.Fatalf("len(lock.Events) = %d, want %d", got, want)
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
