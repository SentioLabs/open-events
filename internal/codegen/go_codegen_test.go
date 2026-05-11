package codegen

import (
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
)

func TestRenderGoRejectsCollidingEnumTypeNames(t *testing.T) {
	reg := registry.Registry{
		Package: registry.PackageConfig{Go: "github.com/acme/events"},
		Events: []registry.Event{
			{
				Name:    "user.created",
				Version: 1,
				Properties: map[string]registry.Field{
					"status": {Name: "status", Type: registry.FieldTypeEnum, Values: []string{"active"}},
				},
			},
			{
				Name:    "order.created",
				Version: 1,
				Properties: map[string]registry.Field{
					"status": {Name: "status", Type: registry.FieldTypeEnum, Values: []string{"pending"}},
				},
			},
		},
	}

	_, err := renderGo(reg)
	if err == nil {
		t.Fatalf("renderGo() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "enum") || !strings.Contains(err.Error(), "status") {
		t.Fatalf("renderGo() error = %q, want actionable enum collision error", err)
	}
}

func TestRenderGoEnumOutputDeterministicBySortedFieldNames(t *testing.T) {
	reg := registry.Registry{
		Package: registry.PackageConfig{Go: "github.com/acme/events"},
		Context: map[string]registry.Field{
			"zeta":  {Name: "zeta", Type: registry.FieldTypeEnum, Values: []string{"z"}},
			"alpha": {Name: "alpha", Type: registry.FieldTypeEnum, Values: []string{"a"}},
		},
	}

	got, err := renderGo(reg)
	if err != nil {
		t.Fatalf("renderGo() error = %v, want nil", err)
	}
	alphaIdx := strings.Index(got, "type Alpha string")
	zetaIdx := strings.Index(got, "type Zeta string")
	if alphaIdx == -1 || zetaIdx == -1 {
		t.Fatalf("renderGo() output missing enum types: %s", got)
	}
	if alphaIdx > zetaIdx {
		t.Fatalf("renderGo() enum type order is not deterministic/sorted")
	}
}
