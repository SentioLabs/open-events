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
