// Package golang emits per-domain Go bindings from an OpenEvents registry.
// It produces event_names.go and context.go under a configurable
// output directory, one subdirectory per domain.
package golang

import "fmt"

const defaultOut = "./gen/go"

// Config holds code generation settings for the Go emitter.
type Config struct {
	// Out is the output directory root (relative to registry root or absolute).
	// Defaults to "./gen/go".
	Out string
	// Package is the Go import path base for the generated subpackages.
	// Defaults to the registry's Package.Go value.
	Package string
}

// ParseConfig parses the raw codegen config map for the "go" language target,
// applying defaults for missing or empty fields.
func ParseConfig(raw map[string]any, defaultPackage string) (Config, error) {
	cfg := Config{
		Out:     defaultOut,
		Package: defaultPackage,
	}

	if raw == nil {
		return cfg, nil
	}

	if v, ok := raw["out"]; ok {
		s, ok := v.(string)
		if !ok {
			return Config{}, fmt.Errorf("codegen.go.out must be a string, got %T", v)
		}
		if s != "" {
			cfg.Out = s
		}
	}

	if v, ok := raw["package"]; ok {
		s, ok := v.(string)
		if !ok {
			return Config{}, fmt.Errorf("codegen.go.package must be a string, got %T", v)
		}
		if s != "" {
			cfg.Package = s
		}
	}

	return cfg, nil
}
