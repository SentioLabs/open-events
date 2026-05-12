package schemair

import (
	"fmt"
	"sort"

	"github.com/sentiolabs/open-events/internal/registry"
)

func FromRegistry(reg registry.Registry, lock Lock) (Registry, error) {
	// Validate that we have exactly one version
	if len(reg.Events) == 0 {
		return Registry{}, fmt.Errorf("registry has no events; cannot infer version")
	}

	filesByVersion := make(map[int][]registry.Event)
	versions := make([]int, 0, len(reg.Events))
	seenVersions := make(map[int]struct{}, len(reg.Events))
	for _, event := range reg.Events {
		filesByVersion[event.Version] = append(filesByVersion[event.Version], event)
		if _, ok := seenVersions[event.Version]; ok {
			continue
		}
		seenVersions[event.Version] = struct{}{}
		versions = append(versions, event.Version)
	}
	sort.Ints(versions)

	if len(versions) > 1 {
		return Registry{}, fmt.Errorf("registry contains multiple versions (%v); FromRegistry requires exactly one version per file", versions)
	}

	files := make([]File, 0, len(versions))
	for _, version := range versions {
		events := filesByVersion[version]
		sort.Slice(events, func(i, j int) bool {
			if events[i].Name == events[j].Name {
				return events[i].Version < events[j].Version
			}
			return events[i].Name < events[j].Name
		})

		pkg, err := ProtoPackage(reg.Namespace, version)
		if err != nil {
			return Registry{}, err
		}
		filePath, err := ProtoFilePath(reg.Namespace, version)
		if err != nil {
			return Registry{}, err
		}

		context, err := lowerContextMessage(reg.Context, lock)
		if err != nil {
			return Registry{}, err
		}

		// Track generated message names to detect collisions
		messageNames := make(map[string]string) // map[generatedName]eventKey
		messageNames["Client"] = "Client"
		messageNames["Context"] = "Context"

		messages := []Message{clientMessage(), context}
		for _, event := range events {
			// Validate event name is renderable
			if pascalCase(event.Name) == "" {
				return Registry{}, fmt.Errorf("event name %q cannot be rendered as a valid protobuf message name", event.Name)
			}

			// Validate and track envelope message name
			envelopeName := EventMessageName(event)
			if err := isValidProtoMessageName(envelopeName); err != nil {
				return Registry{}, fmt.Errorf("event %q generates invalid message name %q: %w", event.Name, envelopeName, err)
			}
			if existing, exists := messageNames[envelopeName]; exists {
				return Registry{}, fmt.Errorf("message name collision: events %q and %q both generate message name %q", existing, event.Name, envelopeName)
			}
			messageNames[envelopeName] = event.Name

			// Validate and track properties message name
			propsName := PropertiesMessageName(event)
			if err := isValidProtoMessageName(propsName); err != nil {
				return Registry{}, fmt.Errorf("event %q generates invalid properties message name %q: %w", event.Name, propsName, err)
			}
			if existing, exists := messageNames[propsName]; exists {
				return Registry{}, fmt.Errorf("message name collision: events %q and %q both generate message name %q", existing, event.Name, propsName)
			}
			messageNames[propsName] = event.Name

			properties, err := lowerPropertiesMessage(event, lock)
			if err != nil {
				return Registry{}, err
			}
			messages = append(messages, envelopeMessage(event), properties)
		}

		files = append(files, File{Path: filePath, Package: pkg, Messages: messages})
	}

	return Registry{Namespace: reg.Namespace, Files: files}, nil
}

func clientMessage() Message {
	return Message{
		Name: "Client",
		Fields: []Field{
			{
				Name:     "name",
				Number:   1,
				Type:     TypeRef{Scalar: "string"},
				Optional: true,
			},
			{
				Name:     "version",
				Number:   2,
				Type:     TypeRef{Scalar: "string"},
				Optional: true,
			},
		},
	}
}

