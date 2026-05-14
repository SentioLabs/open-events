package registry

// rootYAML maps the openevents.yaml file at the registry root.
type rootYAML struct {
	Openevents string      `yaml:"openevents"`
	Namespace  string      `yaml:"namespace"`
	Package    packageYAML `yaml:"package"`
	Owners     []ownerYAML `yaml:"owners"`
	Codegen    codegenYAML `yaml:"codegen"`
}

type packageYAML struct {
	Go     string `yaml:"go"`
	Python string `yaml:"python"`
}

type ownerYAML struct {
	Team  string `yaml:"team"`
	Slack string `yaml:"slack"`
	Email string `yaml:"email"`
}

type codegenYAML struct {
	Languages []string               `yaml:"languages"`
	Configs   map[string]interface{} `yaml:"configs"`
}

// domainYAML maps a <domain>/domain.yml file.
type domainYAML struct {
	Description string            `yaml:"description"`
	Owner       string            `yaml:"owner"`
	Context     domainContextYAML `yaml:"context"`
}

type domainContextYAML struct {
	Fields []fieldEntryYAML `yaml:"fields"`
}

// actionYAML maps a per-action <action>.yml file.
type actionYAML struct {
	Version     int              `yaml:"version"`
	Status      string           `yaml:"status"`
	Description string           `yaml:"description"`
	Owner       string           `yaml:"owner"`
	Producer    string           `yaml:"producer"`
	Sources     []string         `yaml:"sources"`
	Properties  []fieldEntryYAML `yaml:"properties"`
}

// fieldEntryYAML is a single field entry in a list-form context or properties block.
type fieldEntryYAML struct {
	Name     string            `yaml:"name"`
	Type     FieldType         `yaml:"type"`
	Required bool              `yaml:"required"`
	PII      PIIClassification `yaml:"pii"`
}
