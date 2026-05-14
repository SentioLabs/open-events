package registry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads a registry from the directory at path. The directory must contain
// an openevents.yaml file at its root, plus one subdirectory per domain, each
// containing domain.yml and optional action YAML files in nested subdirectories.
// Passing a non-directory path returns a diagnostic error.
func Load(path string) (Registry, Diagnostics) {
	info, err := os.Stat(path)
	if err != nil {
		return Registry{}, Diagnostics{{Location: path, Message: err.Error()}}
	}
	if !info.IsDir() {
		return Registry{}, Diagnostics{{Location: path, Message: "expected directory containing openevents.yaml"}}
	}

	// 1. Parse openevents.yaml
	rootPath := filepath.Join(path, "openevents.yaml")
	root, diags := decodeRootYAML(rootPath)
	if diags.HasErrors() {
		return Registry{}, diags
	}

	reg := Registry{
		Version:   root.Openevents,
		Namespace: root.Namespace,
		Package: PackageConfig{
			Go:     root.Package.Go,
			Python: root.Package.Python,
		},
		Owners:  normalizeOwners(root.Owners),
		Context: map[string]Field{},
		Domains: map[string]Domain{},
		Codegen: Codegen{
			Languages: root.Codegen.Languages,
		},
	}

	// 2. Walk top-level subdirectories as domains
	entries, err := os.ReadDir(path)
	if err != nil {
		return Registry{}, Diagnostics{{Location: path, Message: err.Error()}}
	}

	// Collect domain names in sorted order for determinism
	var domainNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			domainNames = append(domainNames, entry.Name())
		}
	}
	sort.Strings(domainNames)

	var allDiags Diagnostics
	for _, domainName := range domainNames {
		domainDir := filepath.Join(path, domainName)
		domain, events, domDiags := loadDomain(domainDir, domainName)
		allDiags = append(allDiags, domDiags...)
		if !domDiags.HasErrors() {
			reg.Domains[domainName] = domain
			reg.Events = append(reg.Events, events...)
		}
	}

	sort.Slice(reg.Events, func(i, j int) bool {
		if reg.Events[i].Name == reg.Events[j].Name {
			return reg.Events[i].Version < reg.Events[j].Version
		}
		return reg.Events[i].Name < reg.Events[j].Name
	})

	return reg, allDiags
}

// loadDomain reads domain.yml and all action files under domainDir.
func loadDomain(domainDir, domainName string) (Domain, []Event, Diagnostics) {
	var diags Diagnostics

	domainYMLPath := filepath.Join(domainDir, "domain.yml")
	dom, domDiags := decodeDomainYAML(domainYMLPath)
	if domDiags.HasErrors() {
		return Domain{}, nil, domDiags
	}

	domain := Domain{
		Name:        domainName,
		Description: dom.Description,
		Owner:       dom.Owner,
		Context:     normalizeFieldList(dom.Context.Fields),
	}

	// Walk the domain subtree for action files
	var events []Event
	walkErr := filepath.WalkDir(domainDir, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		// Skip domain.yml itself
		if name == "domain.yml" {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}

		// Compute relative path segments from domainDir
		rel, relErr := filepath.Rel(domainDir, current)
		if relErr != nil {
			diags = append(diags, Diagnostic{Location: current, Message: relErr.Error()})
			return nil
		}

		// rel = "auth/signup.yml" → segments = ["auth", "signup.yml"]
		segments := strings.Split(rel, string(filepath.Separator))
		// Last segment is the action filename without extension
		actionFile := segments[len(segments)-1]
		actionName := strings.TrimSuffix(actionFile, filepath.Ext(actionFile))
		// Category segments are everything between domain and action
		categories := segments[:len(segments)-1]

		// Composed event name: domain.categories....action
		parts := []string{domainName}
		parts = append(parts, categories...)
		parts = append(parts, actionName)
		eventName := strings.Join(parts, ".")

		// Path = domain + categories (excludes action)
		pathSegments := []string{domainName}
		pathSegments = append(pathSegments, categories...)

		act, actDiags := decodeActionYAML(current)
		if actDiags.HasErrors() {
			diags = append(diags, actDiags...)
			return nil
		}

		events = append(events, Event{
			Name:        eventName,
			Version:     act.Version,
			Status:      act.Status,
			Description: act.Description,
			Owner:       act.Owner,
			Producer:    act.Producer,
			Sources:     act.Sources,
			Properties:  normalizeFieldList(act.Properties),
			Domain:      domainName,
			Path:        pathSegments,
		})
		return nil
	})
	if walkErr != nil {
		diags = append(diags, Diagnostic{Location: domainDir, Message: walkErr.Error()})
	}

	return domain, events, diags
}

