package protogen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/schemair"
)

// TestRenderPerDomainWritesExpectedFiles verifies that Render emits per-domain
// proto files and common.proto under the correct paths when reg.DomainSpecs is
// populated.
func TestRenderPerDomainWritesExpectedFiles(t *testing.T) {
	reg := domainRegistry()
	outDir := t.TempDir()

	if err := Render(reg, outDir); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	// common.proto must exist at <ns-path>/common/v1/common.proto.
	commonPath := filepath.Join(outDir, "proto", "com", "acme", "platform", "common", "v1", "common.proto")
	commonBytes, err := os.ReadFile(commonPath)
	if err != nil {
		t.Fatalf("read common.proto: %v", err)
	}
	commonText := string(commonBytes)

	if !strings.Contains(commonText, "syntax = \"proto3\";") {
		t.Fatalf("common.proto missing syntax declaration:\n%s", commonText)
	}
	if !strings.Contains(commonText, "package com.acme.platform.common.v1;") {
		t.Fatalf("common.proto missing package declaration:\n%s", commonText)
	}
	if !strings.Contains(commonText, "message Client {") {
		t.Fatalf("common.proto missing Client message:\n%s", commonText)
	}

	// user/v1/events.proto must exist at <ns-path>/user/v1/events.proto.
	userPath := filepath.Join(outDir, "proto", "com", "acme", "platform", "user", "v1", "events.proto")
	userBytes, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("read user/v1/events.proto: %v", err)
	}
	userText := string(userBytes)

	if !strings.Contains(userText, "syntax = \"proto3\";") {
		t.Fatalf("user events.proto missing syntax:\n%s", userText)
	}
	if !strings.Contains(userText, "package com.acme.platform.user.v1;") {
		t.Fatalf("user events.proto missing package:\n%s", userText)
	}
	if !strings.Contains(userText, `import "com/acme/platform/common/v1/common.proto";`) {
		t.Fatalf("user events.proto missing common.proto import:\n%s", userText)
	}
	if !strings.Contains(userText, "message UserContext {") {
		t.Fatalf("user events.proto missing UserContext message:\n%s", userText)
	}
	// Event message must reference the domain-local UserContext type.
	if !strings.Contains(userText, "UserContext context =") {
		t.Fatalf("user events.proto event context field must reference UserContext:\n%s", userText)
	}
	if !strings.Contains(userText, "message SignedUpV1 {") {
		t.Fatalf("user events.proto missing SignedUpV1 message:\n%s", userText)
	}
	if !strings.Contains(userText, "message SignedUpV1Properties {") {
		t.Fatalf("user events.proto missing SignedUpV1Properties message:\n%s", userText)
	}
}

// TestRenderCommonProtoContentsAreCorrect checks the exact structure of common.proto.
func TestRenderCommonProtoContentsAreCorrect(t *testing.T) {
	reg := domainRegistry()

	got, err := RenderCommonProto(reg)
	if err != nil {
		t.Fatalf("RenderCommonProto() error = %v, want nil", err)
	}
	text := string(got)

	if !strings.Contains(text, "package com.acme.platform.common.v1;") {
		t.Fatalf("common.proto missing package: %s", text)
	}
	if !strings.Contains(text, "message Client {") {
		t.Fatalf("common.proto missing Client message: %s", text)
	}
	if !strings.Contains(text, "optional string name = 1;") {
		t.Fatalf("common.proto Client missing name field: %s", text)
	}
	if !strings.Contains(text, "optional string version = 2;") {
		t.Fatalf("common.proto Client missing version field: %s", text)
	}
}

// TestRenderDomainProtoContentsAreCorrect checks per-domain proto contents.
func TestRenderDomainProtoContentsAreCorrect(t *testing.T) {
	reg := domainRegistry()
	if len(reg.DomainSpecs) == 0 {
		t.Fatal("domainRegistry() returned no DomainSpecs")
	}
	ds := reg.DomainSpecs[0] // "user" domain

	got, err := RenderDomainProto(reg.Namespace, "", ds)
	if err != nil {
		t.Fatalf("RenderDomainProto() error = %v, want nil", err)
	}
	text := string(got)

	if !strings.Contains(text, "syntax = \"proto3\";") {
		t.Fatalf("domain proto missing syntax: %s", text)
	}
	if !strings.Contains(text, "package com.acme.platform.user.v1;") {
		t.Fatalf("domain proto missing package: %s", text)
	}
	if !strings.Contains(text, `import "com/acme/platform/common/v1/common.proto";`) {
		t.Fatalf("domain proto missing common import: %s", text)
	}
	if !strings.Contains(text, "message UserContext {") {
		t.Fatalf("domain proto missing UserContext: %s", text)
	}
	// The context field in the envelope must reference the local UserContext.
	if !strings.Contains(text, "UserContext context = 6;") {
		t.Fatalf("domain proto event context field must reference UserContext: %s", text)
	}
	if !strings.Contains(text, "message SignedUpV1 {") {
		t.Fatalf("domain proto missing SignedUpV1: %s", text)
	}
	if !strings.Contains(text, "message SignedUpV1Properties {") {
		t.Fatalf("domain proto missing SignedUpV1Properties: %s", text)
	}
}

