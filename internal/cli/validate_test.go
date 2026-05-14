package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/sentiolabs/open-events/internal/registry/testfx"
)

// buildValidateRegistry creates a minimal directory-form registry with two
// domains and two events for use in validate tests.
func buildValidateRegistry(t *testing.T) string {
	t.Helper()
	return testfx.New().
		Namespace("com.example.product").
		Package("github.com/example/product/events", "example_product.events").
		Owner("data-platform", "data-platform@example.com").
		Domain("user").
		Owner("data-platform").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signed_up").Version(1).Status("active").Description("User signed up.").Done().
		Done().
		Domain("search").
		Owner("data-platform").
		Context("user_id", registry.FieldTypeString, false, registry.PIIPseudonymous).
		Action([]string{"query"}, "query_submitted").Version(1).Status("active").Description("User submitted a search query.").Done().
		Done().
		Write(t)
}

func TestValidateCommandWithValidRegistry(t *testing.T) {
	registryPath := buildValidateRegistry(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"validate", registryPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}

	// Must contain event/domain counts in the new format.
	want := "ok: registry valid (2 events across 2 domains)"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want containing %q", got, want)
	}
}

func TestValidateCommandIgnoresLockFileInDirectory(t *testing.T) {
	registryPath := buildValidateRegistry(t)

	// Write a lock file first, then validate — lock file must be ignored.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	update := NewRootCommand(&stdout, &stderr)
	update.SetArgs([]string{"lock", "update", registryPath})
	if err := update.Execute(); err != nil {
		t.Fatalf("lock update Execute() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	validate := NewRootCommand(&stdout, &stderr)
	validate.SetArgs([]string{"validate", registryPath})
	if err := validate.Execute(); err != nil {
		t.Fatalf("validate Execute() error = %v", err)
	}
}

func TestValidateCommandWithInvalidPath(t *testing.T) {
	badPath := "../../examples/does-not-exist"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"validate", badPath})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute() error = nil, want non-nil")
	}

	if got := stderr.String(); !strings.Contains(got, badPath) {
		t.Fatalf("stderr = %q, want containing %q", got, badPath)
	}
}
