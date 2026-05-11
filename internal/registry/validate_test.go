package registry

import (
	"reflect"
	"testing"
)

func validRegistry() Registry {
	return Registry{
		Version:   SupportedVersion,
		Namespace: "com.example.product",
		Package: PackageConfig{
			Go:     "github.com/example/product/events",
			Python: "example_product.events",
		},
		Context: map[string]Field{
			"platform": {
				Name:   "platform",
				Type:   FieldTypeEnum,
				PII:    PIINone,
				Values: []string{"ios", "android", "web"},
			},
			"tenant_id": {
				Name: "tenant_id",
				Type: FieldTypeString,
				PII:  PIINone,
			},
		},
		Events: []Event{
			{
				Name:    "user.signed_up",
				Version: 1,
				Status:  "active",
				Properties: map[string]Field{
					"plan": {
						Name: "plan",
						Type: FieldTypeString,
						PII:  PIINone,
					},
					"profile": {
						Name: "profile",
						Type: FieldTypeObject,
						PII:  PIINone,
						Properties: map[string]Field{
							"age": {
								Name: "age",
								Type: FieldTypeInteger,
								PII:  PIINone,
							},
						},
					},
					"signup_method": {
						Name:   "signup_method",
						Type:   FieldTypeEnum,
						PII:    PIINone,
						Values: []string{"email", "google", "apple"},
					},
					"tags": {
						Name: "tags",
						Type: FieldTypeArray,
						PII:  PIINone,
						Items: &Field{
							Name: "items",
							Type: FieldTypeString,
							PII:  PIINone,
						},
					},
				},
			},
		},
	}
}

func TestValidateAcceptsValidRegistry(t *testing.T) {
	if diags := Validate(validRegistry()); diags.HasErrors() {
		t.Fatalf("Validate(validRegistry()) diagnostics = %v", diags)
	}
}

func TestValidateRequiresTopLevelFields(t *testing.T) {
	reg := validRegistry()
	reg.Version = ""
	reg.Namespace = ""

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "openevents", Message: "openevents is required"},
		{Location: "namespace", Message: "namespace is required"},
	})
}

func TestValidateRejectsUnsupportedVersion(t *testing.T) {
	reg := validRegistry()
	reg.Version = "9.9.9"

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "openevents", Message: "unsupported openevents version \"9.9.9\""},
	})
}

func TestValidateRejectsInvalidPackageNamesWhenPresent(t *testing.T) {
	reg := validRegistry()
	reg.Package.Go = "github.com/example/product events"
	reg.Package.Python = "example-product.events"

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "package.go", Message: "package.go must be a valid Go import path"},
		{Location: "package.python", Message: "package.python must be a valid Python package name"},
	})
}

func TestValidateRejectsInvalidEventName(t *testing.T) {
	reg := validRegistry()
	reg.Events[0].Name = "UserSignedUp"

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "events.UserSignedUp.name", Message: "event name must be lowercase dot-separated identifiers"},
	})
}

func TestValidateRejectsInvalidFieldName(t *testing.T) {
	reg := validRegistry()
	reg.Context["BadName"] = Field{Name: "BadName", Type: FieldTypeString, PII: PIINone}
	reg.Events[0].Properties["BadName"] = Field{Name: "BadName", Type: FieldTypeString, PII: PIINone}

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "context.BadName", Message: "field name must be snake_case"},
		{Location: "events.user.signed_up.properties.BadName", Message: "field name must be snake_case"},
	})
}

func TestValidateRejectsUnsupportedFieldType(t *testing.T) {
	reg := validRegistry()
	reg.Events[0].Properties["plan"] = Field{Name: "plan", Type: FieldType("money"), PII: PIINone}

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "events.user.signed_up.properties.plan.type", Message: "unsupported field type \"money\""},
	})
}

