package schemair

import (
	"fmt"
	"go/token"
	"sort"
	"strings"

	"github.com/sentiolabs/open-events/internal/registry"
)

func FromRegistry(reg registry.Registry, lock Lock) (Registry, error) {
	if err := validateLockForLowering(reg, lock); err != nil {
		return Registry{}, err
	}

	if len(reg.Events) == 0 {
		return Registry{}, fmt.Errorf("registry has no events; cannot infer version")
	}

	version := reg.Events[0].Version
	for _, event := range reg.Events {
		if event.Version != version {
			return Registry{}, fmt.Errorf("registry contains multiple versions (saw %d and %d); FromRegistry requires exactly one version per file", version, event.Version)
		}
	}

	if err := validateGoPackage(reg.Package.Go); err != nil {
		return Registry{}, err
	}

	events := append([]registry.Event(nil), reg.Events...)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})

	// Build per-domain specs.
	domainSpecs, err := buildDomainSpecs(reg, lock, version)
	if err != nil {
		return Registry{}, err
	}

	// Build the legacy single-file output (for backward compatibility with render.go).
	files := make([]File, 0, 1)
	{
		pkg, err := ProtoPackage(reg.Namespace, version)
		if err != nil {
			return Registry{}, err
		}
		filePath, err := ProtoFilePath(reg.Namespace, version)
		if err != nil {
			return Registry{}, err
		}

		messageNames := map[string]string{"Client": "Client", "Context": "Context"}

		messages := []Message{clientMessage()}

		// For the per-domain shape, context lives in DomainSpecs. For legacy single-file
		// output (no domains), emit an empty Context message to maintain backward compat.
		if len(reg.Domains) == 0 {
			messages = append(messages, Message{Name: "Context", Fields: []Field{}, Enums: []Enum{}})
		}

		for _, event := range events {
			// Validate event name before case conversion
			if err := validateEventName(event.Name); err != nil {
				return Registry{}, fmt.Errorf("event name %q is invalid: %w", event.Name, err)
			}

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

			// Determine the context type name for this event's domain.
			contextTypeName := contextMessageName(event.Domain)
			messages = append(messages, envelopeMessageWithContext(event, contextTypeName), properties)
		}

		files = append(files, File{Path: filePath, Package: pkg, GoPackage: reg.Package.Go, Messages: messages})
	}

	return Registry{
		Namespace:   reg.Namespace,
		GoPackage:   reg.Package.Go,
		Files:       files,
		DomainSpecs: domainSpecs,
		CommonSpec:  CommonSpec{Client: clientMessage()},
	}, nil
}

// buildDomainSpecs constructs one DomainSpec per domain in reg, ordered alphabetically.
func buildDomainSpecs(reg registry.Registry, lock Lock, version int) ([]DomainSpec, error) {
	bundles := GroupByDomain(reg)
	specs := make([]DomainSpec, 0, len(bundles))

	for _, bundle := range bundles {
		domainName := bundle.Domain.Name
		lockedDomain, _ := lock.Domains[domainName]

		// Lower context fields for this domain.
		contextName := contextMessageName(domainName)
		contextFields, contextEnums, err := lowerDomainContext(domainName, bundle.Domain.Context, lockedDomain.Context)
		if err != nil {
			return nil, err
		}

		// Sort events for determinism.
		domainEvents := append([]registry.Event(nil), bundle.Events...)
		sort.Slice(domainEvents, func(i, j int) bool {
			return domainEvents[i].Name < domainEvents[j].Name
		})

		// Validate message name uniqueness within this domain.
		messageNames := map[string]string{
			contextName: "<context>",
		}

		events := make([]DomainEvent, 0, len(domainEvents))
		for _, event := range domainEvents {
			if err := validateEventName(event.Name); err != nil {
				return nil, fmt.Errorf("event name %q is invalid: %w", event.Name, err)
			}
			if pascalCase(event.Name) == "" {
				return nil, fmt.Errorf("event name %q cannot be rendered as a valid protobuf message name", event.Name)
			}

			envelopeName := EventMessageName(event)
			if err := isValidProtoMessageName(envelopeName); err != nil {
				return nil, fmt.Errorf("event %q generates invalid message name %q: %w", event.Name, envelopeName, err)
			}
			if existing, exists := messageNames[envelopeName]; exists {
				return nil, fmt.Errorf("message name collision: events %q and %q both generate message name %q", existing, event.Name, envelopeName)
			}
			messageNames[envelopeName] = event.Name

			propsName := PropertiesMessageName(event)
			if err := isValidProtoMessageName(propsName); err != nil {
				return nil, fmt.Errorf("event %q generates invalid properties message name %q: %w", event.Name, propsName, err)
			}
			if existing, exists := messageNames[propsName]; exists {
				return nil, fmt.Errorf("message name collision: events %q and %q both generate message name %q", existing, event.Name, propsName)
			}
			messageNames[propsName] = event.Name

			properties, err := lowerPropertiesMessage(event, lock)
			if err != nil {
				return nil, err
			}

			events = append(events, DomainEvent{
				Envelope:   envelopeMessageWithContext(event, contextName),
				Properties: properties,
			})
		}

		specs = append(specs, DomainSpec{
			Name:          domainName,
			ContextName:   contextName,
			ContextFields: contextFields,
			ContextEnums:  contextEnums,
			Events:        events,
		})
	}

	return specs, nil
}

