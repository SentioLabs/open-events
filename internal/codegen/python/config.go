// Package python emits per-domain Python bindings from an OpenEvents registry.
// It produces event_names/<domain>.py and context/<domain>.py under a configurable
// output directory, with __init__.py files for each sub-package.
package python

import "fmt"

const defaultOut = "./gen/python"

// Config holds code generation settings for the Python emitter.
type Config struct {
	// Out is the output directory root (relative to registry root or absolute).
	// Defaults to "./gen/python".
	Out string
	// Package is the Python package name for the generated modules.
	// Defaults to the registry's Package.Python value.
	Package string
}

// ParseConfig parses the raw codegen config map for the "python" language target,
// applying defaults for missing or empty fields.
func ParseConfig(raw map[string]any, defaultPackage string, registryRoot string) (Config, error) {
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
			return Config{}, fmt.Errorf("codegen.python.out must be a string, got %T", v)
		}
		if s != "" {
			cfg.Out = s
		}
	}

	if v, ok := raw["package"]; ok {
		s, ok := v.(string)
		if !ok {
			return Config{}, fmt.Errorf("codegen.python.package must be a string, got %T", v)
		}
		if s != "" {
			cfg.Package = s
		}
	}

	return cfg, nil
}
