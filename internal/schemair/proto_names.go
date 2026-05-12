package schemair

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/sentiolabs/open-events/internal/registry"
)

// Protobuf reserved keywords and reserved words
var protobufKeywords = map[string]bool{
	"syntax":   true,
	"import":   true,
	"package":  true,
	"option":   true,
	"message":  true,
	"enum":     true,
	"service":  true,
	"rpc":      true,
	"returns":  true,
	"reserved": true,
	"repeated": true,
	"optional": true,
	"required": true,
	"oneof":    true,
	"map":      true,
	"extend":   true,
	"extends":  true,
	"group":    true,
	"to":       true,
	"max":      true,
	// Protobuf scalar types
	"double":   true,
	"float":    true,
	"int32":    true,
	"int64":    true,
	"uint32":   true,
	"uint64":   true,
	"sint32":   true,
	"sint64":   true,
	"fixed32":  true,
	"fixed64":  true,
	"sfixed32": true,
	"sfixed64": true,
	"bool":     true,
	"string":   true,
	"bytes":    true,
	"true":     true,
	"false":    true,
}

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
	return fmt.Sprintf("%sV%d", base, event.Version)
}

func PropertiesMessageName(event registry.Event) string {
	return EventMessageName(event) + "Properties"
}

func EnumTypeName(fieldName string) string {
	return pascalCase(fieldName)
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
	byName := make(map[string]string, len(values)+1)

	// Reserve the zero value name (e.g., PAYMENT_METHOD_UNSPECIFIED)
	prefix, err := upperIdentifier(enumName)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid enum name %q: %w", fieldPath, enumName, err)
	}
	reservedZeroName := prefix + "_UNSPECIFIED"
	byName[reservedZeroName] = "<reserved zero value>"

	for i, raw := range values {
		name, err := EnumValueName(enumName, raw)
		if err != nil {
			return nil, fmt.Errorf("%s.values[%d]: %w", fieldPath, i, err)
		}
		if first, ok := byName[name]; ok {
			if first == "<reserved zero value>" {
				return nil, fmt.Errorf("%s.values[%d]: value %q normalizes to reserved zero value name %q", fieldPath, i, raw, name)
			}
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
		// Check first character is a letter
		if len(raw) > 0 {
			first := rune(raw[0])
			if !isASCIILetter(first) {
				return nil, fmt.Errorf("protobuf namespace segment %q must start with a letter", raw)
			}
		}
		// Check for ASCII-only and valid characters
		for _, r := range raw {
			if r > 127 {
				return nil, fmt.Errorf("protobuf namespace segment %q contains non-ASCII character", raw)
			}
			if !isASCIILetter(r) && !isASCIIDigit(r) && r != '_' {
				return nil, fmt.Errorf("protobuf namespace segment %q is invalid; use ASCII letters, digits, or underscore", raw)
			}
		}
		lower := strings.ToLower(raw)
		// Check if lowercase version is a keyword
		if isProtobufKeyword(lower) {
			return nil, fmt.Errorf("protobuf namespace segment %q is a reserved keyword", raw)
		}
		parts = append(parts, lower)
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

	// Check for non-ASCII and unsupported characters
	for _, r := range raw {
		if r > 127 {
			return "", fmt.Errorf("name contains non-ASCII character: %q", string(r))
		}
		// Allow letters, digits, underscore, hyphen (hyphen will be converted to underscore)
		if !isASCIILetter(r) && !isASCIIDigit(r) && r != '_' && r != '-' {
			return "", fmt.Errorf("name contains unsupported character: %q", string(r))
		}
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
		if !isASCIILetter(r) && !isASCIIDigit(r) {
			flush()
			continue
		}
		if len(current) > 0 {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && isASCIILower(runes[i+1])
			if isASCIIUpper(r) && (isASCIILower(prev) || isASCIIUpper(prev) && nextLower) {
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
		return isASCIIDigit(r)
	}
	return false
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isASCIIUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func isASCIILower(r rune) bool {
	return r >= 'a' && r <= 'z'
}

func isProtobufKeyword(name string) bool {
	return protobufKeywords[strings.ToLower(name)]
}

// isValidProtoIdentifier checks if a string is a valid protobuf identifier:
// ASCII-only, starts with letter, contains only letters/digits/underscore,
// and is not a reserved keyword.
func isValidProtoIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier must not be empty")
	}

	// Check first character is a letter
	if len(name) > 0 {
		first := rune(name[0])
		if !isASCIILetter(first) {
			return fmt.Errorf("identifier must start with a letter, got %q", string(first))
		}
	}

	// Check all characters are ASCII alphanumeric or underscore
	for _, r := range name {
		if r > 127 {
			return fmt.Errorf("identifier contains non-ASCII character: %q", string(r))
		}
		if !isASCIILetter(r) && !isASCIIDigit(r) && r != '_' {
			return fmt.Errorf("identifier contains invalid character: %q", string(r))
		}
	}

	// Check if it's a keyword
	if isProtobufKeyword(name) {
		return fmt.Errorf("identifier %q is a reserved keyword", name)
	}

	return nil
}

// isValidProtoMessageName checks if a generated message name is valid.
func isValidProtoMessageName(name string) error {
	if name == "" {
		return fmt.Errorf("message name must not be empty")
	}

	// Check first character is uppercase letter
	if len(name) > 0 {
		first := rune(name[0])
		if !isASCIIUpper(first) {
			return fmt.Errorf("message name must start with uppercase letter, got %q", name)
		}
	}

	// Check all characters are ASCII alphanumeric
	for _, r := range name {
		if r > 127 {
			return fmt.Errorf("message name contains non-ASCII character in %q", name)
		}
		if !isASCIILetter(r) && !isASCIIDigit(r) {
			return fmt.Errorf("message name contains invalid character in %q", name)
		}
	}

	// Message names don't need keyword check as they follow PascalCase convention
	return nil
}
