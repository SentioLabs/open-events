package codegen

import (
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
)

func TestRenderPythonRejectsKeywordFieldName(t *testing.T) {
	reg := registry.Registry{
		Package: registry.PackageConfig{Python: "acme.events"},
		Context: map[string]registry.Field{
			"class": {Name: "class", Type: registry.FieldTypeString},
		},
	}

	_, err := renderPython(reg)
	if err == nil {
		t.Fatalf("renderPython() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "class") || !strings.Contains(err.Error(), "python") {
		t.Fatalf("renderPython() error = %q, want actionable python keyword error", err)
	}
}