// contextMessageName returns the PascalCase Context message name for a domain.
// For domain "user", returns "UserContext". For empty domain, returns "Context".
func contextMessageName(domain string) string {
	if domain == "" {
		return "Context"
	}
	return pascalCase(domain) + "Context"
}

// lowerDomainContext lowers the context fields for one domain using the domain's lock entries.
func lowerDomainContext(domainName string, contextFields map[string]registry.Field, lockedContext map[string]LockedField) ([]Field, []Enum, error) {
	if len(contextFields) == 0 {
		return []Field{}, []Enum{}, nil
	}

	if len(lockedContext) == 0 {
		// Missing lock entries for all context fields.
		fieldNames := make([]string, 0, len(contextFields))
		for name := range contextFields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)
		return nil, nil, fmt.Errorf("schema lock is missing context.%s", fieldNames[0])
	}

	fields := make([]Field, 0, len(contextFields))
	enums := make([]Enum, 0)
	usedNumbers := make(map[int]string)
	enumTypeNames := make(map[string]string)
	enumValueNames := make(map[string]string)

	for _, name := range sortedRegistryFieldNames(contextFields) {
		field := contextFields[name]

		if err := isValidProtoIdentifier(name); err != nil {
			return nil, nil, fmt.Errorf("context.%s: %w", name, err)
		}

		locked, ok := lockedContext[name]
		if !ok {
			return nil, nil, fmt.Errorf("schema lock is missing context.%s", name)
		}

		if locked.StableID != name {
			return nil, nil, fmt.Errorf("schema lock StableID mismatch for context.%s: lock has %q, expected %q", name, locked.StableID, name)
		}

		if err := validateProtoNumber("context."+name, locked.ProtoNumber); err != nil {
			return nil, nil, err
		}

		if existing, exists := usedNumbers[locked.ProtoNumber]; exists {
			return nil, nil, fmt.Errorf("context has duplicate proto number %d used by both %q and %q", locked.ProtoNumber, existing, name)
		}
		usedNumbers[locked.ProtoNumber] = name

		lowered, enum, _, err := lowerField(field, locked.ProtoNumber, "context."+name)
		if err != nil {
			return nil, nil, err
		}
		fields = append(fields, lowered)
		if enum != nil {
			if existing, exists := enumTypeNames[enum.Name]; exists {
				return nil, nil, fmt.Errorf("context enum type name collision: fields %q and %q both generate enum type %q", existing, name, enum.Name)
			}
			enumTypeNames[enum.Name] = name

			zeroValueName := EnumZeroValueName(enum.Name)
			if existing, exists := enumValueNames[zeroValueName]; exists {
				return nil, nil, fmt.Errorf("context enum value collision: field %q zero value %q conflicts with %s", name, zeroValueName, existing)
			}
			enumValueNames[zeroValueName] = fmt.Sprintf("field %q zero value", name)

			for _, val := range enum.Values {
				if existing, exists := enumValueNames[val.Name]; exists {
					return nil, nil, fmt.Errorf("context enum value collision: field %q value %q (from %q) conflicts with %s", name, val.Name, val.Original, existing)
				}
				enumValueNames[val.Name] = fmt.Sprintf("field %q value %q", name, val.Original)
			}

			enums = append(enums, *enum)
		}
	}

	// Validate no stale lock entries for this domain's context.
	for _, name := range sortedLockedFieldNames(lockedContext) {
		if _, ok := contextFields[name]; !ok {
			return nil, nil, fmt.Errorf("schema lock has stale context entry %q not in domain %q context", name, domainName)
		}
	}

	return fields, enums, nil
}

