package codegen

import (
	"strings"
	"testing"
)

func TestGoPackageNameRejectsInvalidIdentifier(t *testing.T) {
	_, err := goPackageName("github.com/acme/open-events")
	if err == nil {
		t.Fatalf("goPackageName() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "package.go") || !strings.Contains(err.Error(), "open-events") {
		t.Fatalf("goPackageName() error = %q, want message mentioning package.go and invalid basename", err)
	}
}

func TestGoPackageNameRejectsKeyword(t *testing.T) {
	_, err := goPackageName("github.com/acme/type")
	if err == nil {
		t.Fatalf("goPackageName() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "package.go") || !strings.Contains(err.Error(), "type") {
		t.Fatalf("goPackageName() error = %q, want message mentioning package.go and keyword basename", err)
	}
}