func TestValidateRejectsUnsupportedPII(t *testing.T) {
	reg := validRegistry()
	reg.Context["tenant_id"] = Field{Name: "tenant_id", Type: FieldTypeString, PII: PIIClassification("secret")}

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "context.tenant_id.pii", Message: "unsupported pii classification \"secret\""},
	})
}

func TestValidateRejectsBadEnumShapes(t *testing.T) {
	t.Run("empty values", func(t *testing.T) {
		reg := validRegistry()
		reg.Context["platform"] = Field{Name: "platform", Type: FieldTypeEnum, PII: PIINone}

		assertDiagnostics(t, Validate(reg), Diagnostics{
			{Location: "context.platform.values", Message: "enum fields must define at least one value"},
		})
	})

	t.Run("blank values", func(t *testing.T) {
		reg := validRegistry()
		reg.Context["platform"] = Field{Name: "platform", Type: FieldTypeEnum, PII: PIINone, Values: []string{""}}

		assertDiagnostics(t, Validate(reg), Diagnostics{
			{Location: "context.platform.values[0]", Message: "enum values must not be empty"},
		})
	})

	t.Run("duplicate values", func(t *testing.T) {
		reg := validRegistry()
		reg.Context["platform"] = Field{Name: "platform", Type: FieldTypeEnum, PII: PIINone, Values: []string{"ios", "ios"}}

		assertDiagnostics(t, Validate(reg), Diagnostics{
			{Location: "context.platform.values[1]", Message: "duplicate enum value \"ios\""},
		})
	})
}

func TestValidateRejectsMissingArrayItems(t *testing.T) {
	reg := validRegistry()
	reg.Events[0].Properties["tags"] = Field{Name: "tags", Type: FieldTypeArray, PII: PIINone}

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "events.user.signed_up.properties.tags.items", Message: "array fields must define items"},
	})
}

func TestValidateRejectsEmptyObjectProperties(t *testing.T) {
	reg := validRegistry()
	reg.Events[0].Properties["profile"] = Field{Name: "profile", Type: FieldTypeObject, PII: PIINone}

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "events.user.signed_up.properties.profile.properties", Message: "object fields must define properties"},
	})
}

func TestValidateRejectsDuplicateEventNameVersion(t *testing.T) {
	reg := validRegistry()
	reg.Events = append(reg.Events, reg.Events[0])

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "events[1]", Message: "duplicate event name/version \"user.signed_up@1\""},
	})
}

func TestValidateRejectsUnsupportedStatus(t *testing.T) {
	reg := validRegistry()
	reg.Events[0].Status = "paused"

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "events.user.signed_up.status", Message: "unsupported event status \"paused\""},
	})
}

func TestValidateRejectsNonPositiveVersion(t *testing.T) {
	for _, version := range []int{0, -1} {
		t.Run("version", func(t *testing.T) {
			reg := validRegistry()
			reg.Events[0].Version = version

			assertDiagnostics(t, Validate(reg), Diagnostics{
				{Location: "events.user.signed_up.version", Message: "event version must be positive"},
			})
		})
	}
}

func TestValidateRecursesIntoNestedFieldsInSortedOrder(t *testing.T) {
	reg := validRegistry()
	reg.Events[0].Properties["profile"] = Field{
		Name: "profile",
		Type: FieldTypeObject,
		PII:  PIINone,
		Properties: map[string]Field{
			"z_field": {Name: "z_field", Type: FieldTypeString, PII: PIIClassification("secret")},
			"a_field": {Name: "a_field", Type: FieldType("money"), PII: PIINone},
		},
	}

	assertDiagnostics(t, Validate(reg), Diagnostics{
		{Location: "events.user.signed_up.properties.profile.properties.a_field.type", Message: "unsupported field type \"money\""},
		{Location: "events.user.signed_up.properties.profile.properties.z_field.pii", Message: "unsupported pii classification \"secret\""},
	})
}

func assertDiagnostics(t *testing.T, got, want Diagnostics) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Validate() diagnostics = %#v, want %#v\nDiagnostics:\n%s", got, want, got.Error())
	}
}
