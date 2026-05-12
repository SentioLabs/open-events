package protogen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/schemair"
)

func TestRenderGoldenFiles(t *testing.T) {
	reg := demoRegistry()

	protoBytes, err := RenderFile(reg.Files[0])
	if err != nil {
		t.Fatalf("RenderFile() error = %v, want nil", err)
	}
	assertGoldenBytes(t, "demo.golden.proto", protoBytes)

	assertGoldenBytes(t, "buf.golden.yaml", RenderBufYAML())
	assertGoldenBytes(t, "buf.gen.golden.yaml", RenderBufGenYAML())

	metadataBytes, err := RenderMetadata(reg)
	if err != nil {
		t.Fatalf("RenderMetadata() error = %v, want nil", err)
	}
	assertGoldenBytes(t, "metadata.golden.yaml", metadataBytes)
}

func TestRenderWritesOutputTree(t *testing.T) {
	outDir := t.TempDir()

	if err := Render(demoRegistry(), outDir); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	wantFiles := map[string]string{
		"buf.yaml":     "buf.golden.yaml",
		"buf.gen.yaml": "buf.gen.golden.yaml",
		"proto/com/acme/storefront/v1/events.proto": "demo.golden.proto",
		"openevents.metadata.yaml":                  "metadata.golden.yaml",
	}
	for relPath, goldenName := range wantFiles {
		t.Run(relPath, func(t *testing.T) {
			got, err := os.ReadFile(filepath.Join(outDir, filepath.FromSlash(relPath)))
			if err != nil {
				t.Fatalf("read rendered file %q: %v", relPath, err)
			}
			assertGoldenBytes(t, goldenName, got)
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
	reg := demoRegistry()

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
	file := demoRegistry().Files[0]

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

func demoRegistry() schemair.Registry {
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

func assertGoldenBytes(t *testing.T, name string, got []byte) {
	t.Helper()

	want, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden %q: %v", name, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s mismatch\nwant:\n%s\ngot:\n%s", name, want, got)
	}
}
