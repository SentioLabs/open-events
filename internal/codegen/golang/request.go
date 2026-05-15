package golang

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sentiolabs/open-events/internal/registry"
)

// writeEventRequests emits one <category>_<action>_request.go file per event
// under dir. Each generated file contains:
//   - A top-level <Action>Request struct with one field per registry property
//   - Nested *Request struct types for every object / array-of-object field
//   - A Validate() method on the top-level struct (and Validate(prefix string)
//     on each nested type) that returns []eventmap.FieldError
//   - Private value-set helpers for every enum field
//
// Required primitive fields use pointer types so the validator can distinguish
// "field omitted from JSON" from "field set to its zero value" — the CODEX-4
// pattern, now applied uniformly at the source.
func writeEventRequests(dir string, b domainBundle) error {
	for _, ev := range b.events {
		if err := writeEventRequest(dir, b.contextTypeName, b.eventmapImport, ev); err != nil {
			return err
		}
	}
	return nil
}

// domainBundle carries the per-domain context type name (e.g. "UserContext")
// and the eventmap parent-package import path alongside the events so the
// request emitter can reference the right type without recomputing the
// domain → context-type mapping on every event.
type domainBundle struct {
	domain          string
	contextTypeName string
	eventmapImport  string // e.g. github.com/example/foo/eventmap
	events          []registry.Event
}

func writeEventRequest(dir, contextTypeName, eventmapImport string, ev registry.Event) error {
	base := eventFileBase(ev)
	eventType := eventRequestTypeName(ev)

	var sb strings.Builder
	sb.WriteString(generatedHeader)
	sb.WriteString("\n\n")
	fmt.Fprintf(&sb, "package %s\n\n", ev.Domain)

	needsTime := propertiesNeedTimeParse(ev.Properties)
	needsFmt := propertiesContainArrayOfObject(ev.Properties)

	sb.WriteString("import (\n")
	if needsFmt {
		sb.WriteString("\t\"fmt\"\n")
	}
	if needsTime {
		sb.WriteString("\t\"time\"\n")
	}
	if needsFmt || needsTime {
		sb.WriteString("\n")
	}
	fmt.Fprintf(&sb, "\t%q\n", eventmapImport)
	sb.WriteString(")\n\n")

	// Top-level request struct (with Context field).
	emitRequestStruct(&sb, eventType, ev.Properties, contextTypeName)

	// Nested struct types (recursive, deterministic order by field name).
	walkNested(ev.Properties, make(map[string]struct{}), func(typeName string, props map[string]registry.Field) {
		emitRequestStruct(&sb, typeName, props, "")
	})

	// Top-level Validate method.
	emitTopLevelValidate(&sb, eventType, ev.Properties)

	// Nested Validate(prefix string) methods.
	walkNested(ev.Properties, make(map[string]struct{}), func(typeName string, props map[string]registry.Field) {
		emitNestedValidate(&sb, typeName, props)
	})

	// Enum value-set helpers, owned by the type that contains the enum field.
	emitted := make(map[string]struct{})
	emitEnumValueSets(&sb, eventType, ev.Properties, emitted)
	walkNested(ev.Properties, make(map[string]struct{}), func(typeName string, props map[string]registry.Field) {
		emitEnumValueSets(&sb, typeName, props, emitted)
	})

	return writeFile(filepath.Join(dir, base+"_request.go"), sb.String())
}

// emitRequestStruct writes a Go struct definition for a request type. If
// contextTypeName is non-empty, a `Context <contextTypeName>` field is
// included as the first field (used for the top-level event request struct);
// nested types pass "" to omit it.
func emitRequestStruct(sb *strings.Builder, typeName string, properties map[string]registry.Field, contextTypeName string) {
	fmt.Fprintf(sb, "// %s is the JSON body shape for this request.\n", typeName)
	sb.WriteString("// Required primitive fields use pointer types so the validator can\n")
	sb.WriteString("// distinguish \"field omitted from JSON\" from \"field set to its zero value\".\n")
	fmt.Fprintf(sb, "type %s struct {\n", typeName)
	if contextTypeName != "" {
		fmt.Fprintf(sb, "\tContext %s `json:\"context\"`\n", contextTypeName)
	}
	for _, name := range sortedFieldNames(properties) {
		f := properties[name]
		goName := camelToExported(name)
		goType := fieldGoTypeForRequest(f)
		tag := buildJSONTag(name, f.Required)
		fmt.Fprintf(sb, "\t%s %s `json:\"%s\"`\n", goName, goType, tag)
	}
	sb.WriteString("}\n\n")
}

