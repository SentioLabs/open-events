package registry_test

import (
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/sentiolabs/open-events/internal/registry/testfx"
)

// happyPathRoot builds a valid registry tree with two domains for use in tests.
func happyPathRoot(t *testing.T) string {
	t.Helper()
	return testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Owner("infra", "infra@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("user signed up").Done().
		Done().
		Domain("device").
		Owner("infra").
		Context("device_id", registry.FieldTypeString, true, registry.PIIPseudonymous).
		Action([]string{"info"}, "connected").Version(1).Status("active").Description("device connected").Done().
		Done().
		Write(t)
}

// loadOrFatal calls Load and fatals if there are load diagnostics.
func loadOrFatal(t *testing.T, root string) registry.Registry {
	t.Helper()
	reg, diags := registry.Load(root)
	if diags.HasErrors() {
		t.Fatalf("load failed: %v", diags.Error())
	}
	return reg
}

// --- Happy path ---

func TestValidate_HappyPath(t *testing.T) {
	reg := loadOrFatal(t, happyPathRoot(t))
	diags := registry.Validate(reg)
	if diags.HasErrors() {
		t.Fatalf("expected no diagnostics, got: %v", diags.Error())
	}
}

// --- Structural rules ---

func TestValidate_EmptyDomains(t *testing.T) {
	// A registry with no domains should produce an error.
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Write(t)

	reg := loadOrFatal(t, root)
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for empty domains, got none")
	}
	if !containsDiag(diags, "", "domain") {
		t.Fatalf("expected diagnostic about domains; got: %v", diags.Error())
	}
}

func TestValidate_NonSnakeCasePathSegment(t *testing.T) {
	// Domain names that violate snake_case should produce an error.
	// We build a registry manually to inject an event with a bad path segment.
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"UserDomain": {Name: "UserDomain", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "UserDomain.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "UserDomain",
				Path:    []string{"UserDomain", "auth"},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for non-snake_case path segment, got none")
	}
	if !containsDiag(diags, "", "snake_case") {
		t.Fatalf("expected diagnostic about snake_case; got: %v", diags.Error())
	}
}

func TestValidate_DepthTooShallow(t *testing.T) {
	// Path with only 1 segment (domain only, no category) means depth < 2.
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user"}, // only 1 segment = depth < 2
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for depth < 2, got none")
	}
	if !containsDiag(diags, "", "depth") {
		t.Fatalf("expected diagnostic about depth; got: %v", diags.Error())
	}
}

func TestValidate_DepthTooDeep(t *testing.T) {
	// Path with 5 segments means event name would have 6 parts = depth > 4.
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.a.b.c.d.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "a", "b", "c", "d"}, // 5 segments = depth > 4
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for depth > 4, got none")
	}
	if !containsDiag(diags, "", "depth") {
		t.Fatalf("expected diagnostic about depth; got: %v", diags.Error())
	}
}

// --- Referential rules ---

func TestValidate_UndeclaredDomainOwner(t *testing.T) {
	root := testfx.New().
		Owner("growth", "g@example.com").
		Domain("user").
		Owner("nonexistent").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "signup").Version(1).Status("active").Description("s").Done().
		Done().
		Write(t)

	reg := loadOrFatal(t, root)
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected validation error for undeclared domain owner")
	}
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "nonexistent") && strings.Contains(d.Location, "user/domain.yml") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected location user/domain.yml and 'nonexistent' in message; got %v", diags.Error())
	}
}

func TestValidate_UndeclaredEventOwner(t *testing.T) {
	// Event-level owner that doesn't match any declared owner slug.
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
				Owner:   "phantom-team",
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for undeclared event owner, got none")
	}
	if !containsDiag(diags, "", "phantom-team") {
		t.Fatalf("expected diagnostic mentioning 'phantom-team'; got: %v", diags.Error())
	}
}

// --- Uniqueness rules ---

func TestValidate_DuplicateComposedEventName(t *testing.T) {
	// Two events with the same name@version should produce a uniqueness error.
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
			},
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for duplicate composed event name, got none")
	}
	if !containsDiag(diags, "user.auth.signup@1", "") {
		t.Fatalf("expected diagnostic with location 'user.auth.signup@1'; got: %v", diags.Error())
	}
}

// --- Field-level rules (carried over) ---

func TestValidate_InvalidFieldType(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
				Properties: map[string]registry.Field{
					"plan": {Name: "plan", Type: registry.FieldType("money"), PII: registry.PIINone},
				},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for invalid field type, got none")
	}
	if !containsDiag(diags, "user/auth/signup.yml:properties.plan.type", "") {
		t.Fatalf("expected diagnostic with file path location; got: %v", diags.Error())
	}
}

func TestValidate_MissingArrayItems(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
				Properties: map[string]registry.Field{
					"tags": {Name: "tags", Type: registry.FieldTypeArray, PII: registry.PIINone},
				},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for missing array items, got none")
	}
	if !containsDiag(diags, "user/auth/signup.yml:properties.tags.items", "") {
		t.Fatalf("expected diagnostic with file path location for array items; got: %v", diags.Error())
	}
}

