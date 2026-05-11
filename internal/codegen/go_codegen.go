package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sentiolabs/open-events/internal/registry"
)

func renderGo(reg registry.Registry) (string, error) {
	pkg, err := goPackageName(reg.Package.Go)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import \"time\"\n\n")
	b.WriteString("type Client struct {\n")
	b.WriteString("\tName string `json:\"name\"`\n")
	b.WriteString("\tVersion string `json:\"version\"`\n")
	b.WriteString("}\n\n")

	enums, err := collectEnums(reg)
	if err != nil {
		return "", err
	}
	for _, enum := range enums {
		b.WriteString("type " + enum.typeName + " string\n\n")
		b.WriteString("const (\n")
		for _, value := range enum.values {
			b.WriteString("\t" + enum.typeName + exportedName(value) + " " + enum.typeName + " = " + strconv.Quote(value) + "\n")
		}
		b.WriteString(")\n\n")
	}

	b.WriteString("type Context struct {\n")
	for _, name := range sortedFieldNames(reg.Context) {
		field := reg.Context[name]
		b.WriteString("\t" + exportedName(name) + " " + goTypeForField(exportedName(name), field, !field.Required) + " `json:\"" + name + jsonTagSuffix(field.Required) + "\"`\n")
	}
	b.WriteString("}\n\n")

	b.WriteString("type Envelope[T any] struct {\n")
	b.WriteString("\tEventName string `json:\"event_name\"`\n")
	b.WriteString("\tEventVersion int `json:\"event_version\"`\n")
	b.WriteString("\tEventID string `json:\"event_id\"`\n")
	b.WriteString("\tEventTS time.Time `json:\"event_ts\"`\n")
	b.WriteString("\tClient Client `json:\"client\"`\n")
	b.WriteString("\tContext Context `json:\"context\"`\n")
	b.WriteString("\tProperties T `json:\"properties\"`\n")
	b.WriteString("}\n\n")

	for _, event := range reg.Events {
		eventName := exportedName(event.Name) + "V" + fmt.Sprintf("%d", event.Version)
		propsType := eventName + "Properties"
		b.WriteString("type " + propsType + " struct {\n")
		for _, name := range sortedFieldNames(event.Properties) {
			field := event.Properties[name]
			b.WriteString("\t" + exportedName(name) + " " + goTypeForField(eventName+exportedName(name), field, !field.Required) + " `json:\"" + name + jsonTagSuffix(field.Required) + "\"`\n")
		}
		b.WriteString("}\n\n")
		b.WriteString("type " + eventName + " = Envelope[" + propsType + "]\n\n")
		b.WriteString("func New" + eventName + "(eventID string, eventTS time.Time, client Client, context Context, properties " + propsType + ") " + eventName + " {\n")
		b.WriteString("\treturn " + eventName + "{\n")
		b.WriteString("\t\tEventName: \"" + event.Name + "\",\n")
		b.WriteString("\t\tEventVersion: " + fmt.Sprintf("%d", event.Version) + ",\n")
		b.WriteString("\t\tEventID: eventID,\n")
		b.WriteString("\t\tEventTS: eventTS,\n")
		b.WriteString("\t\tClient: client,\n")
		b.WriteString("\t\tContext: context,\n")
		b.WriteString("\t\tProperties: properties,\n")
		b.WriteString("\t}\n")
		b.WriteString("}\n\n")
	}

	return b.String(), nil
}

type enumDef struct {
	typeName string
	values   []string
}

func validateEnumConstantNames(typeName, fieldPath string, values []string) error {
	valueByConstName := map[string]string{}
	for _, value := range values {
		constName := typeName + exportedName(value)
		if firstValue, exists := valueByConstName[constName]; exists {
			return fmt.Errorf("enum constant name collision for type %q at %s: values %q and %q both generate %q; rename one enum value to avoid generated Go constant conflicts", typeName, fieldPath, firstValue, value, constName)
		}
		valueByConstName[constName] = value
	}
	return nil
}

