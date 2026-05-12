package registry

import (
	"fmt"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

var (
	eventNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	fieldNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	goPackagePattern     = regexp.MustCompile(`^[a-z0-9]+([._/-][a-z0-9]+)*$`)
	pythonPackagePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
)

func Validate(reg Registry) Diagnostics {
	var diags Diagnostics

	if strings.TrimSpace(reg.Version) == "" {
		diags = append(diags, Diagnostic{Location: "openevents", Message: "openevents is required"})
	} else if reg.Version != SupportedVersion {
		diags = append(diags, Diagnostic{Location: "openevents", Message: fmt.Sprintf("unsupported openevents version %q", reg.Version)})
	}

	if strings.TrimSpace(reg.Namespace) == "" {
		diags = append(diags, Diagnostic{Location: "namespace", Message: "namespace is required"})
	}

	validatePackages(reg.Package, &diags)
	validateContext(reg.Context, &diags)
	validateEvents(reg.Events, &diags)

	return diags
}

func validatePackages(pkg PackageConfig, diags *Diagnostics) {
	if pkg.Go != "" {
		if !goPackagePattern.MatchString(pkg.Go) {
			*diags = append(*diags, Diagnostic{Location: "package.go", Message: "package.go must be a valid Go import path"})
		} else if !strings.Contains(pkg.Go, ".") && !strings.Contains(pkg.Go, "/") {
			*diags = append(*diags, Diagnostic{Location: "package.go", Message: "package.go must include at least one '.' or '/' in the import path"})
		} else {
			parts := strings.Split(pkg.Go, "/")
			base := parts[len(parts)-1]
			if token.Lookup(base).IsKeyword() {
				*diags = append(*diags, Diagnostic{Location: "package.go", Message: "package.go basename must not be a Go keyword"})
			}
		}
	}
	if pkg.Python != "" && !pythonPackagePattern.MatchString(pkg.Python) {
		*diags = append(*diags, Diagnostic{Location: "package.python", Message: "package.python must be a valid Python package name"})
	}
}

func validateContext(context map[string]Field, diags *Diagnostics) {
	for _, name := range sortedFieldKeys(context) {
		validateField("context."+name, context[name], diags)
	}
}

func validateEvents(events []Event, diags *Diagnostics) {
	seen := make(map[string]struct{}, len(events))
	for index, event := range events {
		location := eventLocation(index, event)

		if !eventNamePattern.MatchString(event.Name) {
			*diags = append(*diags, Diagnostic{Location: location + ".name", Message: "event name must be lowercase dot-separated identifiers"})
		}
		if event.Version <= 0 {
			*diags = append(*diags, Diagnostic{Location: location + ".version", Message: "event version must be positive"})
		}

		key := fmt.Sprintf("%s@%d", event.Name, event.Version)
		if _, exists := seen[key]; exists {
			*diags = append(*diags, Diagnostic{Location: fmt.Sprintf("events[%d]", index), Message: fmt.Sprintf("duplicate event name/version %q", key)})
		} else {
			seen[key] = struct{}{}
		}

		if !isSupportedStatus(event.Status) {
			*diags = append(*diags, Diagnostic{Location: location + ".status", Message: fmt.Sprintf("unsupported event status %q", event.Status)})
		}

		for _, name := range sortedFieldKeys(event.Properties) {
			validateField(location+".properties."+name, event.Properties[name], diags)
		}
	}
}

func eventLocation(index int, event Event) string {
	if event.Name == "" {
		return fmt.Sprintf("events[%d]", index)
	}
	return "events." + event.Name
}

func validateField(location string, field Field, diags *Diagnostics) {
	if !fieldNamePattern.MatchString(field.Name) && field.Name != "items" {
		*diags = append(*diags, Diagnostic{Location: location, Message: "field name must be snake_case"})
	}
	if !isSupportedFieldType(field.Type) {
		*diags = append(*diags, Diagnostic{Location: location + ".type", Message: fmt.Sprintf("unsupported field type %q", field.Type)})
	}
	if !isSupportedPII(field.PII) {
		*diags = append(*diags, Diagnostic{Location: location + ".pii", Message: fmt.Sprintf("unsupported pii classification %q", field.PII)})
	}

	switch field.Type {
	case FieldTypeEnum:
		validateEnum(location, field.Values, diags)
	case FieldTypeArray:
		if field.Items == nil {
			*diags = append(*diags, Diagnostic{Location: location + ".items", Message: "array fields must define items"})
		}
	case FieldTypeObject:
		if len(field.Properties) == 0 {
			*diags = append(*diags, Diagnostic{Location: location + ".properties", Message: "object fields must define properties"})
		}
	}

	if field.Items != nil {
		validateField(location+".items", *field.Items, diags)
	}
	for _, name := range sortedFieldKeys(field.Properties) {
		validateField(location+".properties."+name, field.Properties[name], diags)
	}
}

func validateEnum(location string, values []string, diags *Diagnostics) {
	if len(values) == 0 {
		*diags = append(*diags, Diagnostic{Location: location + ".values", Message: "enum fields must define at least one value"})
		return
	}

	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		valueLocation := fmt.Sprintf("%s.values[%d]", location, index)
		if strings.TrimSpace(value) == "" {
			*diags = append(*diags, Diagnostic{Location: valueLocation, Message: "enum values must not be empty"})
			continue
		}
		if _, exists := seen[value]; exists {
			*diags = append(*diags, Diagnostic{Location: valueLocation, Message: fmt.Sprintf("duplicate enum value %q", value)})
			continue
		}
		seen[value] = struct{}{}
	}
}

func isSupportedFieldType(fieldType FieldType) bool {
	switch fieldType {
	case FieldTypeString,
		FieldTypeInteger,
		FieldTypeNumber,
		FieldTypeBoolean,
		FieldTypeTimestamp,
		FieldTypeDate,
		FieldTypeUUID,
		FieldTypeEnum,
		FieldTypeObject,
		FieldTypeArray:
		return true
	default:
		return false
	}
}

func isSupportedPII(pii PIIClassification) bool {
	switch pii {
	case PIINone, PIIPseudonymous, PIIPersonal, PIISensitive:
		return true
	default:
		return false
	}
}

func isSupportedStatus(status string) bool {
	switch status {
	case "active", "deprecated", "experimental":
		return true
	default:
		return false
	}
}

func sortedFieldKeys(fields map[string]Field) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