func TestValidate_MissingObjectProperties(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
				Properties: map[string]registry.Field{
					"profile": {Name: "profile", Type: registry.FieldTypeObject, PII: registry.PIINone},
				},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for missing object properties, got none")
	}
	if !containsDiag(diags, "user/auth/signup.yml:properties.profile.properties", "") {
		t.Fatalf("expected diagnostic with file path location for object properties; got: %v", diags.Error())
	}
}

func TestValidate_DuplicateEnumValues(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
				Properties: map[string]registry.Field{
					"method": {
						Name:   "method",
						Type:   registry.FieldTypeEnum,
						PII:    registry.PIINone,
						Values: []string{"email", "email"},
					},
				},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for duplicate enum values, got none")
	}
	if !containsDiag(diags, "user/auth/signup.yml:properties.method.values[1]", "") {
		t.Fatalf("expected diagnostic with file path location for duplicate enum; got: %v", diags.Error())
	}
}

func TestValidate_NonSnakeCaseFieldName(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
				Properties: map[string]registry.Field{
					"BadName": {Name: "BadName", Type: registry.FieldTypeString, PII: registry.PIINone},
				},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for non-snake_case field name, got none")
	}
	if !containsDiag(diags, "user/auth/signup.yml:properties.BadName", "") {
		t.Fatalf("expected diagnostic with file path location for bad field name; got: %v", diags.Error())
	}
}

// --- Domain context field-level rules ---

func TestValidate_DomainContextInvalidField(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {
				Name:  "user",
				Owner: "growth",
				Context: map[string]registry.Field{
					"platform": {Name: "platform", Type: registry.FieldTypeEnum, PII: registry.PIINone, Values: []string{}},
				},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for empty enum values in domain context, got none")
	}
	if !containsDiag(diags, "user/domain.yml:context.platform.values", "") {
		t.Fatalf("expected diagnostic with domain file path location; got: %v", diags.Error())
	}
}

// --- Version / Status / Action-name regression tests (CODEX-3) ---

func TestValidate_ZeroVersion(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 0, // invalid: must be positive
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for version 0, got none")
	}
	if !containsDiag(diags, "version", "positive") {
		t.Fatalf("expected diagnostic about positive version; got: %v", diags.Error())
	}
}

func TestValidate_NegativeVersion(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: -1, // invalid: must be positive
				Status:  "active",
				Domain:  "user",
				Path:    []string{"user", "auth"},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for negative version, got none")
	}
	if !containsDiag(diags, "version", "positive") {
		t.Fatalf("expected diagnostic about positive version; got: %v", diags.Error())
	}
}

func TestValidate_UnsupportedStatus(t *testing.T) {
	reg := registry.Registry{
		Owners: []registry.Owner{{Team: "growth"}},
		Domains: map[string]registry.Domain{
			"user": {Name: "user", Owner: "growth"},
		},
		Events: []registry.Event{
			{
				Name:    "user.auth.signup",
				Version: 1,
				Status:  "retired", // invalid status
				Domain:  "user",
				Path:    []string{"user", "auth"},
			},
		},
	}
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for unsupported status, got none")
	}
	if !containsDiag(diags, "status", "retired") {
		t.Fatalf("expected diagnostic mentioning 'retired' status; got: %v", diags.Error())
	}
}

func TestValidate_NonSnakeCaseActionFilename(t *testing.T) {
	root := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme_platform.events").
		Owner("growth", "growth@example.com").
		Language("go").
		Domain("user").
		Owner("growth").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"auth"}, "SignUp"). // non-snake_case action name
		Version(1).Status("active").Description("user signed up").Done().
		Done().
		Write(t)

	reg := loadOrFatal(t, root)
	diags := registry.Validate(reg)
	if !diags.HasErrors() {
		t.Fatal("expected error for non-snake_case action filename, got none")
	}
	if !containsDiag(diags, "", "snake_case") && !containsDiag(diags, "", "action") {
		t.Fatalf("expected diagnostic about snake_case action; got: %v", diags.Error())
	}
}

func TestValidate_ValidStatuses(t *testing.T) {
	for _, status := range []string{"active", "deprecated", "experimental"} {
		t.Run(status, func(t *testing.T) {
			reg := registry.Registry{
				Owners: []registry.Owner{{Team: "growth"}},
				Domains: map[string]registry.Domain{
					"user": {Name: "user", Owner: "growth"},
				},
				Events: []registry.Event{
					{
						Name:    "user.auth.signup",
						Version: 1,
						Status:  status,
						Domain:  "user",
						Path:    []string{"user", "auth"},
					},
				},
			}
			diags := registry.Validate(reg)
			if diags.HasErrors() {
				t.Fatalf("status %q should be valid but got: %v", status, diags.Error())
			}
		})
	}
}

// --- helpers ---

// containsDiag returns true if any diagnostic matches both location (substring) and message (substring).
// Pass empty string to skip checking that field.
func containsDiag(diags registry.Diagnostics, locationSubstr, messageSubstr string) bool {
	for _, d := range diags {
		locMatch := locationSubstr == "" || strings.Contains(d.Location, locationSubstr)
		msgMatch := messageSubstr == "" || strings.Contains(d.Message, messageSubstr)
		if locMatch && msgMatch {
			return true
		}
	}
	return false
}
