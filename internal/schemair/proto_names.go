package schemair

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/sentiolabs/open-events/internal/registry"
)

// Protobuf reserved keywords and reserved words.
var protobufKeywords = map[string]struct{}{
	// Grammar keywords
	"syntax":     {},
	"import":     {},
	"package":    {},
	"option":     {},
	"message":    {},
	"enum":       {},
	"service":    {},
	"rpc":        {},
	"returns":    {},
	"reserved":   {},
	"repeated":   {},
	"optional":   {},
	"required":   {},
	"oneof":      {},
	"map":        {},
	"extend":     {},
	"extends":    {},
	"extensions": {},
	"group":      {},
	"to":         {},
	"max":        {},
	"public":     {},
	"weak":       {},
	"stream":     {},
	// Protobuf scalar types
	"double":   {},
	"float":    {},
	"int32":    {},
	"int64":    {},
	"uint32":   {},
	"uint64":   {},
	"sint32":   {},
	"sint64":   {},
	"fixed32":  {},
	"fixed64":  {},
	"sfixed32": {},
	"sfixed64": {},
	"bool":     {},
	"string":   {},
	"bytes":    {},
	"true":     {},
	"false":    {},
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

func validateEventName(name string) error {
	if name == "" {
		return fmt.Errorf("event name must not be empty")
	}

	if first := rune(name[0]); !isASCIILetter(first) {
		return fmt.Errorf("event name must start with a letter, got %q", string(first))
	}

	for _, r := range name {
		if r > 127 {
			return fmt.Errorf("event name contains non-ASCII character: %q", string(r))
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return fmt.Errorf("event name contains whitespace")
		}
		if !isASCIILetter(r) && !isASCIIDigit(r) && r != '.' && r != '_' && r != '-' {
			return fmt.Errorf("event name contains unsupported character: %q", string(r))
		}
	}

	last := rune(name[len(name)-1])
	if last == '.' || last == '_' || last == '-' {
		return fmt.Errorf("event name contains empty segment (trailing separator)")
	}

	prevWasSeparator := false
	for _, r := range name {
		isSeparator := r == '.' || r == '_' || r == '-'
		if isSeparator && prevWasSeparator {
			return fmt.Errorf("event name contains empty segment (consecutive separators)")
		}
		prevWasSeparator = isSeparator
	}

	return nil
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

	// Validate raw enum value has no whitespace before processing
	if err := validateNoWhitespace(raw); err != nil {
		return "", fmt.Errorf("invalid enum value %q: %w", raw, err)
	}

	value, err := upperIdentifier(raw)
	if err != nil {
		return "", fmt.Errorf("invalid enum value %q: %w", raw, err)
	}

	// Check if the normalized value collides with the reserved zero value
	// Compare without underscores since "un-specified" -> "UN_SPECIFIED"
	valueNoUnderscore := strings.ReplaceAll(strings.ToUpper(value), "_", "")
	if valueNoUnderscore == "UNSPECIFIED" {
		return "", fmt.Errorf("enum value %q normalizes to UNSPECIFIED which collides with reserved zero value %s_UNSPECIFIED", raw, prefix)
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

func validateNoWhitespace(s string) error {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return fmt.Errorf("contains whitespace character")
		}
	}
	return nil
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
	_, ok := protobufKeywords[strings.ToLower(name)]
	return ok
}

// isValidProtoIdentifier checks if a string is a valid protobuf identifier:
// ASCII-only, starts with letter, contains only letters/digits/underscore,
// and is not a reserved keyword.
func isValidProtoIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier must not be empty")
	}

	if first := rune(name[0]); !isASCIILetter(first) {
		return fmt.Errorf("identifier must start with a letter, got %q", string(first))
	}

	for _, r := range name {
		if r > 127 {
			return fmt.Errorf("identifier contains non-ASCII character: %q", string(r))
		}
		if !isASCIILetter(r) && !isASCIIDigit(r) && r != '_' {
			return fmt.Errorf("identifier contains invalid character: %q", string(r))
		}
	}

	if isProtobufKeyword(name) {
		return fmt.Errorf("identifier %q is a reserved keyword", name)
	}

	return nil
}

// EnumZeroValueName computes the synthesized zero value name for an enum
// type. Enum type names are already validated as ASCII PascalCase, so
// splitIdentifier produces deterministic output that the protogen renderer
// can rely on without duplicating the algorithm.
func EnumZeroValueName(enumTypeName string) string {
	parts := splitIdentifier(enumTypeName)
	if len(parts) == 0 {
		return "ENUM_UNSPECIFIED"
	}
	for i := range parts {
		parts[i] = strings.ToUpper(parts[i])
	}
	return strings.Join(parts, "_") + "_UNSPECIFIED"
}

func isValidProtoMessageName(name string) error {
	if name == "" {
		return fmt.Errorf("message name must not be empty")
	}

	if first := rune(name[0]); !isASCIIUpper(first) {
		return fmt.Errorf("message name must start with uppercase letter, got %q", name)
	}

	for _, r := range name {
		if r > 127 {
			return fmt.Errorf("message name contains non-ASCII character in %q", name)
		}
		if !isASCIILetter(r) && !isASCIIDigit(r) {
			return fmt.Errorf("message name contains invalid character in %q", name)
		}
	}

	return nil
}
