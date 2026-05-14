// Package testfx provides a fluent builder for synthetic registry trees used in tests.
package testfx

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sentiolabs/open-events/internal/registry"
)

// Builder constructs a synthetic registry directory tree under t.TempDir().
type Builder struct {
	namespace string
	goPkg     string
	pyPkg     string
	owners    []ownerEntry
	languages []string
	domains   []*DomainBuilder
}

type ownerEntry struct {
	Team  string `yaml:"team"`
	Email string `yaml:"email,omitempty"`
}

// New returns a new Builder.
func New() *Builder {
	return &Builder{}
}

// Namespace sets the registry namespace.
func (b *Builder) Namespace(s string) *Builder {
	b.namespace = s
	return b
}

// Package sets the Go and Python package identifiers.
func (b *Builder) Package(goPkg, pyPkg string) *Builder {
	b.goPkg = goPkg
	b.pyPkg = pyPkg
	return b
}

// Owner adds an owner entry.
func (b *Builder) Owner(team, email string) *Builder {
	b.owners = append(b.owners, ownerEntry{Team: team, Email: email})
	return b
}

// Language adds a codegen language.
func (b *Builder) Language(name string) *Builder {
	b.languages = append(b.languages, name)
	return b
}

// Domain starts a new domain sub-builder.
func (b *Builder) Domain(name string) *DomainBuilder {
	db := &DomainBuilder{name: name, parent: b}
	b.domains = append(b.domains, db)
	return db
}

// Write serializes the registry tree to t.TempDir() and returns the root path.
func (b *Builder) Write(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Build openevents.yaml
	type packageYAML struct {
		Go     string `yaml:"go,omitempty"`
		Python string `yaml:"python,omitempty"`
	}
	type codegenYAML struct {
		Languages []string `yaml:"languages,omitempty"`
	}
	type rootYAML struct {
		Version   string       `yaml:"version"`
		Namespace string       `yaml:"namespace,omitempty"`
		Package   packageYAML  `yaml:"package,omitempty"`
		Owners    []ownerEntry `yaml:"owners,omitempty"`
		Codegen   codegenYAML  `yaml:"codegen,omitempty"`
	}
	root_ := rootYAML{
		Version:   registry.SupportedVersion,
		Namespace: b.namespace,
		Package:   packageYAML{Go: b.goPkg, Python: b.pyPkg},
		Owners:    b.owners,
		Codegen:   codegenYAML{Languages: b.languages},
	}
	writeYAML(t, filepath.Join(root, "openevents.yaml"), root_)

	// Write each domain
	for _, db := range b.domains {
		db.write(t, root)
	}

	return root
}

// DomainBuilder constructs a domain directory with domain.yml and action files.
type DomainBuilder struct {
	name        string
	description string
	owner       string
	context     []fieldEntry
	actions     []*ActionBuilder
	parent      *Builder
}

type fieldEntry struct {
	Name     string                     `yaml:"name"`
	Type     registry.FieldType         `yaml:"type"`
	Required bool                       `yaml:"required"`
	PII      registry.PIIClassification `yaml:"pii"`
}

// Description sets the domain description.
func (db *DomainBuilder) Description(s string) *DomainBuilder {
	db.description = s
	return db
}

// Owner sets the domain owner team.
func (db *DomainBuilder) Owner(team string) *DomainBuilder {
	db.owner = team
	return db
}

// Context adds a context field to the domain.
func (db *DomainBuilder) Context(name string, fieldType registry.FieldType, required bool, pii registry.PIIClassification) *DomainBuilder {
	db.context = append(db.context, fieldEntry{Name: name, Type: fieldType, Required: required, PII: pii})
	return db
}

// Action starts a new action sub-builder.
func (db *DomainBuilder) Action(categoryPath []string, action string) *ActionBuilder {
	ab := &ActionBuilder{categoryPath: categoryPath, action: action, parent: db}
	db.actions = append(db.actions, ab)
	return ab
}

// Done returns the parent Builder.
func (db *DomainBuilder) Done() *Builder {
	return db.parent
}

func (db *DomainBuilder) write(t *testing.T, root string) {
	t.Helper()
	domainDir := filepath.Join(root, db.name)
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", domainDir, err)
	}

	type contextBlock struct {
		Fields []fieldEntry `yaml:"fields,omitempty"`
	}
	type domainYAML struct {
		Description string       `yaml:"description,omitempty"`
		Owner       string       `yaml:"owner,omitempty"`
		Context     contextBlock `yaml:"context,omitempty"`
	}
	dom := domainYAML{
		Description: db.description,
		Owner:       db.owner,
		Context:     contextBlock{Fields: db.context},
	}
	writeYAML(t, filepath.Join(domainDir, "domain.yml"), dom)

	for _, ab := range db.actions {
		ab.write(t, root, db.name)
	}
}

// ActionBuilder constructs an action YAML file.
type ActionBuilder struct {
	categoryPath []string
	action       string
	version      int
	status       string
	description  string
	properties   []fieldEntry
	parent       *DomainBuilder
}

// Version sets the action version.
func (ab *ActionBuilder) Version(v int) *ActionBuilder {
	ab.version = v
	return ab
}

// Status sets the action status.
func (ab *ActionBuilder) Status(s string) *ActionBuilder {
	ab.status = s
	return ab
}

// Description sets the action description.
func (ab *ActionBuilder) Description(s string) *ActionBuilder {
	ab.description = s
	return ab
}

// Property adds a property field to the action.
func (ab *ActionBuilder) Property(name string, fieldType registry.FieldType, required bool, pii registry.PIIClassification) *ActionBuilder {
	ab.properties = append(ab.properties, fieldEntry{Name: name, Type: fieldType, Required: required, PII: pii})
	return ab
}

// Done returns the parent DomainBuilder.
func (ab *ActionBuilder) Done() *DomainBuilder {
	return ab.parent
}

func (ab *ActionBuilder) write(t *testing.T, root, domainName string) {
	t.Helper()
	parts := append([]string{root, domainName}, ab.categoryPath...)
	dir := filepath.Join(parts...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	type actionYAML struct {
		Version     int          `yaml:"version,omitempty"`
		Status      string       `yaml:"status,omitempty"`
		Description string       `yaml:"description,omitempty"`
		Properties  []fieldEntry `yaml:"properties,omitempty"`
	}
	act := actionYAML{
		Version:     ab.version,
		Status:      ab.status,
		Description: ab.description,
		Properties:  ab.properties,
	}
	writeYAML(t, filepath.Join(dir, ab.action+".yml"), act)
}

func writeYAML(t *testing.T, path string, v any) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}
