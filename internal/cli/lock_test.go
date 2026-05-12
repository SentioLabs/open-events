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
	lock.Context = map[string]schemair.LockedField{}
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

func TestLockFilePathForDirectoryPath(t *testing.T) {
	registryPath := t.TempDir()
	got := lockFilePath(registryPath)
	want := filepath.Join(registryPath, "openevents.lock.yaml")
	if got != want {
		t.Fatalf("lockFilePath() = %q, want %q", got, want)
	}
}