// TestRenderDomainProtoContextEnumIsEmitted verifies enum types in domain context are rendered.
func TestRenderDomainProtoContextEnumIsEmitted(t *testing.T) {
	reg := domainRegistryWithEnum()
	if len(reg.DomainSpecs) == 0 {
		t.Fatal("domainRegistryWithEnum() returned no DomainSpecs")
	}
	ds := reg.DomainSpecs[0]

	got, err := RenderDomainProto(reg.Namespace, "", ds)
	if err != nil {
		t.Fatalf("RenderDomainProto() error = %v, want nil", err)
	}
	text := string(got)

	if !strings.Contains(text, "enum Platform {") {
		t.Fatalf("domain proto missing Platform enum: %s", text)
	}
	if !strings.Contains(text, "PLATFORM_UNSPECIFIED = 0;") {
		t.Fatalf("domain proto missing Platform zero value: %s", text)
	}
	if !strings.Contains(text, "PLATFORM_IOS = 1;") {
		t.Fatalf("domain proto missing PLATFORM_IOS value: %s", text)
	}
}

// TestRenderPerDomainIsDeterministic verifies repeated renders produce identical output.
func TestRenderPerDomainIsDeterministic(t *testing.T) {
	reg := domainRegistry()

	first, err := RenderCommonProto(reg)
	if err != nil {
		t.Fatalf("RenderCommonProto() first error = %v", err)
	}
	second, err := RenderCommonProto(reg)
	if err != nil {
		t.Fatalf("RenderCommonProto() second error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("RenderCommonProto() repeated renders differ")
	}

	ds := reg.DomainSpecs[0]
	firstDomain, err := RenderDomainProto(reg.Namespace, "", ds)
	if err != nil {
		t.Fatalf("RenderDomainProto() first error = %v", err)
	}
	secondDomain, err := RenderDomainProto(reg.Namespace, "", ds)
	if err != nil {
		t.Fatalf("RenderDomainProto() second error = %v", err)
	}
	if !bytes.Equal(firstDomain, secondDomain) {
		t.Fatalf("RenderDomainProto() repeated renders differ")
	}
}

// TestRenderWritesOutputTree verifies the full output tree using the legacy Registry shape.
func TestRenderWritesOutputTree(t *testing.T) {
	outDir := t.TempDir()

	if err := Render(legacyRegistry(), outDir); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	wantPaths := []string{
		"buf.yaml",
		"buf.gen.yaml",
		"proto/com/acme/storefront/v1/events.proto",
		"openevents.metadata.yaml",
	}
	for _, relPath := range wantPaths {
		t.Run(relPath, func(t *testing.T) {
			fullPath := filepath.Join(outDir, filepath.FromSlash(relPath))
			if _, err := os.Stat(fullPath); err != nil {
				t.Fatalf("expected file %q not found: %v", relPath, err)
			}
		})
	}
}

func TestRenderRejectsInvalidProtoFilePaths(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "empty path",
			path: "",
			want: "must not be empty",
		},
		{
			name: "absolute path",
			path: "/tmp/outside.proto",
			want: "must be relative",
		},
		{
			name: "backslash separators",
			path: "com\\acme\\outside.proto",
			want: "must use slash-separated relative paths",
		},
		{
			name: "parent segment",
			path: "../outside.proto",
			want: "must not contain '..' segments",
		},
		{
			name: "path escapes root after clean",
			path: "com/acme/../../outside.proto",
			want: "must not contain '..' segments",
		},
		{
			name: "windows drive-qualified absolute path",
			path: "C:/outside.proto",
			want: "must not be drive-qualified",
		},
		{
			name: "windows drive-relative path",
			path: "C:outside.proto",
			want: "must not be drive-qualified",
		},
		{
			name: "lowercase windows drive-qualified absolute path",
			path: "z:/foo/bar.proto",
			want: "must not be drive-qualified",
		},
		{
			name: "uppercase windows drive-relative path",
			path: "Z:foo.proto",
			want: "must not be drive-qualified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Render(schemair.Registry{
				Namespace: "example.v1",
				Files: []schemair.File{
					{
						Path:    tt.path,
						Package: "example.v1",
						Messages: []schemair.Message{
							{
								Name: "Example",
								Fields: []schemair.Field{
									{Name: "name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}},
								},
							},
						},
					},
				},
			}, t.TempDir())
			if err == nil {
				t.Fatalf("Render() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), "invalid proto file path") {
				t.Fatalf("Render() error = %q, want invalid path context", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Render() error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestRenderFileEmitsGoPackageOption(t *testing.T) {
	got, err := RenderFile(schemair.File{
		Path:      "example/v1/events.proto",
		Package:   "example.v1",
		GoPackage: "github.com/acme/storefront/events",
		Messages: []schemair.Message{
			{Name: "Example"},
		},
	})
	if err != nil {
		t.Fatalf("RenderFile() error = %v, want nil", err)
	}

	text := string(got)
	if !strings.Contains(text, "package example.v1;\noption go_package = \"github.com/acme/storefront/events;events\";\n") {
		t.Fatalf("RenderFile() output missing go_package option:\n%s", text)
	}
}

func TestRenderFileRejectsInvalidGoPackage(t *testing.T) {
	tests := []struct {
		name      string
		goPackage string
		want      string
	}{
		{name: "keyword alias", goPackage: "github.com/acme/type", want: "keyword"},
		{name: "single segment", goPackage: "events", want: "at least one '.' or '/'"},
		{name: "semicolon", goPackage: "github.com/acme/storefront/events;evil", want: "invalid package.go"},
		{name: "newline", goPackage: "github.com/acme/storefront/events\nnext", want: "invalid package.go"},
		{name: "control", goPackage: "github.com/acme/storefront/events\x01", want: "invalid package.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderFile(schemair.File{
				Path:      "example/v1/events.proto",
				Package:   "example.v1",
				GoPackage: tt.goPackage,
				Messages:  []schemair.Message{{Name: "Example"}},
			})
			if err == nil {
				t.Fatalf("RenderFile() error = nil, want non-nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("RenderFile() error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestRenderFileEmitsOptionalScalarsAndNeverOptionalRepeatedFields(t *testing.T) {
	got, err := RenderFile(schemair.File{
		Path:    "example/v1/events.proto",
		Package: "example.v1",
		Messages: []schemair.Message{
			{
				Name: "Example",
				Fields: []schemair.Field{
					{Name: "name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
					{Name: "tags", Number: 2, Type: schemair.TypeRef{Scalar: "string"}, Optional: true, Repeated: true},
					{Name: "client", Number: 3, Type: schemair.TypeRef{Message: "Client"}, Optional: true},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderFile() error = %v, want nil", err)
	}

	text := string(got)
	if !strings.Contains(text, "  optional string name = 1;\n") {
		t.Fatalf("RenderFile() output missing optional scalar field:\n%s", text)
	}
	if !strings.Contains(text, "  repeated string tags = 2;\n") {
		t.Fatalf("RenderFile() output missing repeated scalar field without optional:\n%s", text)
	}
	if !strings.Contains(text, "  Client client = 3;\n") {
		t.Fatalf("RenderFile() output missing optional message field without optional label:\n%s", text)
	}
	if strings.Contains(text, "  optional Client client = 3;\n") {
		t.Fatalf("RenderFile() output rendered an optional message field with optional label:\n%s", text)
	}
	if strings.Contains(text, "optional repeated") || strings.Contains(text, "repeated optional") {
		t.Fatalf("RenderFile() output rendered an optional repeated field:\n%s", text)
	}
}

func TestRenderFileEmitsDescriptionComments(t *testing.T) {
	got, err := RenderFile(schemair.File{
		Path:    "example/v1/events.proto",
		Package: "example.v1",
		Messages: []schemair.Message{
			{
				Name:        "Example",
				Description: "Message line one.\nMessage line two.",
				Fields: []schemair.Field{
					{Name: "name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}, Optional: true, Description: "Field line one.\nField line two."},
					{Name: "version", Number: 2, Type: schemair.TypeRef{Scalar: "string"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderFile() error = %v, want nil", err)
	}

	text := string(got)
	if !strings.Contains(text, "// Message line one.\n// Message line two.\nmessage Example {\n") {
		t.Fatalf("RenderFile() output missing message comments:\n%s", text)
	}
	if !strings.Contains(text, "  // Field line one.\n  // Field line two.\n  optional string name = 1;\n") {
		t.Fatalf("RenderFile() output missing field comments:\n%s", text)
	}
	if !strings.Contains(text, "  optional string name = 1;\n  string version = 2;\n") {
		t.Fatalf("RenderFile() output inserted unexpected comments for empty descriptions:\n%s", text)
	}
	if strings.Contains(text, "  //\n") {
		t.Fatalf("RenderFile() output rendered empty comment lines:\n%s", text)
	}
}

func TestRenderFileEmitsEnumZeroValue(t *testing.T) {
	got, err := RenderFile(schemair.File{
		Path:    "example/v1/events.proto",
		Package: "example.v1",
		Messages: []schemair.Message{
			{
				Name: "Example",
				Enums: []schemair.Enum{
					{
						Name: "DeliveryStatus",
						Values: []schemair.EnumValue{
							{Name: "DELIVERY_STATUS_SENT", Number: 1},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderFile() error = %v, want nil", err)
	}

	text := string(got)
	if !strings.Contains(text, "    DELIVERY_STATUS_UNSPECIFIED = 0;\n") {
		t.Fatalf("RenderFile() output missing generated enum zero value:\n%s", text)
	}
	if !strings.Contains(text, "    DELIVERY_STATUS_SENT = 1;\n") {
		t.Fatalf("RenderFile() output missing IR enum value:\n%s", text)
	}
}

func TestRenderMetadataIsDeterministic(t *testing.T) {
	reg := legacyRegistry()

	first, err := RenderMetadata(reg)
	if err != nil {
		t.Fatalf("RenderMetadata() first error = %v, want nil", err)
	}
	second, err := RenderMetadata(reg)
	if err != nil {
		t.Fatalf("RenderMetadata() second error = %v, want nil", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("RenderMetadata() repeated renders differ\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestRenderFileIsDeterministic(t *testing.T) {
	file := legacyRegistry().Files[0]

	first, err := RenderFile(file)
	if err != nil {
		t.Fatalf("RenderFile() first error = %v, want nil", err)
	}
	second, err := RenderFile(file)
	if err != nil {
		t.Fatalf("RenderFile() second error = %v, want nil", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("RenderFile() repeated renders differ\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestRenderMetadataRejectsInvalidTypeRefs(t *testing.T) {
	tests := []struct {
		name  string
		field schemair.Field
		want  string
	}{
		{
			name:  "missing type",
			field: schemair.Field{Name: "broken", Number: 1},
			want:  "exactly one TypeRef",
		},
		{
			name: "ambiguous type",
			field: schemair.Field{
				Name:   "broken",
				Number: 1,
				Type:   schemair.TypeRef{Scalar: "string", Message: "Nested"},
			},
			want: "exactly one TypeRef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderMetadata(schemair.Registry{
				Namespace: "example.v1",
				Files: []schemair.File{
					{
						Path:    "example/v1/events.proto",
						Package: "example.v1",
						Messages: []schemair.Message{
							{Name: "Example", Fields: []schemair.Field{tt.field}},
						},
					},
				},
			})
			if err == nil {
				t.Fatalf("RenderMetadata() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), "Example.broken") || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("RenderMetadata() error = %q, want message containing field path and %q", err, tt.want)
			}
		})
	}
}

// domainRegistry returns a Registry with per-domain DomainSpecs for testing
// the T4 per-domain emission path.
func domainRegistry() schemair.Registry {
	return schemair.Registry{
		Namespace: "com.acme.platform",
		CommonSpec: schemair.CommonSpec{
			Client: schemair.Message{
				Name: "Client",
				Fields: []schemair.Field{
					{Name: "name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
					{Name: "version", Number: 2, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
				},
			},
		},
		DomainSpecs: []schemair.DomainSpec{
			{
				Name:          "user",
				ContextName:   "UserContext",
				ContextFields: []schemair.Field{},
				ContextEnums:  []schemair.Enum{},
				Events: []schemair.DomainEvent{
					{
						Envelope: schemair.Message{
							Name: "SignedUpV1",
							Fields: []schemair.Field{
								{Name: "event_name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}},
								{Name: "event_version", Number: 2, Type: schemair.TypeRef{Scalar: "integer"}},
								{Name: "event_id", Number: 3, Type: schemair.TypeRef{Scalar: "uuid"}},
								{Name: "event_ts", Number: 4, Type: schemair.TypeRef{Scalar: "timestamp"}},
								{Name: "client", Number: 5, Type: schemair.TypeRef{Message: "Client"}},
								{Name: "context", Number: 6, Type: schemair.TypeRef{Message: "UserContext"}},
								{Name: "properties", Number: 7, Type: schemair.TypeRef{Message: "SignedUpV1Properties"}},
							},
						},
						Properties: schemair.Message{
							Name: "SignedUpV1Properties",
							Fields: []schemair.Field{
								{Name: "email", Number: 1, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
							},
						},
					},
				},
			},
		},
	}
}

// domainRegistryWithEnum returns a Registry with a domain context that has an enum field.
func domainRegistryWithEnum() schemair.Registry {
	return schemair.Registry{
		Namespace: "com.acme.platform",
		CommonSpec: schemair.CommonSpec{
			Client: schemair.Message{
				Name: "Client",
				Fields: []schemair.Field{
					{Name: "name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
					{Name: "version", Number: 2, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
				},
			},
		},
		DomainSpecs: []schemair.DomainSpec{
			{
				Name:        "user",
				ContextName: "UserContext",
				ContextFields: []schemair.Field{
					{Name: "platform", Number: 1, Type: schemair.TypeRef{Enum: "Platform"}, Optional: true, Required: true},
				},
				ContextEnums: []schemair.Enum{
					{
						Name: "Platform",
						Values: []schemair.EnumValue{
							{Name: "PLATFORM_IOS", Original: "ios", Number: 1},
							{Name: "PLATFORM_ANDROID", Original: "android", Number: 2},
						},
					},
				},
				Events: []schemair.DomainEvent{
					{
						Envelope: schemair.Message{
							Name: "SignedUpV1",
							Fields: []schemair.Field{
								{Name: "event_name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}},
								{Name: "event_version", Number: 2, Type: schemair.TypeRef{Scalar: "integer"}},
								{Name: "event_id", Number: 3, Type: schemair.TypeRef{Scalar: "uuid"}},
								{Name: "event_ts", Number: 4, Type: schemair.TypeRef{Scalar: "timestamp"}},
								{Name: "client", Number: 5, Type: schemair.TypeRef{Message: "Client"}},
								{Name: "context", Number: 6, Type: schemair.TypeRef{Message: "UserContext"}},
								{Name: "properties", Number: 7, Type: schemair.TypeRef{Message: "SignedUpV1Properties"}},
							},
						},
						Properties: schemair.Message{
							Name:   "SignedUpV1Properties",
							Fields: []schemair.Field{},
						},
					},
				},
			},
		},
	}
}

// TestRenderDomainProtoArrayOfObjectEmitsNestedMessage verifies that a
// properties message with an array-of-object field renders a nested message
// and a repeated field referencing it.
func TestRenderDomainProtoArrayOfObjectEmitsNestedMessage(t *testing.T) {
	ds := schemair.DomainSpec{
		Name:          "device",
		ContextName:   "DeviceContext",
		ContextFields: []schemair.Field{},
		ContextEnums:  []schemair.Enum{},
		Events: []schemair.DomainEvent{
			{
				Envelope: schemair.Message{
					Name: "DiagnosticsStackUsageV1",
					Fields: []schemair.Field{
						{Name: "event_name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}},
						{Name: "event_version", Number: 2, Type: schemair.TypeRef{Scalar: "integer"}},
						{Name: "event_id", Number: 3, Type: schemair.TypeRef{Scalar: "uuid"}},
						{Name: "event_ts", Number: 4, Type: schemair.TypeRef{Scalar: "timestamp"}},
						{Name: "client", Number: 5, Type: schemair.TypeRef{Message: "Client"}},
						{Name: "context", Number: 6, Type: schemair.TypeRef{Message: "DeviceContext"}},
						{Name: "properties", Number: 7, Type: schemair.TypeRef{Message: "DiagnosticsStackUsageV1Properties"}},
					},
				},
				Properties: schemair.Message{
					Name: "DiagnosticsStackUsageV1Properties",
					Fields: []schemair.Field{
						{Name: "thread_count", Number: 1, Type: schemair.TypeRef{Scalar: "integer"}, Optional: true},
						{Name: "threads", Number: 2, Type: schemair.TypeRef{Message: "Threads"}, Repeated: true},
					},
					NestedMessages: []schemair.Message{
						{
							Name: "Threads",
							Fields: []schemair.Field{
								{Name: "name", Number: 1, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
								{Name: "stack_size_bytes", Number: 2, Type: schemair.TypeRef{Scalar: "integer"}, Optional: true},
								{Name: "state", Number: 3, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
							},
						},
					},
				},
			},
		},
	}

	got, err := RenderDomainProto("com.acme.platform", "", ds)
	if err != nil {
		t.Fatalf("RenderDomainProto() error = %v, want nil", err)
	}
	text := string(got)

	// The nested message must be emitted inside Properties.
	if !strings.Contains(text, "message Threads {") {
		t.Fatalf("domain proto missing nested Threads message:\n%s", text)
	}
	// The threads field must be repeated.
	if !strings.Contains(text, "repeated Threads threads = 2;") {
		t.Fatalf("domain proto missing repeated Threads field:\n%s", text)
	}
	// The nested message must contain the sub-fields.
	if !strings.Contains(text, "optional string name = 1;") {
		t.Fatalf("domain proto Threads nested message missing name field:\n%s", text)
	}
	if !strings.Contains(text, "optional int64 stack_size_bytes = 2;") {
		t.Fatalf("domain proto Threads nested message missing stack_size_bytes field:\n%s", text)
	}
}

// legacyRegistry returns a Registry using the legacy single-file Files shape for
// backward-compatibility tests (no DomainSpecs).
func legacyRegistry() schemair.Registry {
	return schemair.Registry{
		Namespace: "com.acme.storefront.v1",
		Files: []schemair.File{
			{
				Path:      "com/acme/storefront/v1/events.proto",
				Package:   "com.acme.storefront.v1",
				GoPackage: "github.com/acme/storefront/events",
				Messages: []schemair.Message{
					{
						Name:        "Client",
						Description: "Client application details.",
						Fields: []schemair.Field{
							{
								Name:        "name",
								Number:      1,
								Type:        schemair.TypeRef{Scalar: "string"},
								Optional:    true,
								Description: "Client display name.",
							},
							{Name: "version", Number: 2, Type: schemair.TypeRef{Scalar: "string"}, Optional: true},
						},
					},
					{
						Name:        "OrderCreated",
						Description: "Order creation event.",
						Fields: []schemair.Field{
							{Name: "event_id", Number: 1, Type: schemair.TypeRef{Scalar: "uuid"}, Required: true},
							{Name: "occurred_at", Number: 2, Type: schemair.TypeRef{Scalar: "timestamp"}, Required: true},
							{Name: "client", Number: 3, Type: schemair.TypeRef{Message: "Client"}, Optional: true},
							{Name: "items", Number: 4, Type: schemair.TypeRef{Message: "LineItem"}, Repeated: true},
							{Name: "status", Number: 5, Type: schemair.TypeRef{Enum: "OrderStatus"}, Optional: true},
							{Name: "priority", Number: 6, Type: schemair.TypeRef{Scalar: "integer"}},
							{Name: "total", Number: 7, Type: schemair.TypeRef{Scalar: "number"}, Optional: true},
							{Name: "discounted", Number: 8, Type: schemair.TypeRef{Scalar: "boolean"}, Optional: true},
							{Name: "order_date", Number: 9, Type: schemair.TypeRef{Scalar: "date"}, Optional: true},
						},
						Enums: []schemair.Enum{
							{
								Name: "OrderStatus",
								Values: []schemair.EnumValue{
									{Name: "ORDER_STATUS_CREATED", Original: "created", Number: 1},
									{Name: "ORDER_STATUS_PAID", Original: "paid", Number: 2},
								},
							},
						},
					},
					{
						Name: "LineItem",
						Fields: []schemair.Field{
							{Name: "sku", Number: 1, Type: schemair.TypeRef{Scalar: "string"}, Required: true},
							{Name: "quantity", Number: 2, Type: schemair.TypeRef{Scalar: "integer"}, Optional: true},
						},
					},
					{
						Name: "Catalog",
						Enums: []schemair.Enum{
							{Name: "CatalogState"},
						},
					},
				},
			},
		},
	}
}