// walkNested visits every object field (and array-of-object element type) in
// properties exactly once, in deterministic order, calling visit with the
// generated Go type name and the field map for that nested type.
func walkNested(properties map[string]registry.Field, seen map[string]struct{}, visit func(typeName string, props map[string]registry.Field)) {
	for _, name := range sortedFieldNames(properties) {
		f := properties[name]
		switch f.Type {
		case registry.FieldTypeObject:
			typeName := nestedObjectTypeName(name)
			if _, ok := seen[typeName]; ok {
				continue
			}
			seen[typeName] = struct{}{}
			visit(typeName, f.Properties)
			walkNested(f.Properties, seen, visit)
		case registry.FieldTypeArray:
			if f.Items != nil && f.Items.Type == registry.FieldTypeObject {
				typeName := nestedArrayElementTypeName(name)
				if _, ok := seen[typeName]; ok {
					continue
				}
				seen[typeName] = struct{}{}
				visit(typeName, f.Items.Properties)
				walkNested(f.Items.Properties, seen, visit)
			}
		}
	}
}

// emitTopLevelValidate writes the Validate() method on the top-level event
// request struct. The error-path expression for each top-level field is a
// literal string (e.g. "method") rendered via %q.
func emitTopLevelValidate(sb *strings.Builder, typeName string, properties map[string]registry.Field) {
	fmt.Fprintf(sb, "// Validate returns field-level errors for the request, empty on success.\n")
	fmt.Fprintf(sb, "func (r %s) Validate() []eventmap.FieldError {\n", typeName)
	sb.WriteString("\terrs := validateContext(r.Context)\n")
	for _, name := range sortedFieldNames(properties) {
		f := properties[name]
		// Top-level path expression is the literal JSON name.
		pathExpr := fmt.Sprintf("%q", name)
		emitFieldValidation(sb, name, f, pathExpr, typeName)
	}
	sb.WriteString("\treturn errs\n}\n\n")
}

// emitNestedValidate writes the Validate(prefix string) method on a nested
// request type. The error-path expression for each field is `prefix + "." + name`.
func emitNestedValidate(sb *strings.Builder, typeName string, properties map[string]registry.Field) {
	fmt.Fprintf(sb, "// Validate returns field-level errors for %s under prefix, empty on success.\n", typeName)
	fmt.Fprintf(sb, "func (r *%s) Validate(prefix string) []eventmap.FieldError {\n", typeName)
	sb.WriteString("\tif r == nil {\n\t\treturn nil\n\t}\n")
	sb.WriteString("\tvar errs []eventmap.FieldError\n")
	for _, name := range sortedFieldNames(properties) {
		f := properties[name]
		// Nested path expression: prefix + "." + <name>
		pathExpr := fmt.Sprintf("prefix + %q", "."+name)
		emitFieldValidation(sb, name, f, pathExpr, typeName)
	}
	sb.WriteString("\treturn errs\n}\n\n")
}

