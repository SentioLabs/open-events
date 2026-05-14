package protogen

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sentiolabs/open-events/internal/schemair"
	"gopkg.in/yaml.v3"
)

const (
	metadataVersion = 1
	metadataBackend = "protobuf"
)

var goPackagePattern = regexp.MustCompile(`^[a-z0-9]+([._/-][a-z0-9]+)*$`)

// Render writes protobuf backend files for reg into outDir.
//
// When reg.DomainSpecs is populated (T4+ per-domain shape), Render emits:
//   - <outDir>/<ns-path>/common/v1/common.proto   — shared Client message
//   - <outDir>/<ns-path>/<domain>/v1/events.proto — per-domain context + events
//
// Legacy Files (for backward compatibility) are also emitted when present.
func Render(reg schemair.Registry, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", outDir, err)
	}

	if err := os.WriteFile(filepath.Join(outDir, "buf.yaml"), RenderBufYAML(), 0o644); err != nil {
		return fmt.Errorf("write buf.yaml: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "buf.gen.yaml"), RenderBufGenYAML(), 0o644); err != nil {
		return fmt.Errorf("write buf.gen.yaml: %w", err)
	}

	protoRoot := filepath.Join(outDir, "proto")

	// Emit per-domain proto files when DomainSpecs is populated.
	if len(reg.DomainSpecs) > 0 {
		nsPath := namespaceToPath(reg.Namespace)

		// Emit common.proto.
		commonProtoPath := filepath.Join(protoRoot, filepath.FromSlash(nsPath), "common", "v1", "common.proto")
		commonBytes, err := RenderCommonProto(reg)
		if err != nil {
			return fmt.Errorf("render common.proto: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(commonProtoPath), 0o755); err != nil {
			return fmt.Errorf("create directory for common.proto: %w", err)
		}
		if err := os.WriteFile(commonProtoPath, commonBytes, 0o644); err != nil {
			return fmt.Errorf("write common.proto: %w", err)
		}

		// Emit per-domain events.proto.
		for _, ds := range reg.DomainSpecs {
			domainProtoPath := filepath.Join(protoRoot, filepath.FromSlash(nsPath), ds.Name, "v1", "events.proto")
			domainBytes, err := RenderDomainProto(reg.Namespace, ds)
			if err != nil {
				return fmt.Errorf("render %s/v1/events.proto: %w", ds.Name, err)
			}
			if err := os.MkdirAll(filepath.Dir(domainProtoPath), 0o755); err != nil {
				return fmt.Errorf("create directory for %s/v1/events.proto: %w", ds.Name, err)
			}
			if err := os.WriteFile(domainProtoPath, domainBytes, 0o644); err != nil {
				return fmt.Errorf("write %s/v1/events.proto: %w", ds.Name, err)
			}
		}
	}

	// Emit legacy single-file output (backward compatibility).
	for _, file := range reg.Files {
		protoPath, err := resolveProtoOutputPath(protoRoot, file.Path)
		if err != nil {
			return err
		}

		protoBytes, err := RenderFile(file)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(protoPath), 0o755); err != nil {
			return fmt.Errorf("create proto directory for %q: %w", file.Path, err)
		}
		if err := os.WriteFile(protoPath, protoBytes, 0o644); err != nil {
			return fmt.Errorf("write proto file %q: %w", file.Path, err)
		}
	}

	metadataBytes, err := RenderMetadata(reg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "openevents.metadata.yaml"), metadataBytes, 0o644); err != nil {
		return fmt.Errorf("write openevents.metadata.yaml: %w", err)
	}

	return nil
}

// namespaceToPath converts a namespace like "com.acme.platform" to "com/acme/platform".
func namespaceToPath(namespace string) string {
	return strings.ReplaceAll(namespace, ".", "/")
}

// RenderCommonProto renders the common.proto file containing shared types (e.g. Client).
func RenderCommonProto(reg schemair.Registry) ([]byte, error) {
	nsPath := namespaceToPath(reg.Namespace)
	pkg := reg.Namespace + ".common.v1"

	var b strings.Builder
	b.WriteString("syntax = \"proto3\";\n\n")
	fmt.Fprintf(&b, "package %s;\n\n", pkg)

	if err := renderMessage(&b, reg.CommonSpec.Client); err != nil {
		return nil, err
	}

	_ = nsPath // nsPath used for import path in callers; kept here for documentation.
	return []byte(b.String()), nil
}

