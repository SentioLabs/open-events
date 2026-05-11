package codegen

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/sentiolabs/open-events/internal/registry"
)

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func writeFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func goPackageName(modulePath string) (string, error) {
	if modulePath == "" {
		return "", fmt.Errorf("package.go is required")
	}
	return path.Base(modulePath), nil
}

func splitDotPath(pkg string) []string {
	parts := strings.Split(pkg, ".")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func sortedFieldNames(fields map[string]registry.Field) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func exportedName(name string) string {
	tokens := nonAlphaNum.Split(name, -1)
	var b strings.Builder
	for _, token := range tokens {
		if token == "" {
			continue
		}
		lower := strings.ToLower(token)
		switch lower {
		case "id":
			b.WriteString("ID")
		case "ts":
			b.WriteString("TS")
		case "uuid":
			b.WriteString("UUID")
		default:
			b.WriteString(strings.ToUpper(lower[:1]))
			if len(lower) > 1 {
				b.WriteString(lower[1:])
			}
		}
	}
	result := b.String()
	if result == "" {
		return "Field"
	}
	if result[0] >= '0' && result[0] <= '9' {
		return "X" + result
	}
	return result
}

func pythonClassName(name string) string {
	return exportedName(name)
}