func validateGoPackage(goPackage string) error {
	if goPackage == "" {
		return nil
	}
	if !strings.Contains(goPackage, ".") && !strings.Contains(goPackage, "/") {
		return fmt.Errorf("package.go must include at least one '.' or '/' in the import path")
	}
	parts := strings.Split(goPackage, "/")
	base := parts[len(parts)-1]
	if token.Lookup(base).IsKeyword() {
		return fmt.Errorf("package.go basename %q must not be a Go keyword", base)
	}
	return nil
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

func lowerPropertiesMessage(event registry.Event, lock Lock) (Message, error) {
	key := eventKey(event)
	lockedEvent, ok := lock.Events[key]
	if !ok {
		return Message{}, fmt.Errorf("schema lock is missing events.%s", key)
	}

	message := Message{Name: PropertiesMessageName(event), Fields: make([]Field, 0, len(event.Properties)), Enums: []Enum{}}
	usedNumbers := make(map[int]string)       // map[protoNumber]fieldName for duplicate detection
	enumTypeNames := make(map[string]string)  // map[enumTypeName]fieldName for collision detection
	enumValueNames := make(map[string]string) // map[renderedValueName]source for value collision detection

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

		lowered, enum, nestedMsg, err := lowerFieldLocked(field, locked, "events."+key+".properties."+name)
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

			// Check for enum value name collisions across all enums in this message
			// Reserve the synthesized zero value name
			zeroValueName := EnumZeroValueName(enum.Name)
			if existing, exists := enumValueNames[zeroValueName]; exists {
				return Message{}, fmt.Errorf("events.%s.properties enum value collision: field %q zero value %q conflicts with %s", key, name, zeroValueName, existing)
			}
			enumValueNames[zeroValueName] = fmt.Sprintf("field %q zero value", name)

			// Reserve all authored value names
			for _, val := range enum.Values {
				if existing, exists := enumValueNames[val.Name]; exists {
					return Message{}, fmt.Errorf("events.%s.properties enum value collision: field %q value %q (from %q) conflicts with %s", key, name, val.Name, val.Original, existing)
				}
				enumValueNames[val.Name] = fmt.Sprintf("field %q value %q", name, val.Original)
			}

			message.Enums = append(message.Enums, *enum)
		}
		if nestedMsg != nil {
			message.NestedMessages = append(message.NestedMessages, *nestedMsg)
		}
	}

	return message, nil
}

// envelopeMessageWithContext builds an envelope message where the context field
// references the given contextTypeName (e.g. "UserContext").
func envelopeMessageWithContext(event registry.Event, contextTypeName string) Message {
	return Message{
		Name:        EventMessageName(event),
		Description: event.Description,
		Fields: []Field{
			{Name: "event_name", Number: envelopeNumbers["event_name"], Type: TypeRef{Scalar: "string"}},
			{Name: "event_version", Number: envelopeNumbers["event_version"], Type: TypeRef{Scalar: "integer"}},
			{Name: "event_id", Number: envelopeNumbers["event_id"], Type: TypeRef{Scalar: "uuid"}},
			{Name: "event_ts", Number: envelopeNumbers["event_ts"], Type: TypeRef{Scalar: "timestamp"}},
			{Name: "client", Number: envelopeNumbers["client"], Type: TypeRef{Message: "Client"}},
			{Name: "context", Number: envelopeNumbers["context"], Type: TypeRef{Message: contextTypeName}},
			{Name: "properties", Number: envelopeNumbers["properties"], Type: TypeRef{Message: PropertiesMessageName(event)}},
		},
	}
}

func envelopeMessage(event registry.Event) Message {
	return envelopeMessageWithContext(event, "Context")
}

