package registry

const SupportedVersion = "0.1.0"

type Registry struct {
	Version   string
	Namespace string
	Package   PackageConfig
	Owners    []Owner
	Context   map[string]Field
	Events    []Event
	Domains   map[string]Domain
	Codegen   Codegen
}

type PackageConfig struct {
	// Go is the import path of the per-domain Go bindings package emitted by
	// internal/codegen/golang (e.g. "github.com/acme/foo/eventmap"). Services
	// import this for event_names.go, context.go, and *_request.go.
	Go string

	// Python is the package name of the per-domain Python bindings emitted by
	// internal/codegen/python (e.g. "consumer"). Modules live under
	// `<Python>/event_names/` and `<Python>/context/`.
	Python string

	// ProtoGoModule is the Go module path under which buf-generated *.pb.go
	// files live (e.g. "github.com/acme/foo/gen/go"). Optional. If set,
	// protogen emits `option go_package = "<ProtoGoModule>/<namespacePath>/<domain>/v1"`
	// matching what buf produces with `paths=source_relative`. If empty,
	// protogen falls back to the legacy `<Go>/pb/<domain>` convention for
	// backward compatibility with monorepo layouts that colocate proto with
	// the consumer.
	ProtoGoModule string
}

type Owner struct {
	Team  string
	Slack string
	Email string
}

type Event struct {
	Name        string
	Version     int
	Status      string
	Description string
	Owner       string
	Producer    string
	Sources     []string
	Properties  map[string]Field
	Domain      string   // first path segment
	Path        []string // full path from registry root, excluding action filename
}

type Field struct {
	Name        string
	Type        FieldType
	Required    bool
	Description string
	PII         PIIClassification
	Deprecated  bool
	Default     any
	Examples    []any
	Values      []string
	Items       *Field
	Properties  map[string]Field
}

type FieldType string

const (
	FieldTypeString    FieldType = "string"
	FieldTypeInteger   FieldType = "integer"
	FieldTypeNumber    FieldType = "number"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeTimestamp FieldType = "timestamp"
	FieldTypeDate      FieldType = "date"
	FieldTypeUUID      FieldType = "uuid"
	FieldTypeEnum      FieldType = "enum"
	FieldTypeObject    FieldType = "object"
	FieldTypeArray     FieldType = "array"
)

type PIIClassification string

const (
	PIINone         PIIClassification = "none"
	PIIPseudonymous PIIClassification = "pseudonymous"
	PIIPersonal     PIIClassification = "personal"
	PIISensitive    PIIClassification = "sensitive"
)

// Domain represents a logical grouping of related events within a registry.
type Domain struct {
	Name        string
	Description string
	Owner       string
	Context     map[string]Field
}

// Codegen holds code generation configuration for a registry.
type Codegen struct {
	Languages []string
	Configs   map[string]map[string]any
}
