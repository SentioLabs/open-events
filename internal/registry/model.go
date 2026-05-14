package registry

const SupportedVersion = "0.1.0"

type Registry struct {
	Version   string
	Namespace string
	Package   PackageConfig
	Defaults  Defaults
	Owners    []Owner
	Context   map[string]Field
	Events    []Event
	Domains   map[string]Domain // NEW
	Codegen   Codegen           // NEW
}

type PackageConfig struct {
	Go     string
	Python string
}

type Defaults struct {
	Queue     string
	Snowflake SnowflakeDefaults
}

type SnowflakeDefaults struct {
	Database string
	Schema   string
}

type Owner struct {
	Team  string
	Slack string
	Email string
}

type Destination struct {
	Queue          string
	SnowflakeTable string
}

type Event struct {
	Name        string
	Version     int
	Status      string
	Description string
	Owner       string
	Producer    string
	Sources     []string
	Destination Destination
	Properties  map[string]Field
	Domain      string   // NEW: first path segment
	Path        []string // NEW: full path from registry root, excluding action filename
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
