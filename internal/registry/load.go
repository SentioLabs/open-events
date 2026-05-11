package registry

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Registry, Diagnostics) {
	files, diags := discoverYAMLFiles(path)
	if diags.HasErrors() {
		return Registry{}, diags
	}

	merged := Registry{Context: map[string]Field{}}
	for _, file := range files {
		fragmentYAML, fileDiags := decodeYAMLFile(file)
		if fileDiags.HasErrors() {
			diags = append(diags, fileDiags...)
			continue
		}

		fragment, normalizeDiags := normalizeYAML(file, fragmentYAML)
		if normalizeDiags.HasErrors() {
			diags = append(diags, normalizeDiags...)
			continue
		}

		diags = append(diags, mergeRegistry(&merged, fragment, file)...)
	}

	sort.Slice(merged.Events, func(i, j int) bool {
		if merged.Events[i].Name == merged.Events[j].Name {
			return merged.Events[i].Version < merged.Events[j].Version
		}
		return merged.Events[i].Name < merged.Events[j].Name
	})

	return merged, diags
}

func discoverYAMLFiles(path string) ([]string, Diagnostics) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, Diagnostics{{Location: path, Message: err.Error()}}
	}

	if !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil, Diagnostics{{Location: path, Message: "expected .yaml or .yml file"}}
		}
		return []string{path}, nil
	}

	type discovered struct {
		rel  string
		full string
	}

	var found []discovered
	walkErr := filepath.WalkDir(path, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		rel, relErr := filepath.Rel(path, current)
		if relErr != nil {
			return relErr
		}

		found = append(found, discovered{rel: rel, full: current})
		return nil
	})
	if walkErr != nil {
		return nil, Diagnostics{{Location: path, Message: walkErr.Error()}}
	}

	if len(found) == 0 {
		return nil, Diagnostics{{Location: path, Message: "no .yaml or .yml files found"}}
	}

	sort.Slice(found, func(i, j int) bool {
		return found[i].rel < found[j].rel
	})

	files := make([]string, 0, len(found))
	for _, file := range found {
		files = append(files, file.full)
	}

	return files, nil
}

func decodeYAMLFile(path string) (registryYAML, Diagnostics) {
	data, err := os.ReadFile(path)
	if err != nil {
		return registryYAML{}, Diagnostics{{Location: path, Message: err.Error()}}
	}

	var out registryYAML
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&out); err != nil {
		return registryYAML{}, Diagnostics{{Location: path, Message: err.Error()}}
	}

	var trailing registryYAML
	if err := decoder.Decode(&trailing); err != io.EOF {
		return registryYAML{}, Diagnostics{{Location: path, Message: "additional YAML documents are not supported"}}
	}

	return out, nil
}

func normalizeYAML(path string, in registryYAML) (Registry, Diagnostics) {
	out := Registry{
		Version:   in.OpenEvents,
		Namespace: in.Namespace,
		Package: PackageConfig{
			Go:     in.Package.Go,
			Python: in.Package.Python,
		},
		Defaults: Defaults{
			Queue: in.Defaults.Queue,
			Snowflake: SnowflakeDefaults{
				Database: in.Defaults.Snowflake.Database,
				Schema:   in.Defaults.Snowflake.Schema,
			},
		},
		Owners:  make([]Owner, 0, len(in.Owners)),
		Context: map[string]Field{},
		Events:  make([]Event, 0, len(in.Events)),
	}

	for _, owner := range in.Owners {
		out.Owners = append(out.Owners, Owner{
			Team:  owner.Team,
			Slack: owner.Slack,
			Email: owner.Email,
		})
	}

	contextKeys := make([]string, 0, len(in.Context))
	for name := range in.Context {
		contextKeys = append(contextKeys, name)
	}
	sort.Strings(contextKeys)

	for _, name := range contextKeys {
		out.Context[name] = normalizeField(name, in.Context[name])
	}

	eventNames := make([]string, 0, len(in.Events))
	for name := range in.Events {
		eventNames = append(eventNames, name)
	}
	sort.Strings(eventNames)

	for _, name := range eventNames {
		event := in.Events[name]

		properties := map[string]Field{}
		propertyKeys := make([]string, 0, len(event.Properties))
		for key := range event.Properties {
			propertyKeys = append(propertyKeys, key)
		}
		sort.Strings(propertyKeys)
		for _, key := range propertyKeys {
			properties[key] = normalizeField(key, event.Properties[key])
		}

		out.Events = append(out.Events, Event{
			Name:        name,
			Version:     event.Version,
			Status:      event.Status,
			Description: event.Description,
			Owner:       event.Owner,
			Producer:    event.Producer,
			Sources:     append([]string(nil), event.Sources...),
			Destination: Destination{
				Queue:          event.Destination.Queue,
				SnowflakeTable: event.Destination.SnowflakeTable,
			},
			Properties: properties,
		})
	}

	return out, nil
}

