package registry

import "testing"

func TestRegistryModelContracts(t *testing.T) {
	var _ string = Registry{}.Version
	var _ string = Registry{}.Namespace
	var _ PackageConfig = Registry{}.Package
	var _ []Owner = Registry{}.Owners
	var _ map[string]Field = Registry{}.Context
	var _ []Event = Registry{}.Events

	var _ string = Event{}.Name
	var _ int = Event{}.Version
	var _ map[string]Field = Event{}.Properties

	var _ string = Field{}.Name
	var _ FieldType = Field{}.Type
	var _ bool = Field{}.Required
	var _ PIIClassification = Field{}.PII
	var _ []string = Field{}.Values
	var _ *Field = Field{}.Items
	var _ map[string]Field = Field{}.Properties
}

func TestDiagnosticContracts(t *testing.T) {
	var _ error = Diagnostics{}
	var _ string = Diagnostic{}.Location
	var _ string = Diagnostic{}.Message
}

func TestNewModelContracts(t *testing.T) {
	var _ map[string]Domain = Registry{}.Domains
	var _ Codegen = Registry{}.Codegen
	var _ []string = Codegen{}.Languages
	var _ map[string]map[string]any = Codegen{}.Configs
	var _ map[string]Field = Domain{}.Context
	var _ string = Event{}.Domain
	var _ []string = Event{}.Path
}