func lowerField(field registry.Field, number int, path string) (Field, *Enum, *Message, error) {
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
			return Field{}, nil, nil, fmt.Errorf("%s: field name %q cannot be rendered as a valid enum type name", path, field.Name)
		}
		if err := isValidProtoMessageName(enumName); err != nil {
			return Field{}, nil, nil, fmt.Errorf("%s: enum type name %q is invalid: %w", path, enumName, err)
		}
		values, err := buildEnumValues(enumName, field.Values, path)
		if err != nil {
			return Field{}, nil, nil, err
		}
		lowered.Type = TypeRef{Enum: enumName}
		lowered.Optional = true
		return lowered, &Enum{Name: enumName, Values: values}, nil, nil
	case registry.FieldTypeArray:
		if field.Items == nil {
			return Field{}, nil, nil, fmt.Errorf("%s.items: array fields must define items", path)
		}
		if field.Items.Type == registry.FieldTypeObject {
			// Generate a nested message for the array element type.
			nestedMsgName := pascalCase(field.Name)
			if nestedMsgName == "" {
				return Field{}, nil, nil, fmt.Errorf("%s: field name %q cannot be rendered as a valid nested message name", path, field.Name)
			}
			nestedMsg, err := lowerObjectToMessage(nestedMsgName, field.Items.Properties, path+".items")
			if err != nil {
				return Field{}, nil, nil, err
			}
			lowered.Type = TypeRef{Message: nestedMsgName}
			lowered.Repeated = true
			lowered.Optional = false
			return lowered, nil, nestedMsg, nil
		}
		if field.Items.Type == registry.FieldTypeEnum {
			return Field{}, nil, nil, fmt.Errorf("%s.items: array of enum is not supported", path)
		}
		if field.Items.Type == registry.FieldTypeArray {
			return Field{}, nil, nil, fmt.Errorf("%s.items: array of array is not supported", path)
		}
		item := *field.Items
		item.Name = field.Name
		loweredItem, _, _, err := lowerField(item, number, path+".items")
		if err != nil {
			return Field{}, nil, nil, err
		}
		lowered.Type = loweredItem.Type
		lowered.Repeated = true
		lowered.Optional = false
	case registry.FieldTypeObject:
		// Generate a nested message for the object type.
		nestedMsgName := pascalCase(field.Name)
		if nestedMsgName == "" {
			return Field{}, nil, nil, fmt.Errorf("%s: field name %q cannot be rendered as a valid nested message name", path, field.Name)
		}
		if len(field.Properties) == 0 {
			return Field{}, nil, nil, fmt.Errorf("%s: object fields must define properties", path)
		}
		nestedMsg, err := lowerObjectToMessage(nestedMsgName, field.Properties, path)
		if err != nil {
			return Field{}, nil, nil, err
		}
		lowered.Type = TypeRef{Message: nestedMsgName}
		lowered.Optional = true
		return lowered, nil, nestedMsg, nil
	default:
		return Field{}, nil, nil, fmt.Errorf("%s: unsupported field type %q", path, field.Type)
	}

	return lowered, nil, nil, nil
}

// lowerFieldLocked is like lowerField but uses the full LockedField (including
// nested Properties for object sub-fields) so nested object subfield numbers
// come from the lock rather than sorted order.
func lowerFieldLocked(field registry.Field, locked LockedField, path string) (Field, *Enum, *Message, error) {
	if field.Type == registry.FieldTypeObject && len(field.Properties) > 0 {
		nestedMsgName := pascalCase(field.Name)
		if nestedMsgName == "" {
			return Field{}, nil, nil, fmt.Errorf("%s: field name %q cannot be rendered as a valid nested message name", path, field.Name)
		}
		nestedMsg, err := lowerObjectToMessageLocked(nestedMsgName, field.Properties, locked.Properties, path)
		if err != nil {
			return Field{}, nil, nil, err
		}
		lowered := Field{
			Name:        field.Name,
			Number:      locked.ProtoNumber,
			Required:    field.Required,
			Description: field.Description,
			Type:        TypeRef{Message: nestedMsgName},
			Optional:    true,
		}
		return lowered, nil, nestedMsg, nil
	}
	return lowerField(field, locked.ProtoNumber, path)
}

// lowerObjectToMessageLocked builds a nested Message from an object's sub-properties,
// reading proto numbers from lockedSubFields when available.
// If lockedSubFields is nil or a subfield is not in the lock, sequential numbers
// starting at 1 are used (backward-compatible with pre-existing lockfiles).
func lowerObjectToMessageLocked(msgName string, properties map[string]registry.Field, lockedSubFields map[string]LockedField, path string) (*Message, error) {
	names := sortedRegistryFieldNames(properties)
	msg := &Message{
		Name:   msgName,
		Fields: make([]Field, 0, len(names)),
		Enums:  []Enum{},
	}
	for i, name := range names {
		field := properties[name]
		if err := isValidProtoIdentifier(name); err != nil {
			return nil, fmt.Errorf("%s.%s: %w", path, name, err)
		}
		var number int
		var subLocked LockedField
		if lockedSubFields != nil {
			if l, ok := lockedSubFields[name]; ok {
				number = l.ProtoNumber
				subLocked = l
			}
		}
		if number == 0 {
			// Backward compat: no lock entry for this subfield — use sorted order.
			number = i + 1
		}
		var lowered Field
		var enum *Enum
		var err error
		if field.Type == registry.FieldTypeObject && len(field.Properties) > 0 {
			lowered, enum, _, err = lowerFieldLocked(field, LockedField{ProtoNumber: number, StableID: name, Properties: subLocked.Properties}, path+"."+name)
		} else {
			lowered, enum, _, err = lowerField(field, number, path+"."+name)
		}
		if err != nil {
			return nil, err
		}
		msg.Fields = append(msg.Fields, lowered)
		if enum != nil {
			msg.Enums = append(msg.Enums, *enum)
		}
	}
	return msg, nil
}