// emitFieldValidation emits the validation block for a single field. pathExpr
// is a Go-source expression that evaluates to the field's error path at runtime
// (e.g. `"method"` for top-level, or `prefix + ".major"` for nested).
// ownerType is used to build enum value-set helper names so they stay unique
// across nested structs.
func emitFieldValidation(sb *strings.Builder, jsonName string, f registry.Field, pathExpr, ownerType string) {
	goName := camelToExported(jsonName)
	required := f.Required

	switch f.Type {
	case registry.FieldTypeString, registry.FieldTypeUUID, registry.FieldTypeDate:
		if required {
			fmt.Fprintf(sb, "\tif r.%s == nil {\n", goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"required\"})\n", pathExpr)
			sb.WriteString("\t}\n")
		}

	case registry.FieldTypeTimestamp:
		if required {
			fmt.Fprintf(sb, "\tif r.%s == nil {\n", goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"required\"})\n", pathExpr)
			fmt.Fprintf(sb, "\t} else if _, err := time.Parse(time.RFC3339, *r.%s); err != nil {\n", goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"must be RFC3339 timestamp\"})\n", pathExpr)
			sb.WriteString("\t}\n")
		} else {
			fmt.Fprintf(sb, "\tif r.%s != \"\" {\n", goName)
			fmt.Fprintf(sb, "\t\tif _, err := time.Parse(time.RFC3339, r.%s); err != nil {\n", goName)
			fmt.Fprintf(sb, "\t\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"must be RFC3339 timestamp\"})\n", pathExpr)
			sb.WriteString("\t\t}\n\t}\n")
		}

	case registry.FieldTypeInteger, registry.FieldTypeNumber, registry.FieldTypeBoolean:
		if required {
			fmt.Fprintf(sb, "\tif r.%s == nil {\n", goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"required\"})\n", pathExpr)
			sb.WriteString("\t}\n")
		}

	case registry.FieldTypeEnum:
		valuesFn := enumValueSetFnName(ownerType, jsonName)
		valuesMsg := "must be one of " + strings.Join(f.Values, " | ")
		if required {
			fmt.Fprintf(sb, "\tif r.%s == nil {\n", goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"required\"})\n", pathExpr)
			fmt.Fprintf(sb, "\t} else if !%s(*r.%s) {\n", valuesFn, goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: %q})\n", pathExpr, valuesMsg)
			sb.WriteString("\t}\n")
		} else {
			fmt.Fprintf(sb, "\tif r.%s != \"\" && !%s(r.%s) {\n", goName, valuesFn, goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: %q})\n", pathExpr, valuesMsg)
			sb.WriteString("\t}\n")
		}

	case registry.FieldTypeObject:
		if required {
			fmt.Fprintf(sb, "\tif r.%s == nil {\n", goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"required\"})\n", pathExpr)
			sb.WriteString("\t} else {\n")
			fmt.Fprintf(sb, "\t\terrs = append(errs, r.%s.Validate(%s)...)\n", goName, pathExpr)
			sb.WriteString("\t}\n")
		} else {
			fmt.Fprintf(sb, "\tif r.%s != nil {\n", goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, r.%s.Validate(%s)...)\n", goName, pathExpr)
			sb.WriteString("\t}\n")
		}

	case registry.FieldTypeArray:
		if f.Items != nil && f.Items.Type == registry.FieldTypeObject {
			if required {
				fmt.Fprintf(sb, "\tif r.%s == nil {\n", goName)
				fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"required\"})\n", pathExpr)
				sb.WriteString("\t} else {\n")
			} else {
				fmt.Fprintf(sb, "\tif r.%s != nil {\n", goName)
			}
			fmt.Fprintf(sb, "\t\tfor i, elem := range r.%s {\n", goName)
			fmt.Fprintf(sb, "\t\t\telemPath := fmt.Sprintf(\"%%s[%%d]\", %s, i)\n", pathExpr)
			sb.WriteString("\t\t\tif elem == nil {\n")
			sb.WriteString("\t\t\t\terrs = append(errs, eventmap.FieldError{Field: elemPath, Message: \"required\"})\n")
			sb.WriteString("\t\t\t} else {\n")
			sb.WriteString("\t\t\t\terrs = append(errs, elem.Validate(elemPath)...)\n")
			sb.WriteString("\t\t\t}\n")
			sb.WriteString("\t\t}\n\t}\n")
		} else if required {
			fmt.Fprintf(sb, "\tif r.%s == nil {\n", goName)
			fmt.Fprintf(sb, "\t\terrs = append(errs, eventmap.FieldError{Field: %s, Message: \"required\"})\n", pathExpr)
			sb.WriteString("\t}\n")
		}
	}
}

// emitEnumValueSets writes a private value-set helper for every enum field
// directly owned by properties (not nested types — those are handled by
// separate emitEnumValueSets calls per nested type). The emitted map records
// which helper names have already been written, preventing duplicate emission
// when a nested enum's owner is shared.
func emitEnumValueSets(sb *strings.Builder, ownerType string, properties map[string]registry.Field, emitted map[string]struct{}) {
	for _, name := range sortedFieldNames(properties) {
		f := properties[name]
		if f.Type != registry.FieldTypeEnum {
			continue
		}
		fnName := enumValueSetFnName(ownerType, name)
		if _, ok := emitted[fnName]; ok {
			continue
		}
		emitted[fnName] = struct{}{}
		values := append([]string(nil), f.Values...)
		sort.Strings(values)
		fmt.Fprintf(sb, "var %sSet = map[string]struct{}{\n", fnName)
		for _, v := range values {
			fmt.Fprintf(sb, "\t%q: {},\n", v)
		}
		sb.WriteString("}\n\n")
		fmt.Fprintf(sb, "func %s(s string) bool {\n", fnName)
		fmt.Fprintf(sb, "\t_, ok := %sSet[s]\n", fnName)
		sb.WriteString("\treturn ok\n}\n\n")
	}
}

// enumValueSetFnName builds the helper-function name for an enum field's
// value-set check. Format: `valid<OwnerType><FieldPascal>`. The ownerType
// disambiguates enums named the same across nested types.
func enumValueSetFnName(ownerType, fieldName string) string {
	return "valid" + ownerType + camelToExported(fieldName)
}

