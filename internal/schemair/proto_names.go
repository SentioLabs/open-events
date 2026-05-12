package schemair

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/sentiolabs/open-events/internal/registry"
)

func ProtoPackage(namespace string, version int) (string, error) {
	if version <= 0 {
		return "", fmt.Errorf("protobuf version must be positive")
	}

	parts, err := namespaceParts(namespace)
	if err != nil {
		return "", err
	}

	return strings.Join(parts, ".") + ".v" + strconv.Itoa(version), nil
}

func ProtoFilePath(namespace string, version int) (string, error) {
	pkg, err := ProtoPackage(namespace, version)
	if err != nil {
		return "", err
	}

	return path.Join(strings.ReplaceAll(pkg, ".", "/"), "events.proto"), nil
}

func EventMessageName(event registry.Event) string {
	base := pascalCase(event.Name)
	if base == "" {
		base = "Event"
	}
	return fmt.Sprintf("%sV%d", base, event.Version)
}

func PropertiesMessageName(event registry.Event) string {
	return EventMessageName(event) + "Properties"
}

func EnumTypeName(fieldName string) string {
	name := pascalCase(fieldName)
	if name == "" {
		return "Enum"
	}
	return name
}

func EnumValueName(enumName string, raw string) (string, error) {
	prefix, err := upperIdentifier(enumName)
	if err != nil {
		return "", fmt.Errorf("invalid enum name %q: %w", enumName, err)
	}

	value, err := upperIdentifier(raw)
	if err != nil {
		return "", fmt.Errorf("invalid enum value %q: %w", raw, err)
	}

	return prefix + "_" + value, nil
}

func buildEnumValues(enumName string, values []string, fieldPath string) ([]EnumValue, error) {
	out := make([]EnumValue, 0, len(values))
	byName := make(map[string]string, len(values))
	for i, raw := range values {
		name, err := EnumValueName(enumName, raw)
		if err != nil {
			return nil, fmt.Errorf("%s.values[%d]: %w", fieldPath, i, err)
		}
		if first, ok := byName[name]; ok {
			return nil, fmt.Errorf("enum values collide after normalization at %s: %q and %q both map to %q", fieldPath, first, raw, name)
		}
		byName[name] = raw
		out = append(out, EnumValue{Name: name, Original: raw, Number: i + 1})
	}
	return out, nil
}

func namespaceParts(namespace string) ([]string, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("protobuf namespace must not be empty")
	}

	rawParts := strings.Split(namespace, ".")
	parts := make([]string, 0, len(rawParts))
	for _, raw := range rawParts {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, fmt.Errorf("protobuf namespace %q has an empty segment", namespace)
		}
		if startsWithDigit(raw) {
			return nil, fmt.Errorf("protobuf namespace segment %q starts with a digit", raw)
		}
		for _, r := range raw {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
				return nil, fmt.Errorf("protobuf namespace segment %q is invalid; use letters, digits, or underscore", raw)
			}
		}
		parts = append(parts, strings.ToLower(raw))
	}

	return parts, nil
}

func upperIdentifier(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("name must not be empty")
	}

	if startsWithDigit(raw) {
		return "", fmt.Errorf("name starts with a digit")
	}

	parts := splitIdentifier(raw)
	if len(parts) == 0 {
		return "", fmt.Errorf("name must include letters or digits")
	}
	if startsWithDigit(parts[0]) {
		return "", fmt.Errorf("name starts with a digit")
	}

	for i := range parts {
		parts[i] = strings.ToUpper(parts[i])
	}
	return strings.Join(parts, "_"), nil
}

func pascalCase(raw string) string {
	parts := splitIdentifier(raw)
	if len(parts) == 0 {
		return ""
	}

	var b strings.Builder
	for _, part := range parts {
		lower := strings.ToLower(part)
		b.WriteString(strings.ToUpper(lower[:1]))
		if len(lower) > 1 {
			b.WriteString(lower[1:])
		}
	}
	return b.String()
}

func splitIdentifier(raw string) []string {
	runes := []rune(raw)
	parts := make([]string, 0, len(runes))
	current := make([]rune, 0, len(runes))

	flush := func() {
		if len(current) == 0 {
			return
		}
		parts = append(parts, string(current))
		current = current[:0]
	}

	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(current) > 0 {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsUpper(prev) && nextLower) {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()

	return parts
}

func startsWithDigit(s string) bool {
	for _, r := range s {
		return unicode.IsDigit(r)
	}
	return false
}
