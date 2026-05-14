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
	Go     string
	Python string
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
