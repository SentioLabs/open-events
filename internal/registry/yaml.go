package registry

type registryYAML struct {
	OpenEvents string               `yaml:"openevents"`
	Namespace  string               `yaml:"namespace"`
	Package    packageYAML          `yaml:"package"`
	Defaults   defaultsYAML         `yaml:"defaults"`
	Owners     []ownerYAML          `yaml:"owners"`
	Context    map[string]fieldYAML `yaml:"context"`
	Events     map[string]eventYAML `yaml:"events"`
}

type packageYAML struct {
	Go     string `yaml:"go"`
	Python string `yaml:"python"`
}

type defaultsYAML struct {
	Queue     string        `yaml:"queue"`
	Snowflake snowflakeYAML `yaml:"snowflake"`
}

type snowflakeYAML struct {
	Database string `yaml:"database"`
	Schema   string `yaml:"schema"`
}

type ownerYAML struct {
	Team  string `yaml:"team"`
	Slack string `yaml:"slack"`
	Email string `yaml:"email"`
}

type destinationYAML struct {
	Queue          string `yaml:"queue"`
	SnowflakeTable string `yaml:"snowflake_table"`
}

type eventYAML struct {
	Version     int                  `yaml:"version"`
	Status      string               `yaml:"status"`
	Description string               `yaml:"description"`
	Owner       string               `yaml:"owner"`
	Producer    string               `yaml:"producer"`
	Sources     []string             `yaml:"sources"`
	Destination destinationYAML      `yaml:"destination"`
	Properties  map[string]fieldYAML `yaml:"properties"`
}

type fieldYAML struct {
	Type        FieldType            `yaml:"type"`
	Required    *bool                `yaml:"required"`
	Description string               `yaml:"description"`
	PII         PIIClassification    `yaml:"pii"`
	Deprecated  *bool                `yaml:"deprecated"`
	Default     any                  `yaml:"default"`
	Examples    []any                `yaml:"examples"`
	Values      []string             `yaml:"values"`
	Items       *fieldYAML           `yaml:"items"`
	Properties  map[string]fieldYAML `yaml:"properties"`
}