func lowerContextMessage(context map[string]registry.Field, lock Lock) (Message, error) {
	message := Message{Name: "Context", Fields: make([]Field, 0, len(context)), Enums: []Enum{}}
	usedNumbers := make(map[int]string)      // map[protoNumber]fieldName for duplicate detection
	enumTypeNames := make(map[string]string) // map[enumTypeName]fieldName for collision detection

	for _, name := range sortedRegistryFieldNames(context) {
		field := context[name]

		// Validate field name is a valid protobuf identifier
		if err := isValidProtoIdentifier(name); err != nil {
			return Message{}, fmt.Errorf("context.%s: %w", name, err)
		}

		locked, ok := lock.Context[name]
		if !ok {
			return Message{}, fmt.Errorf("schema lock is missing context.%s", name)
		}

		// Validate StableID matches field name
		if locked.StableID != name {
			return Message{}, fmt.Errorf("schema lock StableID mismatch for context.%s: lock has %q, expected %q", name, locked.StableID, name)
		}

		// Validate proto number
		if err := validateProtoNumber("context."+name, locked.ProtoNumber); err != nil {
			return Message{}, err
		}

		// Check for duplicate numbers
		if existing, exists := usedNumbers[locked.ProtoNumber]; exists {
			return Message{}, fmt.Errorf("context has duplicate proto number %d used by both %q and %q", locked.ProtoNumber, existing, name)
		}
		usedNumbers[locked.ProtoNumber] = name

		lowered, enum, err := lowerField(field, locked.ProtoNumber, "context."+name)
		if err != nil {
			return Message{}, err
		}
		message.Fields = append(message.Fields, lowered)
		if enum != nil {
			// Check for enum type name collision
			if existing, exists := enumTypeNames[enum.Name]; exists {
				return Message{}, fmt.Errorf("context enum type name collision: fields %q and %q both generate enum type %q", existing, name, enum.Name)
			}
			enumTypeNames[enum.Name] = name
			message.Enums = append(message.Enums, *enum)
		}
	}
	return message, nil
}

func lowerPropertiesMessage(event registry.Event, lock Lock) (Message, error) {
	key := eventKey(event)
	lockedEvent, ok := lock.Events[key]
	if !ok {
		return Message{}, fmt.Errorf("schema lock is missing events.%s", key)
	}

	message := Message{Name: PropertiesMessageName(event), Fields: make([]Field, 0, len(event.Properties)), Enums: []Enum{}}
	usedNumbers := make(map[int]string)      // map[protoNumber]fieldName for duplicate detection
	enumTypeNames := make(map[string]string) // map[enumTypeName]fieldName for collision detection

	for _, name := range sortedRegistryFieldNames(event.Properties) {
		field := event.Properties[name]

		// Validate field name is a valid protobuf identifier
		if err := isValidProtoIdentifier(name); err != nil {
			return Message{}, fmt.Errorf("events.%s.properties.%s: %w", key, name, err)
		}

		locked, ok := lockedEvent.Properties[name]
		if !ok {
			return Message{}, fmt.Errorf("schema lock is missing events.%s.properties.%s", key, name)
		}

		// Validate StableID matches field name
		if locked.StableID != name {
			return Message{}, fmt.Errorf("schema lock StableID mismatch for events.%s.properties.%s: lock has %q, expected %q", key, name, locked.StableID, name)
		}

		// Validate proto number
		if err := validateProtoNumber("events."+key+".properties."+name, locked.ProtoNumber); err != nil {
			return Message{}, err
		}

		// Check for duplicate numbers
		if existing, exists := usedNumbers[locked.ProtoNumber]; exists {
			return Message{}, fmt.Errorf("events.%s.properties has duplicate proto number %d used by both %q and %q", key, locked.ProtoNumber, existing, name)
		}
		usedNumbers[locked.ProtoNumber] = name

		lowered, enum, err := lowerField(field, locked.ProtoNumber, "events."+key+".properties."+name)
		if err != nil {
			return Message{}, err
		}
		message.Fields = append(message.Fields, lowered)
		if enum != nil {
			// Check for enum type name collision
			if existing, exists := enumTypeNames[enum.Name]; exists {
				return Message{}, fmt.Errorf("events.%s.properties enum type name collision: fields %q and %q both generate enum type %q", key, existing, name, enum.Name)
			}
			enumTypeNames[enum.Name] = name
			message.Enums = append(message.Enums, *enum)
		}
	}

	return message, nil
}