// RenderDomainProto renders the events.proto for a single domain.
func RenderDomainProto(namespace string, ds schemair.DomainSpec) ([]byte, error) {
	nsPath := namespaceToPath(namespace)
	pkg := namespace + "." + ds.Name + ".v1"
	commonImport := nsPath + "/common/v1/common.proto"

	var b strings.Builder
	b.WriteString("syntax = \"proto3\";\n\n")
	fmt.Fprintf(&b, "package %s;\n\n", pkg)
	fmt.Fprintf(&b, "import %s;\n\n", strconv.Quote(commonImport))

	// Render context message.
	contextMsg := schemair.Message{
		Name:   ds.ContextName,
		Fields: ds.ContextFields,
		Enums:  ds.ContextEnums,
	}
	if err := renderMessage(&b, contextMsg); err != nil {
		return nil, err
	}

	// Render event messages (envelope + properties pairs).
	for _, de := range ds.Events {
		b.WriteString("\n")
		if err := renderMessage(&b, de.Envelope); err != nil {
			return nil, err
		}
		b.WriteString("\n")
		if err := renderMessage(&b, de.Properties); err != nil {
			return nil, err
		}
	}

	return []byte(b.String()), nil
}

// RenderFile renders one schema IR file as a proto3 file.
func RenderFile(file schemair.File) ([]byte, error) {
	hasTimestamp, err := fileUsesTimestamp(file)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString("syntax = \"proto3\";\n\n")
	fmt.Fprintf(&b, "package %s;\n", file.Package)
	if file.GoPackage != "" {
		if err := validateGoPackage(file.GoPackage); err != nil {
			return nil, err
		}
		alias, err := goPackageAlias(file.GoPackage)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&b, "option go_package = %s;\n", strconv.Quote(file.GoPackage+";"+alias))
	}
	if hasTimestamp {
		b.WriteString("\nimport \"google/protobuf/timestamp.proto\";\n")
	}
	b.WriteString("\n")

	for i, message := range file.Messages {
		if i > 0 {
			b.WriteString("\n")
		}
		if err := renderMessage(&b, message); err != nil {
			return nil, err
		}
	}

	return []byte(b.String()), nil
}

// RenderBufYAML renders the Buf module configuration.
func RenderBufYAML() []byte {
	return []byte("version: v2\nmodules:\n  - path: proto\n")
}

// RenderBufGenYAML renders the Buf generation configuration.
func RenderBufGenYAML() []byte {
	return []byte("version: v2\nplugins:\n  - local: protoc-gen-go\n    out: gen/go\n    opt: paths=source_relative\n  - protoc_builtin: python\n    out: gen/python\n")
}

