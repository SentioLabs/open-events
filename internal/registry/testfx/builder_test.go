package testfx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/sentiolabs/open-events/internal/registry/testfx"
)

func TestBuilder_WritesExpectedTree(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Language("go").
		Language("python").
		Domain("user").
		Description("user events").
		Owner("growth").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signup").
		Version(1).
		Status("active").
		Description("signup").
		Property("method", registry.FieldTypeString, true, registry.PIINone).
		Done().
		Done().
		Write(t)
	for _, rel := range []string{"openevents.yaml", "user/domain.yml", "user/auth/signup.yml"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}
