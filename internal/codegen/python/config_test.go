package python_test

import (
	"testing"

	python "github.com/sentiolabs/open-events/internal/codegen/python"
)

func TestParseConfig_Defaults(t *testing.T) {
	cfg, err := python.ParseConfig(nil, "acme.events", "/registry/root")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v, want nil", err)
	}
	if cfg.Out != "./gen/python" {
		t.Errorf("cfg.Out = %q, want %q", cfg.Out, "./gen/python")
	}
	if cfg.Package != "acme.events" {
		t.Errorf("cfg.Package = %q, want %q", cfg.Package, "acme.events")
	}
}

func TestParseConfig_EmptyRaw_Defaults(t *testing.T) {
	cfg, err := python.ParseConfig(map[string]any{}, "acme.events", "/registry/root")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v, want nil", err)
	}
	if cfg.Out != "./gen/python" {
		t.Errorf("cfg.Out = %q, want %q", cfg.Out, "./gen/python")
	}
	if cfg.Package != "acme.events" {
		t.Errorf("cfg.Package = %q, want %q", cfg.Package, "acme.events")
	}
}

func TestParseConfig_Overrides(t *testing.T) {
	raw := map[string]any{
		"out":     "./custom/out",
		"package": "custom.pkg",
	}
	cfg, err := python.ParseConfig(raw, "acme.events", "/registry/root")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v, want nil", err)
	}
	if cfg.Out != "./custom/out" {
		t.Errorf("cfg.Out = %q, want %q", cfg.Out, "./custom/out")
	}
	if cfg.Package != "custom.pkg" {
		t.Errorf("cfg.Package = %q, want %q", cfg.Package, "custom.pkg")
	}
}

func TestParseConfig_InvalidOutType(t *testing.T) {
	raw := map[string]any{
		"out": 42,
	}
	_, err := python.ParseConfig(raw, "acme.events", "/registry/root")
	if err == nil {
		t.Fatal("ParseConfig() error = nil, want non-nil for invalid out type")
	}
}

func TestParseConfig_InvalidPackageType(t *testing.T) {
	raw := map[string]any{
		"package": true,
	}
	_, err := python.ParseConfig(raw, "acme.events", "/registry/root")
	if err == nil {
		t.Fatal("ParseConfig() error = nil, want non-nil for invalid package type")
	}
}