func normalizeField(name string, in fieldYAML) Field {
	required := false
	if in.Required != nil {
		required = *in.Required
	}

	deprecated := false
	if in.Deprecated != nil {
		deprecated = *in.Deprecated
	}

	out := Field{
		Name:        name,
		Type:        in.Type,
		Required:    required,
		Description: in.Description,
		PII:         in.PII,
		Deprecated:  deprecated,
		Default:     in.Default,
		Examples:    append([]any(nil), in.Examples...),
		Values:      append([]string(nil), in.Values...),
		Properties:  map[string]Field{},
	}

	if in.Items != nil {
		item := normalizeField(name, *in.Items)
		out.Items = &item
	}

	keys := make([]string, 0, len(in.Properties))
	for key := range in.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out.Properties[key] = normalizeField(key, in.Properties[key])
	}

	return out
}

func mergeRegistry(dst *Registry, src Registry, sourcePath string) Diagnostics {
	if dst.Context == nil {
		dst.Context = map[string]Field{}
	}

	var diags Diagnostics
	mergeSingletonString := func(field, next *string, fieldName string) {
		if *next == "" {
			return
		}
		if *field == "" {
			*field = *next
			return
		}
		if *field != *next {
			diags = append(diags, Diagnostic{
				Location: fmt.Sprintf("%s: %s", sourcePath, fieldName),
				Message:  fmt.Sprintf("conflicting value %q; already set to %q", *next, *field),
			})
		}
	}

	mergeSingletonString(&dst.Version, &src.Version, "openevents")
	mergeSingletonString(&dst.Namespace, &src.Namespace, "namespace")
	mergeSingletonString(&dst.Package.Go, &src.Package.Go, "package.go")
	mergeSingletonString(&dst.Package.Python, &src.Package.Python, "package.python")
	mergeSingletonString(&dst.Defaults.Queue, &src.Defaults.Queue, "defaults.queue")
	mergeSingletonString(&dst.Defaults.Snowflake.Database, &src.Defaults.Snowflake.Database, "defaults.snowflake.database")
	mergeSingletonString(&dst.Defaults.Snowflake.Schema, &src.Defaults.Snowflake.Schema, "defaults.snowflake.schema")

	dst.Owners = append(dst.Owners, src.Owners...)

	contextKeys := make([]string, 0, len(src.Context))
	for name := range src.Context {
		contextKeys = append(contextKeys, name)
	}
	sort.Strings(contextKeys)
	for _, name := range contextKeys {
		if _, exists := dst.Context[name]; exists {
			diags = append(diags, Diagnostic{
				Location: sourcePath,
				Message:  fmt.Sprintf("context.%s: duplicate context field", name),
			})
			continue
		}
		dst.Context[name] = src.Context[name]
	}

	seen := make(map[string]struct{}, len(dst.Events))
	for _, event := range dst.Events {
		key := fmt.Sprintf("%s#%d", event.Name, event.Version)
		seen[key] = struct{}{}
	}

	for _, event := range src.Events {
		key := fmt.Sprintf("%s#%d", event.Name, event.Version)
		if _, exists := seen[key]; exists {
			diags = append(diags, Diagnostic{
				Location: sourcePath,
				Message:  fmt.Sprintf("events.%s: duplicate event version %d", event.Name, event.Version),
			})
			continue
		}
		dst.Events = append(dst.Events, event)
		seen[key] = struct{}{}
	}

	return diags
}
