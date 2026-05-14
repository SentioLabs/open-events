package registry_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/sentiolabs/open-events/internal/registry/testfx"
)

// TestLoad_SingleDomainHappyPath exercises a single domain with one action.
func TestLoad_SingleDomainHappyPath(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("user signup").Done().
		Done().
		Write(t)

	reg, diags := registry.Load(root)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags.Error())
	}
	if len(reg.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(reg.Events))
	}
	if reg.Events[0].Name != "user.auth.signup" {
		t.Errorf("expected user.auth.signup, got %q", reg.Events[0].Name)
	}
	if reg.Events[0].Domain != "user" {
		t.Errorf("expected Domain=user, got %q", reg.Events[0].Domain)
	}
	if got := strings.Join(reg.Events[0].Path, "/"); got != "user/auth" {
		t.Errorf("expected Path=user/auth, got %q", got)
	}
	if len(reg.Domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(reg.Domains))
	}
	if _, ok := reg.Domains["user"]; !ok {
		t.Errorf("expected Domains[\"user\"] to be populated")
	}
}

// TestLoad_TwoDomainHappyPath exercises two domains, verifying alphabetical event ordering.
func TestLoad_TwoDomainHappyPath(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Owner("device-platform", "device@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("signup").Done().
		Done().
		Domain("device").
		Owner("device-platform").
		Context("device_id", registry.FieldTypeString, true, registry.PIIPseudonymous).
		Action([]string{"info"}, "hardware").Version(1).Status("active").Description("hw").Done().
		Done().
		Write(t)

	reg, diags := registry.Load(root)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags.Error())
	}
	if len(reg.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(reg.Events))
	}
	// Events sorted alphabetically: device.info.hardware < user.auth.signup
	if reg.Events[0].Name != "device.info.hardware" {
		t.Errorf("expected device.info.hardware first, got %q", reg.Events[0].Name)
	}
	if reg.Events[0].Domain != "device" {
		t.Errorf("expected Domain=device, got %q", reg.Events[0].Domain)
	}
	if got := strings.Join(reg.Events[0].Path, "/"); got != "device/info" {
		t.Errorf("expected Path=device/info, got %q", got)
	}
	if len(reg.Domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(reg.Domains))
	}
}

// TestLoad_Depth4Subcategory exercises a deeply nested category path.
func TestLoad_Depth4Subcategory(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("infra", "infra@example.com").
		Language("go").
		Domain("device").
		Owner("infra").
		Context("device_id", registry.FieldTypeString, true, registry.PIIPseudonymous).
		Action([]string{"info", "diagnostics"}, "stack_usage").Version(1).Status("active").Description("stack usage").Done().
		Done().
		Write(t)

	reg, diags := registry.Load(root)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags.Error())
	}
	if len(reg.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(reg.Events))
	}
	if reg.Events[0].Name != "device.info.diagnostics.stack_usage" {
		t.Errorf("expected device.info.diagnostics.stack_usage, got %q", reg.Events[0].Name)
	}
	if reg.Events[0].Domain != "device" {
		t.Errorf("expected Domain=device, got %q", reg.Events[0].Domain)
	}
	if got := strings.Join(reg.Events[0].Path, "/"); got != "device/info/diagnostics" {
		t.Errorf("expected Path=device/info/diagnostics, got %q", got)
	}
}

// TestLoad_MissingOpenEventsYAML checks that a directory without openevents.yaml returns an error.
func TestLoad_MissingOpenEventsYAML(t *testing.T) {
	root := t.TempDir()
	_, diags := registry.Load(root)
	if !diags.HasErrors() {
		t.Fatal("expected error diagnostic for missing openevents.yaml, got none")
	}
	if !strings.Contains(diags.Error(), "openevents.yaml") {
		t.Errorf("expected error mentioning openevents.yaml, got: %v", diags.Error())
	}
}

// TestLoad_MissingDomainYML checks that a top-level directory missing domain.yml returns an error.
func TestLoad_MissingDomainYML(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("user signup").Done().
		Done().
		Write(t)

	// Remove domain.yml to trigger the missing domain.yml error
	if err := os.Remove(filepath.Join(root, "user", "domain.yml")); err != nil {
		t.Fatalf("failed to remove domain.yml: %v", err)
	}

	_, diags := registry.Load(root)
	if !diags.HasErrors() {
		t.Fatal("expected error diagnostic for missing domain.yml, got none")
	}
	if !strings.Contains(diags.Error(), "domain.yml") {
		t.Errorf("expected error mentioning domain.yml, got: %v", diags.Error())
	}
}

// TestLoad_SingleFileRejection checks that passing a file path returns a clear error.
func TestLoad_SingleFileRejection(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("signup").Done().
		Done().
		Write(t)

	// Pass the openevents.yaml file directly instead of the directory
	_, diags := registry.Load(filepath.Join(root, "openevents.yaml"))
	if !diags.HasErrors() {
		t.Fatal("expected error diagnostic for single-file invocation, got none")
	}
	if !strings.Contains(diags.Error(), "expected directory containing openevents.yaml") {
		t.Errorf("expected 'expected directory containing openevents.yaml', got: %v", diags.Error())
	}
}

// TestLoad_MalformedActionYAML checks that a malformed action file returns an error.
func TestLoad_MalformedActionYAML(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("signup").Done().
		Done().
		Write(t)

	// Overwrite the action file with invalid YAML
	badYAML := []byte("version: [invalid\n")
	if err := os.WriteFile(filepath.Join(root, "user", "auth", "signup.yml"), badYAML, 0o644); err != nil {
		t.Fatalf("failed to write malformed YAML: %v", err)
	}

	_, diags := registry.Load(root)
	if !diags.HasErrors() {
		t.Fatal("expected error diagnostic for malformed action YAML, got none")
	}
}