// lowerObjectToMessage builds a nested Message from an object's sub-properties.
// Proto numbers are assigned sequentially starting at 1, ordered by sorted field name.
// Enum sub-fields are collected into the nested message's Enums slice.
func lowerObjectToMessage(msgName string, properties map[string]registry.Field, path string) (*Message, error) {
	return lowerObjectToMessageLocked(msgName, properties, nil, path)
}

func sortedRegistryFieldNames(fields map[string]registry.Field) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func validateLockForLowering(reg registry.Registry, lock Lock) error {
	// Validate lock version
	if lock.Version != LockVersion {
		return fmt.Errorf("schema lock version mismatch: got %d want %d", lock.Version, LockVersion)
	}

	// Validate context lock entries per domain
	if err := validateContextLock(reg, lock); err != nil {
		return err
	}

	// Validate event lock entries
	for _, event := range reg.Events {
		key := eventKey(event)
		if err := validateEventLock(reg, lock, event, key); err != nil {
			return err
		}
	}

	// Check for stale extra lock entries not in registry
	if err := validateNoStaleLockEntries(reg, lock); err != nil {
		return err
	}

	return nil
}

// validateContextLock validates per-domain context lock entries in lock.Domains.
func validateContextLock(reg registry.Registry, lock Lock) error {
	for domainName, domain := range reg.Domains {
		lockedDomain, _ := lock.Domains[domainName]

		for _, name := range sortedRegistryFieldNames(domain.Context) {
			locked, ok := lockedDomain.Context[name]
			if !ok {
				// Missing lock entry handled by lowerDomainContext; skip here to avoid
				// duplicate error messages.
				continue
			}

			// Validate proto number
			if err := validateProtoNumber("context."+name, locked.ProtoNumber); err != nil {
				return err
			}

			// Validate StableID
			if locked.StableID != name {
				return fmt.Errorf("schema lock StableID mismatch for context.%s: lock has %q, expected %q", name, locked.StableID, name)
			}
		}

		// Check for duplicate proto numbers in this domain's context.
		byNumber := make(map[int]string)
		if lockedDomain.Context != nil {
			for _, name := range sortedLockedFieldNames(lockedDomain.Context) {
				locked := lockedDomain.Context[name]
				if existing, exists := byNumber[locked.ProtoNumber]; exists {
					return fmt.Errorf("context has duplicate proto number %d used by both %q and %q", locked.ProtoNumber, existing, name)
				}
				byNumber[locked.ProtoNumber] = name
			}
		}
	}

	return nil
}

func validateEventLock(reg registry.Registry, lock Lock, event registry.Event, key string) error {
	lockedEvent, ok := lock.Events[key]
	if !ok {
		// This is handled elsewhere with more specific error
		return nil
	}

	// Validate envelope lock entries when present
	if err := validateEnvelopeLock(key, lockedEvent); err != nil {
		return err
	}

	// Validate properties lock entries
	if err := validatePropertiesLock(event, key, lockedEvent); err != nil {
		return err
	}

	// Validate reserved entries
	if err := validateReservedEntries(key, lockedEvent); err != nil {
		return err
	}

	return nil
}