func envelopeMessage(event registry.Event) Message {
	return Message{
		Name:        EventMessageName(event),
		Description: event.Description,
		Fields: []Field{
			{Name: "event_name", Number: envelopeNumbers["event_name"], Type: TypeRef{Scalar: "string"}},
			{Name: "event_version", Number: envelopeNumbers["event_version"], Type: TypeRef{Scalar: "integer"}},
			{Name: "event_id", Number: envelopeNumbers["event_id"], Type: TypeRef{Scalar: "uuid"}},
			{Name: "event_ts", Number: envelopeNumbers["event_ts"], Type: TypeRef{Scalar: "timestamp"}},
			{Name: "client", Number: envelopeNumbers["client"], Type: TypeRef{Message: "Client"}},
			{Name: "context", Number: envelopeNumbers["context"], Type: TypeRef{Message: "Context"}},
			{Name: "properties", Number: envelopeNumbers["properties"], Type: TypeRef{Message: PropertiesMessageName(event)}},
		},
	}
}

func lowerField(field registry.Field, number int, path string) (Field, *Enum, error) {
	lowered := Field{
		Name:        field.Name,
		Number:      number,
		Required:    field.Required,
		Description: field.Description,
	}

	switch field.Type {
	case registry.FieldTypeString:
		lowered.Type = TypeRef{Scalar: "string"}
		lowered.Optional = true
	case registry.FieldTypeInteger:
		lowered.Type = TypeRef{Scalar: "integer"}
		lowered.Optional = true
	case registry.FieldTypeNumber:
		lowered.Type = TypeRef{Scalar: "number"}
		lowered.Optional = true
	case registry.FieldTypeBoolean:
		lowered.Type = TypeRef{Scalar: "boolean"}
		lowered.Optional = true
	case registry.FieldTypeTimestamp:
		lowered.Type = TypeRef{Scalar: "timestamp"}
		lowered.Optional = true
	case registry.FieldTypeDate:
		lowered.Type = TypeRef{Scalar: "date"}
		lowered.Optional = true
	case registry.FieldTypeUUID:
		lowered.Type = TypeRef{Scalar: "uuid"}
		lowered.Optional = true
	case registry.FieldTypeEnum:
		enumName := EnumTypeName(field.Name)
		if enumName == "" {
			return Field{}, nil, fmt.Errorf("%s: field name %q cannot be rendered as a valid enum type name", path, field.Name)
		}
		if err := isValidProtoMessageName(enumName); err != nil {
			return Field{}, nil, fmt.Errorf("%s: enum type name %q is invalid: %w", path, enumName, err)
		}
		values, err := buildEnumValues(enumName, field.Values, path)
		if err != nil {
			return Field{}, nil, err
		}
		lowered.Type = TypeRef{Enum: enumName}
		lowered.Optional = true
		return lowered, &Enum{Name: enumName, Values: values}, nil
	case registry.FieldTypeArray:
		if field.Items == nil {
			return Field{}, nil, fmt.Errorf("%s.items: array fields must define items", path)
		}
		if field.Items.Type == registry.FieldTypeObject {
			return Field{}, nil, fmt.Errorf("%s.items: array of object is not supported", path)
		}
		if field.Items.Type == registry.FieldTypeEnum {
			return Field{}, nil, fmt.Errorf("%s.items: array of enum is not supported", path)
		}
		if field.Items.Type == registry.FieldTypeArray {
			return Field{}, nil, fmt.Errorf("%s.items: array of array is not supported", path)
		}
		item := *field.Items
		item.Name = field.Name
		loweredItem, _, err := lowerField(item, number, path+".items")
		if err != nil {
			return Field{}, nil, err
		}
		lowered.Type = loweredItem.Type
		lowered.Repeated = true
		lowered.Optional = false
	case registry.FieldTypeObject:
		return Field{}, nil, fmt.Errorf("%s: object fields are not supported", path)
	default:
		return Field{}, nil, fmt.Errorf("%s: unsupported field type %q", path, field.Type)
	}

	return lowered, nil, nil
}

func sortedRegistryFieldNames(fields map[string]registry.Field) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
