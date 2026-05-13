package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCommandUnknownTargetFails(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", "ruby", "../../examples/basic", t.TempDir()})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want non-nil for unknown generate target")
	}
}

func TestGenerateProtoUnsupportedWithoutLockWritesErrOut(t *testing.T) {
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
	cmd.SetArgs([]string{"generate", "proto", registryPath, t.TempDir()})

	err = cmd.Execute()
	if !errors.Is(err, errGenerationFailed) {
		t.Fatalf("Execute() error = %v, want errGenerationFailed", err)
	}
	if got := stderr.String(); !strings.Contains(got, "openevents.lock.yaml") {
		t.Fatalf("stderr = %q, want lock file error", got)
	}
}

func TestGenerateProtoDoesNotRunBuf(t *testing.T) {
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
	lockCmd := NewRootCommand(&stdout, &stderr)
	lockCmd.SetArgs([]string{"lock", "update", registryPath})
	if err := lockCmd.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v", err)
	}

	workDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workDir, ".tools", "bin"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	bufPath := filepath.Join(workDir, ".tools", "bin", "buf")
	if err := os.WriteFile(bufPath, []byte("#!/bin/sh\nexit 77\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(prevWD)
	}()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"generate", "proto", registryPath, t.TempDir()})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "ok: generated proto schema in") {
		t.Fatalf("stdout = %q, want proto success message", got)
	}
}