// RenderMetadata renders deterministic protobuf sidecar metadata for reg.
func RenderMetadata(reg schemair.Registry) ([]byte, error) {
	root := metadataRoot{
		Version:   metadataVersion,
		Backend:   metadataBackend,
		Namespace: reg.Namespace,
		Files:     make([]metadataFile, 0, len(reg.Files)),
	}

	for _, file := range reg.Files {
		metadataFile := metadataFile{
			Path:     path.Join("proto", file.Path),
			Package:  file.Package,
			Messages: make([]metadataMessage, 0, len(file.Messages)),
		}
		for _, message := range file.Messages {
			metadataMessage := metadataMessage{
				Name:        message.Name,
				Description: message.Description,
				Fields:      make([]metadataField, 0, len(message.Fields)),
				Enums:       make([]metadataEnum, 0, len(message.Enums)),
			}
			for _, field := range message.Fields {
				kind, fieldType, err := typeRefKindAndType(field, message.Name+"."+field.Name)
				if err != nil {
					return nil, err
				}
				metadataMessage.Fields = append(metadataMessage.Fields, metadataField{
					Name:        field.Name,
					Number:      field.Number,
					Kind:        kind,
					Type:        fieldType,
					Repeated:    field.Repeated,
					Optional:    field.Optional,
					Required:    field.Required,
					Description: field.Description,
				})
			}
			for _, enum := range message.Enums {
				metadataEnum := metadataEnum{
					Name:   enum.Name,
					Values: make([]metadataEnumValue, 0, len(enum.Values)),
				}
				for _, value := range enum.Values {
					metadataEnum.Values = append(metadataEnum.Values, metadataEnumValue{
						Name:     value.Name,
						Original: value.Original,
						Number:   value.Number,
					})
				}
				metadataMessage.Enums = append(metadataMessage.Enums, metadataEnum)
			}
			metadataFile.Messages = append(metadataFile.Messages, metadataMessage)
		}
		root.Files = append(root.Files, metadataFile)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(root); err != nil {
		return nil, fmt.Errorf("render metadata: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("render metadata: %w", err)
	}

	return buf.Bytes(), nil
}

func renderMessage(b *strings.Builder, message schemair.Message) error {
	renderProtoComments(b, "", message.Description)
	fmt.Fprintf(b, "message %s {\n", message.Name)
	for _, field := range message.Fields {
		fieldPath := message.Name + "." + field.Name
		kind, _, err := typeRefKindAndType(field, fieldPath)
		if err != nil {
			return err
		}
		fieldType, err := protoFieldType(field, fieldPath)
		if err != nil {
			return err
		}

		renderProtoComments(b, "  ", field.Description)
		label := ""
		switch {
		case field.Repeated:
			label = "repeated "
		case field.Optional && (kind == "scalar" || kind == "enum"):
			label = "optional "
		}
		fmt.Fprintf(b, "  %s%s %s = %d;\n", label, fieldType, field.Name, field.Number)
	}

	if len(message.Fields) > 0 && len(message.Enums) > 0 {
		b.WriteString("\n")
	}
	for i, enum := range message.Enums {
		if i > 0 {
			b.WriteString("\n")
		}
		renderEnum(b, enum)
	}
	b.WriteString("}\n")

	return nil
}

func renderEnum(b *strings.Builder, enum schemair.Enum) {
	fmt.Fprintf(b, "  enum %s {\n", enum.Name)
	fmt.Fprintf(b, "    %s = 0;\n", schemair.EnumZeroValueName(enum.Name))
	for _, value := range enum.Values {
		fmt.Fprintf(b, "    %s = %d;\n", value.Name, value.Number)
	}
	b.WriteString("  }\n")
}

func renderProtoComments(b *strings.Builder, indent string, description string) {
	if description == "" {
		return
	}

	normalized := strings.ReplaceAll(description, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	for _, line := range strings.Split(normalized, "\n") {
		if line == "" {
			fmt.Fprintf(b, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(b, "%s// %s\n", indent, line)
	}
}

func resolveProtoOutputPath(protoRoot string, filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("invalid proto file path %q: must not be empty", filePath)
	}
	if strings.Contains(filePath, "\\") {
		return "", fmt.Errorf("invalid proto file path %q: must use slash-separated relative paths", filePath)
	}
	if isDriveQualifiedPath(filePath) {
		return "", fmt.Errorf("invalid proto file path %q: must not be drive-qualified or drive-relative", filePath)
	}
	if path.IsAbs(filePath) {
		return "", fmt.Errorf("invalid proto file path %q: must be relative", filePath)
	}

	for _, segment := range strings.Split(filePath, "/") {
		if segment == ".." {
			return "", fmt.Errorf("invalid proto file path %q: must not contain '..' segments", filePath)
		}
	}

	cleanFilePath := path.Clean(filePath)
	if cleanFilePath == "." {
		return "", fmt.Errorf("invalid proto file path %q: must not be empty", filePath)
	}
	for _, segment := range strings.Split(cleanFilePath, "/") {
		if segment == ".." {
			return "", fmt.Errorf("invalid proto file path %q: must not contain '..' segments", filePath)
		}
	}

	destination := filepath.Join(protoRoot, filepath.FromSlash(cleanFilePath))
	rel, err := filepath.Rel(protoRoot, destination)
	if err != nil {
		return "", fmt.Errorf("invalid proto file path %q: resolve destination: %w", filePath, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid proto file path %q: destination escapes proto output root", filePath)
	}

	return destination, nil
}

func fileUsesTimestamp(file schemair.File) (bool, error) {
	hasTimestamp := false
	for _, message := range file.Messages {
		for _, field := range message.Fields {
			kind, fieldType, err := typeRefKindAndType(field, message.Name+"."+field.Name)
			if err != nil {
				return false, err
			}
			if kind == "scalar" && fieldType == "timestamp" {
				hasTimestamp = true
			}
		}
	}
	return hasTimestamp, nil
}

func protoFieldType(field schemair.Field, fieldPath string) (string, error) {
	kind, fieldType, err := typeRefKindAndType(field, fieldPath)
	if err != nil {
		return "", err
	}

	switch kind {
	case "scalar":
		switch fieldType {
		case "string", "uuid", "date":
			return "string", nil
		case "integer":
			return "int64", nil
		case "number":
			return "double", nil
		case "boolean":
			return "bool", nil
		case "timestamp":
			return "google.protobuf.Timestamp", nil
		default:
			return "", fmt.Errorf("field %s has unsupported scalar type %q", fieldPath, fieldType)
		}
	case "message", "enum":
		return fieldType, nil
	default:
		return "", fmt.Errorf("field %s has unsupported TypeRef kind %q", fieldPath, kind)
	}
}

func typeRefKindAndType(field schemair.Field, fieldPath string) (string, string, error) {
	count := 0
	kind := ""
	fieldType := ""
	if field.Type.Scalar != "" {
		count++
		kind = "scalar"
		fieldType = field.Type.Scalar
	}
	if field.Type.Message != "" {
		count++
		kind = "message"
		fieldType = field.Type.Message
	}
	if field.Type.Enum != "" {
		count++
		kind = "enum"
		fieldType = field.Type.Enum
	}
	if count != 1 {
		return "", "", fmt.Errorf("field %s must have exactly one TypeRef member set (got %d)", fieldPath, count)
	}
	return kind, fieldType, nil
}

func isDriveQualifiedPath(filePath string) bool {
	if len(filePath) < 2 || filePath[1] != ':' {
		return false
	}

	first := filePath[0]
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')
}

func validateGoPackage(goPackage string) error {
	if !goPackagePattern.MatchString(goPackage) {
		return fmt.Errorf("invalid package.go %q: must be a valid Go import path", goPackage)
	}
	if !strings.Contains(goPackage, ".") && !strings.Contains(goPackage, "/") {
		return fmt.Errorf("invalid package.go %q: must include at least one '.' or '/' in the import path", goPackage)
	}
	if strings.Contains(goPackage, ";") {
		return fmt.Errorf("invalid package.go %q: must not contain semicolons", goPackage)
	}
	for _, r := range goPackage {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("invalid package.go %q: must not contain control characters", goPackage)
		}
	}
	return nil
}

func goPackageAlias(goPackage string) (string, error) {
	lastSlash := strings.LastIndex(goPackage, "/")
	alias := goPackage
	if lastSlash >= 0 {
		alias = goPackage[lastSlash+1:]
	}
	var cleaned strings.Builder
	for _, r := range alias {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			cleaned.WriteRune(r)
			continue
		}
		cleaned.WriteRune('_')
	}
	alias = cleaned.String()
	if alias == "" {
		return "", fmt.Errorf("invalid package.go %q: alias is empty", goPackage)
	}
	if alias[0] >= '0' && alias[0] <= '9' {
		alias = "pkg_" + alias
	}
	if token.Lookup(alias).IsKeyword() {
		return "", fmt.Errorf("invalid package.go %q: alias %q is a Go keyword", goPackage, alias)
	}
	return alias, nil
}

type metadataRoot struct {
	Version   int            `yaml:"version"`
	Backend   string         `yaml:"backend"`
	Namespace string         `yaml:"namespace"`
	Files     []metadataFile `yaml:"files"`
}

type metadataFile struct {
	Path     string            `yaml:"path"`
	Package  string            `yaml:"package"`
	Messages []metadataMessage `yaml:"messages"`
}

type metadataMessage struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description,omitempty"`
	Fields      []metadataField `yaml:"fields"`
	Enums       []metadataEnum  `yaml:"enums"`
}

type metadataField struct {
	Name        string `yaml:"name"`
	Number      int    `yaml:"number"`
	Kind        string `yaml:"kind"`
	Type        string `yaml:"type"`
	Repeated    bool   `yaml:"repeated"`
	Optional    bool   `yaml:"optional"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description,omitempty"`
}

type metadataEnum struct {
	Name   string              `yaml:"name"`
	Values []metadataEnumValue `yaml:"values"`
}

type metadataEnumValue struct {
	Name     string `yaml:"name"`
	Original string `yaml:"original"`
	Number   int    `yaml:"number"`
}