// decodeRootYAML parses a rootYAML from the file at path.
func decodeRootYAML(path string) (rootYAML, Diagnostics) {
	data, err := os.ReadFile(path)
	if err != nil {
		return rootYAML{}, Diagnostics{{Location: path, Message: fmt.Sprintf("openevents.yaml: %s", err.Error())}}
	}
	var out rootYAML
	if decErr := decodeStrictYAML(data, &out); decErr != nil {
		return rootYAML{}, Diagnostics{{Location: path, Message: decErr.Error()}}
	}
	return out, nil
}

// decodeDomainYAML parses a domainYAML from the file at path.
func decodeDomainYAML(path string) (domainYAML, Diagnostics) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domainYAML{}, Diagnostics{{Location: path, Message: fmt.Sprintf("domain.yml: %s", err.Error())}}
	}
	var out domainYAML
	if decErr := decodeStrictYAML(data, &out); decErr != nil {
		return domainYAML{}, Diagnostics{{Location: path, Message: decErr.Error()}}
	}
	return out, nil
}

// decodeActionYAML parses an actionYAML from the file at path.
func decodeActionYAML(path string) (actionYAML, Diagnostics) {
	data, err := os.ReadFile(path)
	if err != nil {
		return actionYAML{}, Diagnostics{{Location: path, Message: err.Error()}}
	}
	var out actionYAML
	if decErr := decodeStrictYAML(data, &out); decErr != nil {
		return actionYAML{}, Diagnostics{{Location: path, Message: decErr.Error()}}
	}
	return out, nil
}

// decodeStrictYAML decodes a YAML document into v using strict mode (unknown fields error).
func decodeStrictYAML(data []byte, v interface{}) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	return decoder.Decode(v)
}

// normalizeOwners converts ownerYAML slice to Owner slice.
func normalizeOwners(in []ownerYAML) []Owner {
	out := make([]Owner, 0, len(in))
	for _, o := range in {
		out = append(out, Owner{
			Team:  o.Team,
			Slack: o.Slack,
			Email: o.Email,
		})
	}
	return out
}

// normalizeFieldList converts a list of fieldEntryYAML to a map[string]Field.
func normalizeFieldList(in []fieldEntryYAML) map[string]Field {
	out := make(map[string]Field, len(in))
	for _, f := range in {
		out[f.Name] = normalizeField(f)
	}
	return out
}

// normalizeField converts a single fieldEntryYAML to a Field, recursively
// populating Values, Items, Properties, and Description.
func normalizeField(f fieldEntryYAML) Field {
	field := Field{
		Name:        f.Name,
		Type:        f.Type,
		Required:    f.Required,
		PII:         f.PII,
		Description: f.Description,
		Values:      f.Values,
	}
	if f.Items != nil {
		item := normalizeField(*f.Items)
		// The validator expects array item fields to have Name == "items".
		item.Name = "items"
		field.Items = &item
	}
	if len(f.Properties) > 0 {
		field.Properties = make(map[string]Field, len(f.Properties))
		for propName, propEntry := range f.Properties {
			// Properties are keyed by name in the map, so inject the name.
			propEntry.Name = propName
			field.Properties[propName] = normalizeField(propEntry)
		}
	}
	return field
}