// fieldGoTypeForRequest maps a registry Field to the Go type used inside a
// generated request struct.
func fieldGoTypeForRequest(f registry.Field) string {
	switch f.Type {
	case registry.FieldTypeString, registry.FieldTypeUUID, registry.FieldTypeDate, registry.FieldTypeTimestamp, registry.FieldTypeEnum:
		if f.Required {
			return "*string"
		}
		return "string"
	case registry.FieldTypeInteger:
		if f.Required {
			return "*int64"
		}
		return "int64"
	case registry.FieldTypeNumber:
		if f.Required {
			return "*float64"
		}
		return "float64"
	case registry.FieldTypeBoolean:
		if f.Required {
			return "*bool"
		}
		return "bool"
	case registry.FieldTypeObject:
		return "*" + nestedObjectTypeName(f.Name)
	case registry.FieldTypeArray:
		if f.Items == nil {
			return "[]any"
		}
		if f.Items.Type == registry.FieldTypeObject {
			return "[]*" + nestedArrayElementTypeName(f.Name)
		}
		return "[]" + primitiveGoType(f.Items.Type)
	default:
		return "any"
	}
}

// nestedTypeName returns the canonical generated type name for a nested
// shape, dispatching on whether the source is an object field or an array
// element. Kept for backward-compatibility with callers that don't yet
// distinguish the two cases.
func nestedTypeName(fieldName string) string {
	return nestedObjectTypeName(fieldName)
}

// nestedObjectTypeName names the Go type for a nested object field.
// E.g. "eeprom_format_version" → "EepromFormatVersionRequest".
func nestedObjectTypeName(fieldName string) string {
	return pascalCase(fieldName) + "Request"
}

// nestedArrayElementTypeName names the Go type for the element type of an
// array-of-object field. The field name is depluralized so the element type
// reads naturally (e.g. `threads []*ThreadRequest`, not `[]*ThreadsRequest`).
func nestedArrayElementTypeName(fieldName string) string {
	return pascalCase(singularize(fieldName)) + "Request"
}

// singularize applies a tiny English-singularization rule: drop a trailing
// "s" when the word ends in one. This is intentionally conservative — it
// handles the demo's only array case ("threads" → "thread") and other
// straightforward plurals without trying to invert "ies" → "y" or other
// irregular plurals. Registries that want a specific element-type name can
// rename the array field accordingly.
func singularize(s string) string {
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return strings.TrimSuffix(s, "s")
	}
	return s
}

func primitiveGoType(t registry.FieldType) string {
	switch t {
	case registry.FieldTypeString, registry.FieldTypeUUID, registry.FieldTypeEnum, registry.FieldTypeTimestamp, registry.FieldTypeDate:
		return "string"
	case registry.FieldTypeInteger:
		return "int64"
	case registry.FieldTypeNumber:
		return "float64"
	case registry.FieldTypeBoolean:
		return "bool"
	default:
		return "any"
	}
}

// eventRequestTypeName returns the Go type name for an event's top-level
// request struct. E.g. user.auth.signup → "AuthSignupRequest".
func eventRequestTypeName(ev registry.Event) string {
	return eventPascalName(ev) + "Request"
}

// eventPascalName returns the PascalCase form of an event's name with the
// domain prefix dropped. E.g. user.auth.signup → "AuthSignup".
func eventPascalName(ev registry.Event) string {
	parts := strings.Split(ev.Name, ".")
	if len(parts) > 0 && parts[0] == ev.Domain {
		parts = parts[1:]
	}
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(camelSegment(p))
	}
	return sb.String()
}

// eventFileBase returns the file-name base for an event's generated request
// file (without _request.go suffix). It joins the post-domain path segments
// of ev.Name with underscores. E.g. user.auth.signup → "auth_signup".
func eventFileBase(ev registry.Event) string {
	parts := strings.Split(ev.Name, ".")
	if len(parts) > 0 && parts[0] == ev.Domain {
		parts = parts[1:]
	}
	return strings.Join(parts, "_")
}

// sortedFieldNames returns the keys of properties in sorted order so emitted
// output is deterministic.
func sortedFieldNames(properties map[string]registry.Field) []string {
	names := make([]string, 0, len(properties))
	for n := range properties {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func propertiesNeedTimeParse(properties map[string]registry.Field) bool {
	for _, f := range properties {
		if f.Type == registry.FieldTypeTimestamp {
			return true
		}
		if f.Type == registry.FieldTypeObject && propertiesNeedTimeParse(f.Properties) {
			return true
		}
		if f.Type == registry.FieldTypeArray && f.Items != nil && f.Items.Type == registry.FieldTypeObject && propertiesNeedTimeParse(f.Items.Properties) {
			return true
		}
	}
	return false
}

func propertiesContainArrayOfObject(properties map[string]registry.Field) bool {
	for _, f := range properties {
		if f.Type == registry.FieldTypeArray && f.Items != nil && f.Items.Type == registry.FieldTypeObject {
			return true
		}
		if f.Type == registry.FieldTypeObject && propertiesContainArrayOfObject(f.Properties) {
			return true
		}
	}
	return false
}