// TestLoad_DomainContextPopulated verifies domain context fields are loaded into Registry.Domains.
func TestLoad_DomainContextPopulated(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Context("session_id", registry.FieldTypeUUID, false, registry.PIIPseudonymous).
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("signup").Done().
		Done().
		Write(t)

	reg, diags := registry.Load(root)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags.Error())
	}
	domain, ok := reg.Domains["user"]
	if !ok {
		t.Fatal("expected Domains[\"user\"] to be populated")
	}
	if len(domain.Context) != 2 {
		t.Fatalf("expected 2 context fields in domain, got %d", len(domain.Context))
	}
	if _, ok := domain.Context["tenant_id"]; !ok {
		t.Errorf("expected tenant_id in domain context")
	}
}

// TestLoad_EnumObjectArrayFields exercises enum, object, and array-of-object field shapes
// end-to-end: build → load → verify Field.Values, Field.Properties, Field.Items.
func TestLoad_EnumObjectArrayFields(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("eng", "eng@example.com").
		Language("go").
		Domain("device").
		Owner("eng").
		Context("device_id", registry.FieldTypeString, true, registry.PIIPseudonymous).
		ContextEnum("platform", true, registry.PIINone, "ios", "android", "web").
		Action([]string{"diagnostics"}, "stack_usage").
		Version(1).Status("active").Description("stack usage snapshot").
		Property("thread_count", registry.FieldTypeInteger, true, registry.PIINone).
		PropertyEnum("breach_type", true, registry.PIINone, "over", "under").
		PropertyObject("eeprom_format_version", true, registry.PIINone,
			testfx.SubField("major", registry.FieldTypeInteger, true, registry.PIINone),
			testfx.SubField("minor", registry.FieldTypeInteger, true, registry.PIINone),
		).
		PropertyArrayOfObject("threads", true, registry.PIINone,
			testfx.SubField("name", registry.FieldTypeString, true, registry.PIINone),
			testfx.SubField("stack_size_bytes", registry.FieldTypeInteger, true, registry.PIINone),
		).
		Done().
		Done().
		Write(t)

	reg, diags := registry.Load(root)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags.Error())
	}
	if len(reg.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(reg.Events))
	}

	// Verify domain context enum field.
	domain := reg.Domains["device"]
	platformField, ok := domain.Context["platform"]
	if !ok {
		t.Fatal("expected platform context field")
	}
	if platformField.Type != registry.FieldTypeEnum {
		t.Errorf("expected platform type=enum, got %q", platformField.Type)
	}
	if len(platformField.Values) != 3 {
		t.Errorf("expected 3 enum values, got %d: %v", len(platformField.Values), platformField.Values)
	}

	props := reg.Events[0].Properties

	// Verify enum property.
	breachField, ok := props["breach_type"]
	if !ok {
		t.Fatal("expected breach_type property")
	}
	if breachField.Type != registry.FieldTypeEnum {
		t.Errorf("expected breach_type type=enum, got %q", breachField.Type)
	}
	if len(breachField.Values) != 2 || breachField.Values[0] != "over" || breachField.Values[1] != "under" {
		t.Errorf("expected breach_type values=[over, under], got %v", breachField.Values)
	}

	// Verify object property.
	eepromField, ok := props["eeprom_format_version"]
	if !ok {
		t.Fatal("expected eeprom_format_version property")
	}
	if eepromField.Type != registry.FieldTypeObject {
		t.Errorf("expected eeprom_format_version type=object, got %q", eepromField.Type)
	}
	if len(eepromField.Properties) != 2 {
		t.Errorf("expected 2 object properties, got %d", len(eepromField.Properties))
	}
	if _, ok := eepromField.Properties["major"]; !ok {
		t.Errorf("expected major sub-property")
	}
	if _, ok := eepromField.Properties["minor"]; !ok {
		t.Errorf("expected minor sub-property")
	}

	// Verify array-of-object property.
	threadsField, ok := props["threads"]
	if !ok {
		t.Fatal("expected threads property")
	}
	if threadsField.Type != registry.FieldTypeArray {
		t.Errorf("expected threads type=array, got %q", threadsField.Type)
	}
	if threadsField.Items == nil {
		t.Fatal("expected threads.items to be set")
	}
	if threadsField.Items.Type != registry.FieldTypeObject {
		t.Errorf("expected threads.items type=object, got %q", threadsField.Items.Type)
	}
	if len(threadsField.Items.Properties) != 2 {
		t.Errorf("expected 2 thread sub-properties, got %d", len(threadsField.Items.Properties))
	}
}

// TestLoad_ActionPropertiesPopulated verifies action properties are loaded into event Properties.
func TestLoad_ActionPropertiesPopulated(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Action([]string{"auth"}, "signup").
		Version(1).Status("active").Description("signup").
		Property("method", registry.FieldTypeString, true, registry.PIINone).
		Property("referral_code", registry.FieldTypeString, false, registry.PIINone).
		Done().
		Done().
		Write(t)

	reg, diags := registry.Load(root)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", diags.Error())
	}
	if len(reg.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(reg.Events))
	}
	if len(reg.Events[0].Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d: %v", len(reg.Events[0].Properties), reg.Events[0].Properties)
	}
	if _, ok := reg.Events[0].Properties["method"]; !ok {
		t.Errorf("expected method property")
	}
}