func validateEnvelopeLock(key string, lockedEvent LockedEvent) error {
	// Envelope entries are optional, but when present they must be valid
	if len(lockedEvent.Envelope) == 0 {
		return nil
	}

	// Track proto numbers to detect duplicates
	byNumber := make(map[int]string)

	for _, name := range sortedLockedFieldNames(lockedEvent.Envelope) {
		locked := lockedEvent.Envelope[name]
		// Validate the envelope key is a known fixed envelope field
		expectedNumber, ok := envelopeNumbers[name]
		if !ok {
			return fmt.Errorf("schema lock has unexpected envelope key at events.%s.envelope.%s: not a valid envelope field", key, name)
		}

		// Validate proto number
		if err := validateProtoNumber("events."+key+".envelope."+name, locked.ProtoNumber); err != nil {
			return err
		}

		// Validate proto number matches the fixed envelope number
		if locked.ProtoNumber != expectedNumber {
			return fmt.Errorf("schema lock envelope proto number mismatch for events.%s.envelope.%s: lock has %d, expected %d", key, name, locked.ProtoNumber, expectedNumber)
		}

		// Validate StableID matches field name
		if locked.StableID != name {
			return fmt.Errorf("schema lock StableID mismatch for events.%s.envelope.%s: lock has %q, expected %q", key, name, locked.StableID, name)
		}

		// Check for duplicate proto numbers
		if existing, exists := byNumber[locked.ProtoNumber]; exists {
			return fmt.Errorf("events.%s.envelope has duplicate proto number %d used by both %q and %q", key, locked.ProtoNumber, existing, name)
		}
		byNumber[locked.ProtoNumber] = name
	}

	return nil
}

func validatePropertiesLock(event registry.Event, key string, lockedEvent LockedEvent) error {
	// Validate all active property fields have valid lock entries
	for _, name := range sortedRegistryFieldNames(event.Properties) {
		locked, ok := lockedEvent.Properties[name]
		if !ok {
			// This is handled elsewhere with more specific error
			continue
		}

		// Validate proto number
		if err := validateProtoNumber("events."+key+".properties."+name, locked.ProtoNumber); err != nil {
			return err
		}

		// Validate StableID
		if locked.StableID != name {
			return fmt.Errorf("schema lock StableID mismatch for events.%s.properties.%s: lock has %q, expected %q", key, name, locked.StableID, name)
		}
	}

	// Check for duplicate proto numbers in properties and reserved
	byNumber := make(map[int]string)
	for _, name := range sortedLockedFieldNames(lockedEvent.Properties) {
		locked := lockedEvent.Properties[name]
		if existing, exists := byNumber[locked.ProtoNumber]; exists {
			return fmt.Errorf("events.%s.properties has duplicate proto number %d used by both %q and %q", key, locked.ProtoNumber, existing, name)
		}
		byNumber[locked.ProtoNumber] = name
	}
	for _, reserved := range lockedEvent.Reserved {
		if existing, exists := byNumber[reserved.ProtoNumber]; exists {
			return fmt.Errorf("events.%s.properties/reserved has duplicate proto number %d used by both %q and %q", key, reserved.ProtoNumber, existing, reserved.Name)
		}
		byNumber[reserved.ProtoNumber] = reserved.Name
	}

	return nil
}

func validateReservedEntries(key string, lockedEvent LockedEvent) error {
	for _, reserved := range lockedEvent.Reserved {
		path := "events." + key + ".reserved." + reserved.Name

		// Validate name is non-empty
		if reserved.Name == "" {
			return fmt.Errorf("schema lock has invalid reserved field at events.%s.reserved: name must be non-empty", key)
		}

		// Validate proto number
		if err := validateProtoNumber(path, reserved.ProtoNumber); err != nil {
			return err
		}

		// Validate StableID matches name
		if reserved.StableID != reserved.Name {
			return fmt.Errorf("schema lock StableID mismatch for %s: lock has %q, expected %q", path, reserved.StableID, reserved.Name)
		}

		// Validate reason
		if reserved.Reason != reservedFieldReasonRemoved {
			return fmt.Errorf("schema lock has invalid reserved reason at %s: got %q want %q", path, reserved.Reason, reservedFieldReasonRemoved)
		}
	}

	return nil
}

func validateNoStaleLockEntries(reg registry.Registry, lock Lock) error {
	// Build map of registry events for quick lookup
	regEvents := make(map[string]registry.Event)
	for _, event := range reg.Events {
		key := eventKey(event)
		regEvents[key] = event
	}

	// Check for stale event entries
	for _, key := range sortedLockedEventKeys(lock.Events) {
		lockedEvent := lock.Events[key]
		event, ok := regEvents[key]
		if !ok {
			return fmt.Errorf("schema lock has stale event entry %q not in registry", key)
		}

		// Check for stale property entries
		for _, name := range sortedLockedFieldNames(lockedEvent.Properties) {
			if _, ok := event.Properties[name]; !ok {
				return fmt.Errorf("schema lock has stale property entry events.%s.properties.%s not in registry", key, name)
			}
		}
	}

	return nil
}
