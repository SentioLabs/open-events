package schemair

type Registry struct {
	Namespace   string
	Files       []File
	DomainSpecs []DomainSpec
	CommonSpec  CommonSpec
}

// DomainSpec carries the per-domain context fields and events for proto emission.
type DomainSpec struct {
	// Name is the domain name (e.g. "user", "order").
	Name string
	// ContextName is the PascalCase Context message name (e.g. "UserContext").
	ContextName string
	// ContextFields holds the lowered context fields for this domain.
	ContextFields []Field
	// ContextEnums holds the enum types nested in the context message.
	ContextEnums []Enum
	// Events holds the lowered event messages (envelope + properties pairs).
	Events []DomainEvent
}

// DomainEvent holds the envelope and properties messages for a single event.
type DomainEvent struct {
	// Envelope is the top-level event message (e.g. CheckoutCompletedV1).
	Envelope Message
	// Properties is the nested properties message (e.g. CheckoutCompletedV1Properties).
	Properties Message
}

// CommonSpec carries the shared types emitted into common.proto.
type CommonSpec struct {
	// Client is the shared Client message.
	Client Message
}

type File struct {
	Path      string
	Package   string
	GoPackage string
	Messages  []Message
}

type Message struct {
	Name           string
	Description    string
	Fields         []Field
	Enums          []Enum
	NestedMessages []Message
}

type Field struct {
	Name        string
	Number      int
	Type        TypeRef
	Repeated    bool
	Optional    bool
	Required    bool
	Description string
}

type Enum struct {
	Name   string
	Values []EnumValue
}

type EnumValue struct {
	Name     string
	Original string
	Number   int
}

type TypeRef struct {
	Scalar  string
	Message string
	Enum    string
}