func collectEnums(reg registry.Registry) ([]enumDef, error) {
	enumsByType := map[string]enumDef{}
	enumPathByType := map[string]string{}
	identPathByName := collectGoTopLevelIdentifiers(reg)

	for _, name := range sortedFieldNames(reg.Context) {
		field := reg.Context[name]
		fieldPath := "context." + name
		if err := validateNoNestedEnums(field, fieldPath); err != nil {
			return nil, err
		}
		if field.Type != registry.FieldTypeEnum {
			continue
		}
		typeName := exportedName(name)
		if firstPath, exists := identPathByName[typeName]; exists {
			return nil, fmt.Errorf("enum type name collision for %q at %s with generated identifier %s; rename the enum field to avoid generated Go type conflicts", typeName, fieldPath, firstPath)
		}
		if firstPath, exists := enumPathByType[typeName]; exists {
			return nil, fmt.Errorf("enum type name collision for %q between %s and %s; rename one field to avoid generated Go type conflicts", typeName, firstPath, fieldPath)
		}
		if err := validateEnumConstantNames(typeName, fieldPath, field.Values); err != nil {
			return nil, err
		}
		enumPathByType[typeName] = fieldPath
		enumsByType[typeName] = enumDef{typeName: typeName, values: append([]string(nil), field.Values...)}
	}

	for _, event := range reg.Events {
		for _, name := range sortedFieldNames(event.Properties) {
			field := event.Properties[name]
			fieldPath := fmt.Sprintf("events[%s.v%d].properties.%s", event.Name, event.Version, name)
			if err := validateNoNestedEnums(field, fieldPath); err != nil {
				return nil, err
			}
			if field.Type != registry.FieldTypeEnum {
				continue
			}
			typeName := exportedName(name)
			if firstPath, exists := identPathByName[typeName]; exists {
				return nil, fmt.Errorf("enum type name collision for %q at %s with generated identifier %s; rename the enum field to avoid generated Go type conflicts", typeName, fieldPath, firstPath)
			}
			if firstPath, exists := enumPathByType[typeName]; exists {
				return nil, fmt.Errorf("enum type name collision for %q between %s and %s; rename one field to avoid generated Go type conflicts", typeName, firstPath, fieldPath)
			}
			if err := validateEnumConstantNames(typeName, fieldPath, field.Values); err != nil {
				return nil, err
			}
			enumPathByType[typeName] = fieldPath
			enumsByType[typeName] = enumDef{typeName: typeName, values: append([]string(nil), field.Values...)}
		}
	}

	typeNames := make([]string, 0, len(enumsByType))
	for typeName := range enumsByType {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)
	out := make([]enumDef, 0, len(typeNames))
	for _, typeName := range typeNames {
		out = append(out, enumsByType[typeName])
	}
	return out, nil
}

func collectGoTopLevelIdentifiers(reg registry.Registry) map[string]string {
	identifiers := map[string]string{
		"Client":   "type Client",
		"Context":  "type Context",
		"Envelope": "type Envelope",
	}
	for _, event := range reg.Events {
		eventName := exportedName(event.Name) + "V" + fmt.Sprintf("%d", event.Version)
		eventPath := fmt.Sprintf("events[%s.v%d]", event.Name, event.Version)
		identifiers[eventName] = eventPath + " type alias"
		identifiers[eventName+"Properties"] = eventPath + " properties type"
		identifiers["New"+eventName] = eventPath + " constructor"
	}
	return identifiers
}

func validateNoNestedEnums(field registry.Field, fieldPath string) error {
	if field.Type == registry.FieldTypeArray && field.Items != nil {
		if field.Items.Type == registry.FieldTypeEnum {
			return fmt.Errorf("unsupported enum field at %s.items: Go codegen only supports top-level enum fields in context/properties", fieldPath)
		}
		return validateNoNestedEnums(*field.Items, fieldPath+".items")
	}
	if field.Type == registry.FieldTypeObject {
		for _, name := range sortedFieldNames(field.Properties) {
			if err := validateNoNestedEnums(field.Properties[name], fieldPath+".properties."+name); err != nil {
				return err
			}
		}
	}
	return nil
}

func goTypeForField(typePrefix string, field registry.Field, optional bool) string {
	var base string
	switch field.Type {
	case registry.FieldTypeString, registry.FieldTypeUUID, registry.FieldTypeDate:
		base = "string"
	case registry.FieldTypeInteger:
		base = "int"
	case registry.FieldTypeNumber:
		base = "float64"
	case registry.FieldTypeBoolean:
		base = "bool"
	case registry.FieldTypeTimestamp:
		base = "time.Time"
	case registry.FieldTypeEnum:
		base = exportedName(field.Name)
	case registry.FieldTypeArray:
		if field.Items == nil {
			base = "[]any"
		} else {
			base = "[]" + goTypeForField(typePrefix+"Item", *field.Items, false)
		}
	case registry.FieldTypeObject:
		base = "map[string]any"
	default:
		base = "any"
	}
	if optional && (base == "string" || base == "int" || base == "float64" || base == "bool" || strings.HasPrefix(base, "time.")) {
		return "*" + base
	}
	return base
}

func jsonTagSuffix(required bool) string {
	if required {
		return ""
	}
	return ",omitempty"
}
