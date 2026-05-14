package golang_test

import (
	"testing"

	golang "github.com/sentiolabs/open-events/internal/codegen/golang"
)

func TestParseConfig_Defaults(t *testing.T) {
	cfg, err := golang.ParseConfig(nil, "github.com/acme/events")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v, want nil", err)
	}
	if cfg.Out != "./gen/go" {
		t.Errorf("cfg.Out = %q, want %q", cfg.Out, "./gen/go")
	}
	if cfg.Package != "github.com/acme/events" {
		t.Errorf("cfg.Package = %q, want %q", cfg.Package, "github.com/acme/events")
	}
}

func TestParseConfig_EmptyRaw_Defaults(t *testing.T) {
	cfg, err := golang.ParseConfig(map[string]any{}, "github.com/acme/events")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v, want nil", err)
	}
	if cfg.Out != "./gen/go" {
		t.Errorf("cfg.Out = %q, want %q", cfg.Out, "./gen/go")
	}
	if cfg.Package != "github.com/acme/events" {
		t.Errorf("cfg.Package = %q, want %q", cfg.Package, "github.com/acme/events")
	}
}

func TestParseConfig_Overrides(t *testing.T) {
	raw := map[string]any{
		"out":     "./custom/out",
		"package": "github.com/custom/pkg",
	}
	cfg, err := golang.ParseConfig(raw, "github.com/acme/events")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v, want nil", err)
	}
	if cfg.Out != "./custom/out" {
		t.Errorf("cfg.Out = %q, want %q", cfg.Out, "./custom/out")
	}
	if cfg.Package != "github.com/custom/pkg" {
		t.Errorf("cfg.Package = %q, want %q", cfg.Package, "github.com/custom/pkg")
	}
}

func TestParseConfig_InvalidOutType(t *testing.T) {
	raw := map[string]any{
		"out": 42,
	}
	_, err := golang.ParseConfig(raw, "github.com/acme/events")
	if err == nil {
		t.Fatal("ParseConfig() error = nil, want non-nil for invalid out type")
	}
}

func TestParseConfig_InvalidPackageType(t *testing.T) {
	raw := map[string]any{
		"package": true,
	}
	_, err := golang.ParseConfig(raw, "github.com/acme/events")
	if err == nil {
		t.Fatal("ParseConfig() error = nil, want non-nil for invalid package type")
	}
}
