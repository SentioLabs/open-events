package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/sentiolabs/open-events/internal/registry"
)

func GenerateGo(reg registry.Registry, outputDir string) error {
	content, err := renderGo(reg)
	if err != nil {
		return err
	}

	if err := ensureDir(outputDir); err != nil {
		return err
	}

	return writeFile(filepath.Join(outputDir, "events_gen.go"), content)
}

func GeneratePython(reg registry.Registry, outputDir string) error {
	packagePath, err := pythonPackagePath(reg.Package.Python)
	if err != nil {
		return err
	}

	root := filepath.Join(outputDir, packagePath)
	if err := ensureDir(root); err != nil {
		return err
	}

	content, err := renderPython(reg)
	if err != nil {
		return err
	}

	return writeFile(filepath.Join(root, "__init__.py"), content)
}

func pythonPackagePath(pkg string) (string, error) {
	if pkg == "" {
		return "", fmt.Errorf("package.python is required")
	}
	return filepath.Join(splitDotPath(pkg)...), nil
}
