package codegen

import (
	"fmt"
	"strings"

	"github.com/sentiolabs/open-events/internal/registry"
)

var pythonKeywords = map[string]struct{}{
	"False": {}, "None": {}, "True": {}, "and": {}, "as": {}, "assert": {}, "async": {}, "await": {},
	"break": {}, "class": {}, "continue": {}, "def": {}, "del": {}, "elif": {}, "else": {}, "except": {},
	"finally": {}, "for": {}, "from": {}, "global": {}, "if": {}, "import": {}, "in": {}, "is": {},
	"lambda": {}, "nonlocal": {}, "not": {}, "or": {}, "pass": {}, "raise": {}, "return": {}, "try": {},
	"while": {}, "with": {}, "yield": {}, "match": {}, "case": {},
}

func renderPython(reg registry.Registry) (string, error) {
	if reg.Package.Python == "" {
		return "", fmt.Errorf("package.python is required")
	}
	if err := validatePythonFieldNames(reg); err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("from dataclasses import dataclass\n")
	b.WriteString("from datetime import datetime\n")
	b.WriteString("from typing import Any, Optional\n\n")
	b.WriteString("@dataclass\nclass Client:\n    name: str\n    version: str\n\n")
	b.WriteString("@dataclass\nclass Context:\n")
	for _, name := range sortedFieldNames(reg.Context) {
		field := reg.Context[name]
		b.WriteString("    " + name + ": " + pyTypeForField(field, !field.Required) + "\n")
	}
	b.WriteString("\n")

	for _, event := range reg.Events {
		eventName := pythonClassName(event.Name) + "V" + fmt.Sprintf("%d", event.Version)
		propsName := eventName + "Properties"
		b.WriteString("@dataclass\nclass " + propsName + ":\n")
		for _, name := range sortedFieldNames(event.Properties) {
			field := event.Properties[name]
			b.WriteString("    " + name + ": " + pyTypeForField(field, !field.Required) + "\n")
		}
		b.WriteString("\n")
		b.WriteString("@dataclass\nclass " + eventName + ":\n")
		b.WriteString("    event_name: str\n")
		b.WriteString("    event_version: int\n")
		b.WriteString("    event_id: str\n")
		b.WriteString("    event_ts: datetime\n")
		b.WriteString("    client: Client\n")
		b.WriteString("    context: Context\n")
		b.WriteString("    properties: " + propsName + "\n\n")
	}

	b.WriteString("def decode_event(raw: dict[str, Any]):\n")
	b.WriteString("    event_name = raw['event_name']\n")
	b.WriteString("    event_version = raw['event_version']\n")
	b.WriteString("    context_raw = raw['context']\n")
	b.WriteString("    client = Client(name=raw['client']['name'], version=raw['client']['version'])\n")
	b.WriteString("    context = Context(\n")
	for _, name := range sortedFieldNames(reg.Context) {
		b.WriteString("        " + name + "=context_raw.get('" + name + "'),\n")
	}
	b.WriteString("    )\n")

	for i, event := range reg.Events {
		prefix := "if"
		if i > 0 {
			prefix = "elif"
		}
		eventName := pythonClassName(event.Name) + "V" + fmt.Sprintf("%d", event.Version)
		propsName := eventName + "Properties"
		b.WriteString("    " + prefix + " event_name == '" + event.Name + "' and event_version == " + fmt.Sprintf("%d", event.Version) + ":\n")
		b.WriteString("        properties_raw = raw['properties']\n")
		b.WriteString("        properties = " + propsName + "(\n")
		for _, name := range sortedFieldNames(event.Properties) {
			b.WriteString("            " + name + "=properties_raw.get('" + name + "'),\n")
		}
		b.WriteString("        )\n")
		b.WriteString("        return " + eventName + "(\n")
		b.WriteString("            event_name=event_name,\n")
		b.WriteString("            event_version=event_version,\n")
		b.WriteString("            event_id=raw['event_id'],\n")
		b.WriteString("            event_ts=datetime.fromisoformat(raw['event_ts'].replace('Z', '+00:00')),\n")
		b.WriteString("            client=client,\n")
		b.WriteString("            context=context,\n")
		b.WriteString("            properties=properties,\n")
		b.WriteString("        )\n")
	}
	b.WriteString("    raise ValueError(f'unsupported event: {event_name} v{event_version}')\n")

	return b.String(), nil
}

func validatePythonFieldNames(reg registry.Registry) error {
	for _, name := range sortedFieldNames(reg.Context) {
		if _, keyword := pythonKeywords[name]; keyword {
			return fmt.Errorf("python generation does not support context field name %q because it is a reserved Python keyword; rename the field", name)
		}
	}
	for _, event := range reg.Events {
		for _, name := range sortedFieldNames(event.Properties) {
			if _, keyword := pythonKeywords[name]; keyword {
				return fmt.Errorf("python generation does not support field %q in event %q v%d because it is a reserved Python keyword; rename the field", name, event.Name, event.Version)
			}
		}
	}
	return nil
}

func pyTypeForField(field registry.Field, optional bool) string {
	var base string
	switch field.Type {
	case registry.FieldTypeString, registry.FieldTypeUUID, registry.FieldTypeDate, registry.FieldTypeEnum:
		base = "str"
	case registry.FieldTypeInteger:
		base = "int"
	case registry.FieldTypeNumber:
		base = "float"
	case registry.FieldTypeBoolean:
		base = "bool"
	case registry.FieldTypeTimestamp:
		base = "datetime"
	case registry.FieldTypeArray:
		if field.Items == nil {
			base = "list[Any]"
		} else {
			base = "list[" + pyTypeForField(*field.Items, false) + "]"
		}
	case registry.FieldTypeObject:
		base = "dict[str, Any]"
	default:
		base = "Any"
	}
	if optional {
		return "Optional[" + base + "]"
	}
	return base
}
