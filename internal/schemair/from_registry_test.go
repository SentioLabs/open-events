package schemair

import (
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
)

func TestFromRegistryLowersDemoShape(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context: map[string]registry.Field{
			"tenant_id": {
				Name:        "tenant_id",
				Type:        registry.FieldTypeString,
				Required:    true,
				Description: "Stable tenant identifier.",
			},
			"platform": {
				Name:     "platform",
				Type:     registry.FieldTypeEnum,
				Required: true,
				Values:   []string{"ios", "android", "web"},
			},
			"session_id": {
				Name:     "session_id",
				Type:     registry.FieldTypeString,
				Required: false,
			},
		},
		Events: []registry.Event{
			{
				Name:        "checkout.completed",
				Version:     1,
				Description: "User completed checkout and payment was accepted.",
				Properties: map[string]registry.Field{
					"order_id": {
						Name:     "order_id",
						Type:     registry.FieldTypeString,
						Required: true,
					},
					"payment_method": {
						Name:     "payment_method",
						Type:     registry.FieldTypeEnum,
						Required: true,
						Values:   []string{"card", "apple_pay", "google_pay"},
					},
					"coupon_code": {
						Name:     "coupon_code",
						Type:     registry.FieldTypeString,
						Required: false,
					},
				},
			},
			{
				Name:        "search.performed",
				Version:     1,
				Description: "User submitted a storefront search query.",
				Properties: map[string]registry.Field{
					"query": {
						Name:     "query",
						Type:     registry.FieldTypeString,
						Required: true,
					},
					"filters": {
						Name:     "filters",
						Type:     registry.FieldTypeArray,
						Required: false,
						Items: &registry.Field{
							Type: registry.FieldTypeString,
						},
					},
				},
			},
		},
	}

	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"tenant_id":  {StableID: "tenant_id", ProtoNumber: 2},
			"session_id": {StableID: "session_id", ProtoNumber: 3},
			"platform":   {StableID: "platform", ProtoNumber: 4},
		},
		Events: map[string]LockedEvent{
			"checkout.completed@1": {
				Envelope: map[string]LockedField{
					"event_name":    {StableID: "event_name", ProtoNumber: 91},
					"event_version": {StableID: "event_version", ProtoNumber: 92},
					"event_id":      {StableID: "event_id", ProtoNumber: 93},
					"event_ts":      {StableID: "event_ts", ProtoNumber: 94},
					"client":        {StableID: "client", ProtoNumber: 95},
					"context":       {StableID: "context", ProtoNumber: 96},
					"properties":    {StableID: "properties", ProtoNumber: 97},
				},
				Properties: map[string]LockedField{
					"payment_method": {StableID: "payment_method", ProtoNumber: 1},
					"coupon_code":    {StableID: "coupon_code", ProtoNumber: 2},
					"order_id":       {StableID: "order_id", ProtoNumber: 3},
				},
			},
			"search.performed@1": {
				Properties: map[string]LockedField{
					"query":   {StableID: "query", ProtoNumber: 1},
					"filters": {StableID: "filters", ProtoNumber: 2},
				},
			},
		},
	}

	got, err := FromRegistry(reg, lock)
	if err != nil {
		t.Fatalf("FromRegistry() error = %v, want nil", err)
	}

	if got.Namespace != "com.acme.storefront" {
		t.Fatalf("Registry.Namespace = %q, want %q", got.Namespace, "com.acme.storefront")
	}
	if len(got.Files) != 1 {
		t.Fatalf("len(Registry.Files) = %d, want 1", len(got.Files))
	}

	file := got.Files[0]
	if file.Package != "com.acme.storefront.v1" {
		t.Fatalf("File.Package = %q, want %q", file.Package, "com.acme.storefront.v1")
	}
	if file.Path != "com/acme/storefront/v1/events.proto" {
		t.Fatalf("File.Path = %q, want %q", file.Path, "com/acme/storefront/v1/events.proto")
	}

	if len(file.Messages) != 6 {
		t.Fatalf("len(File.Messages) = %d, want 6", len(file.Messages))
	}
	if file.Messages[0].Name != "Client" {
		t.Fatalf("Messages[0].Name = %q, want %q", file.Messages[0].Name, "Client")
	}
	if file.Messages[1].Name != "Context" {
		t.Fatalf("Messages[1].Name = %q, want %q", file.Messages[1].Name, "Context")
	}
	if file.Messages[2].Name != "CheckoutCompletedV1" {
		t.Fatalf("Messages[2].Name = %q, want %q", file.Messages[2].Name, "CheckoutCompletedV1")
	}
	if file.Messages[3].Name != "CheckoutCompletedV1Properties" {
		t.Fatalf("Messages[3].Name = %q, want %q", file.Messages[3].Name, "CheckoutCompletedV1Properties")
	}
	if file.Messages[4].Name != "SearchPerformedV1" {
		t.Fatalf("Messages[4].Name = %q, want %q", file.Messages[4].Name, "SearchPerformedV1")
	}
	if file.Messages[5].Name != "SearchPerformedV1Properties" {
		t.Fatalf("Messages[5].Name = %q, want %q", file.Messages[5].Name, "SearchPerformedV1Properties")
	}

	envelope := file.Messages[2]
	wantEnvelopeNumbers := map[string]int{
		"event_name":    1,
		"event_version": 2,
		"event_id":      3,
		"event_ts":      4,
		"client":        5,
		"context":       6,
		"properties":    7,
	}
	if len(envelope.Fields) != len(wantEnvelopeNumbers) {
		t.Fatalf("len(Envelope.Fields) = %d, want %d", len(envelope.Fields), len(wantEnvelopeNumbers))
	}
	for _, field := range envelope.Fields {
		wantNumber, ok := wantEnvelopeNumbers[field.Name]
		if !ok {
			t.Fatalf("unexpected envelope field %q", field.Name)
		}
		if field.Number != wantNumber {
			t.Fatalf("envelope field %q number = %d, want %d", field.Name, field.Number, wantNumber)
		}
	}

	context := file.Messages[1]
	if len(context.Fields) != 3 {
		t.Fatalf("len(Context.Fields) = %d, want 3", len(context.Fields))
	}
	if context.Fields[0].Name != "platform" || context.Fields[0].Number != 4 {
		t.Fatalf("Context field[0] = %#v, want name=platform number=4", context.Fields[0])
	}
	if !context.Fields[0].Optional {
		t.Fatalf("Context.platform Optional = false, want true")
	}
	if !context.Fields[0].Required {
		t.Fatalf("Context.platform Required = false, want true")
	}
	if context.Fields[0].Type.Enum != "Platform" {
		t.Fatalf("Context.platform enum type = %q, want %q", context.Fields[0].Type.Enum, "Platform")
	}
	if context.Fields[1].Name != "session_id" || context.Fields[1].Number != 3 {
		t.Fatalf("Context field[1] = %#v, want name=session_id number=3", context.Fields[1])
	}
	if !context.Fields[1].Optional {
		t.Fatalf("Context.session_id Optional = false, want true")
	}
	if context.Fields[1].Required {
		t.Fatalf("Context.session_id Required = true, want false")
	}
	if context.Fields[2].Name != "tenant_id" || context.Fields[2].Number != 2 {
		t.Fatalf("Context field[2] = %#v, want name=tenant_id number=2", context.Fields[2])
	}
	if !context.Fields[2].Optional {
		t.Fatalf("Context.tenant_id Optional = false, want true")
	}
	if !context.Fields[2].Required {
		t.Fatalf("Context.tenant_id Required = false, want true")
	}

	if len(context.Enums) != 1 {
		t.Fatalf("len(Context.Enums) = %d, want 1", len(context.Enums))
	}
	if context.Enums[0].Name != "Platform" {
		t.Fatalf("Context enum name = %q, want %q", context.Enums[0].Name, "Platform")
	}
	if len(context.Enums[0].Values) != 3 {
		t.Fatalf("len(Context.Enums[0].Values) = %d, want 3", len(context.Enums[0].Values))
	}
	if context.Enums[0].Values[0].Name != "PLATFORM_IOS" {
		t.Fatalf("Context enum value[0].Name = %q, want %q", context.Enums[0].Values[0].Name, "PLATFORM_IOS")
	}

	checkoutProps := file.Messages[3]
	if checkoutProps.Fields[0].Name != "coupon_code" || checkoutProps.Fields[0].Number != 2 {
		t.Fatalf("Checkout properties field[0] = %#v, want name=coupon_code number=2", checkoutProps.Fields[0])
	}
	if !checkoutProps.Fields[0].Optional {
		t.Fatalf("Checkout coupon_code Optional = false, want true")
	}
	if checkoutProps.Fields[0].Required {
		t.Fatalf("Checkout coupon_code Required = true, want false")
	}
	if checkoutProps.Fields[1].Name != "order_id" || checkoutProps.Fields[1].Number != 3 {
		t.Fatalf("Checkout properties field[1] = %#v, want name=order_id number=3", checkoutProps.Fields[1])
	}
	if !checkoutProps.Fields[1].Optional {
		t.Fatalf("Checkout order_id Optional = false, want true")
	}
	if !checkoutProps.Fields[1].Required {
		t.Fatalf("Checkout order_id Required = false, want true")
	}
	if checkoutProps.Fields[2].Name != "payment_method" || checkoutProps.Fields[2].Number != 1 {
		t.Fatalf("Checkout properties field[2] = %#v, want name=payment_method number=1", checkoutProps.Fields[2])
	}
	if !checkoutProps.Fields[2].Optional {
		t.Fatalf("Checkout payment_method Optional = false, want true")
	}
	if !checkoutProps.Fields[2].Required {
		t.Fatalf("Checkout payment_method Required = false, want true")
	}
	if checkoutProps.Fields[2].Type.Enum != "PaymentMethod" {
		t.Fatalf("Checkout payment_method enum type = %q, want %q", checkoutProps.Fields[2].Type.Enum, "PaymentMethod")
	}
	if len(checkoutProps.Enums) != 1 {
		t.Fatalf("len(Checkout properties enums) = %d, want 1", len(checkoutProps.Enums))
	}
	if len(checkoutProps.Enums[0].Values) != 3 {
		t.Fatalf("len(Checkout enum values) = %d, want 3", len(checkoutProps.Enums[0].Values))
	}
	if checkoutProps.Enums[0].Values[1].Name != "PAYMENT_METHOD_APPLE_PAY" {
		t.Fatalf("Checkout enum value[1].Name = %q, want %q", checkoutProps.Enums[0].Values[1].Name, "PAYMENT_METHOD_APPLE_PAY")
	}

	searchProps := file.Messages[5]
	if searchProps.Fields[0].Name != "filters" {
		t.Fatalf("Search properties field[0].Name = %q, want %q", searchProps.Fields[0].Name, "filters")
	}
	if !searchProps.Fields[0].Repeated {
		t.Fatalf("Search filters Repeated = false, want true")
	}
	if searchProps.Fields[0].Optional {
		t.Fatalf("Search filters Optional = true, want false")
	}
	if searchProps.Fields[0].Type.Scalar != "string" {
		t.Fatalf("Search filters scalar type = %q, want %q", searchProps.Fields[0].Type.Scalar, "string")
	}
}

func TestFromRegistryRejectsMissingLockEntries(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context: map[string]registry.Field{
			"tenant_id": {Name: "tenant_id", Type: registry.FieldTypeString},
		},
		Events: []registry.Event{{
			Name:    "checkout.completed",
			Version: 1,
			Properties: map[string]registry.Field{
				"order_id": {Name: "order_id", Type: registry.FieldTypeString},
			},
		}},
	}

	_, err := FromRegistry(reg, Lock{Version: 1})
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want missing lock error")
	}
	if !strings.Contains(err.Error(), "schema lock is missing context.tenant_id") {
		t.Fatalf("FromRegistry() error = %q, want missing context lock entry", err)
	}
}

func TestFromRegistryRejectsMissingPropertyLockEntries(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context: map[string]registry.Field{
			"tenant_id": {Name: "tenant_id", Type: registry.FieldTypeString},
		},
		Events: []registry.Event{{
			Name:    "checkout.completed",
			Version: 1,
			Properties: map[string]registry.Field{
				"order_id": {Name: "order_id", Type: registry.FieldTypeString},
			},
		}},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"tenant_id": {StableID: "tenant_id", ProtoNumber: 1},
		},
		Events: map[string]LockedEvent{
			"checkout.completed@1": {
				Properties: map[string]LockedField{},
			},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want missing property lock error")
	}
	if !strings.Contains(err.Error(), "schema lock is missing events.checkout.completed@1.properties.order_id") {
		t.Fatalf("FromRegistry() error = %q, want missing property lock entry", err)
	}
}

func TestFromRegistryRejectsUnsupportedArrayShapes(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context:   map[string]registry.Field{},
		Events: []registry.Event{{
			Name:    "checkout.completed",
			Version: 1,
			Properties: map[string]registry.Field{
				"tags": {
					Name: "tags",
					Type: registry.FieldTypeArray,
					Items: &registry.Field{
						Type:   registry.FieldTypeEnum,
						Values: []string{"a", "b"},
					},
				},
			},
		}},
	}
	lock := Lock{
		Version: 1,
		Events: map[string]LockedEvent{
			"checkout.completed@1": {
				Properties: map[string]LockedField{
					"tags": {StableID: "tags", ProtoNumber: 1},
				},
			},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want unsupported array shape error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "array") || !strings.Contains(strings.ToLower(err.Error()), "enum") {
		t.Fatalf("FromRegistry() error = %q, want actionable array enum error", err)
	}
}

func TestFromRegistryRejectsInvalidLockNumbers(t *testing.T) {
	tests := []struct {
		name   string
		number int
		want   string
	}{
		{name: "zero", number: 0, want: ">= 1"},
		{name: "reserved range", number: 19000, want: "reserved range"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := registry.Registry{
				Namespace: "com.acme.storefront",
				Context: map[string]registry.Field{
					"tenant_id": {Name: "tenant_id", Type: registry.FieldTypeString},
				},
				Events: []registry.Event{{Name: "test", Version: 1, Properties: map[string]registry.Field{}}},
			}
			lock := Lock{
				Version: 1,
				Context: map[string]LockedField{
					"tenant_id": {StableID: "tenant_id", ProtoNumber: tt.number},
				},
				Events: map[string]LockedEvent{
					"test@1": {Properties: map[string]LockedField{}},
				},
			}

			_, err := FromRegistry(reg, lock)
			if err == nil {
				t.Fatalf("FromRegistry() error = nil, want proto number validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("FromRegistry() error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestFromRegistryRejectsStableIDMismatch(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context: map[string]registry.Field{
			"tenant_id": {Name: "tenant_id", Type: registry.FieldTypeString},
		},
		Events: []registry.Event{{Name: "test", Version: 1, Properties: map[string]registry.Field{}}},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"tenant_id": {StableID: "wrong_name", ProtoNumber: 1},
		},
		Events: map[string]LockedEvent{
			"test@1": {Properties: map[string]LockedField{}},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want StableID mismatch error")
	}
	if !strings.Contains(err.Error(), "tenant_id") || !strings.Contains(err.Error(), "wrong_name") {
		t.Fatalf("FromRegistry() error = %q, want both field names mentioned", err)
	}
}

func TestFromRegistryRejectsDuplicateNumbers(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context: map[string]registry.Field{
			"tenant_id": {Name: "tenant_id", Type: registry.FieldTypeString},
			"user_id":   {Name: "user_id", Type: registry.FieldTypeString},
		},
		Events: []registry.Event{{Name: "test", Version: 1, Properties: map[string]registry.Field{}}},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"tenant_id": {StableID: "tenant_id", ProtoNumber: 1},
			"user_id":   {StableID: "user_id", ProtoNumber: 1},
		},
		Events: map[string]LockedEvent{
			"test@1": {Properties: map[string]LockedField{}},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want duplicate number error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("FromRegistry() error = %q, want duplicate mention", err)
	}
}

func TestFromRegistryRejectsReservedFieldNames(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context: map[string]registry.Field{
			"message": {Name: "message", Type: registry.FieldTypeString},
		},
		Events: []registry.Event{{Name: "test", Version: 1, Properties: map[string]registry.Field{}}},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"message": {StableID: "message", ProtoNumber: 1},
		},
		Events: map[string]LockedEvent{
			"test@1": {Properties: map[string]LockedField{}},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want reserved keyword error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "reserved") || !strings.Contains(strings.ToLower(err.Error()), "keyword") {
		t.Fatalf("FromRegistry() error = %q, want reserved keyword mention", err)
	}
}

func TestFromRegistryRejectsNonASCIIFieldNames(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context: map[string]registry.Field{
			"café": {Name: "café", Type: registry.FieldTypeString},
		},
		Events: []registry.Event{{Name: "test", Version: 1, Properties: map[string]registry.Field{}}},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"café": {StableID: "café", ProtoNumber: 1},
		},
		Events: map[string]LockedEvent{
			"test@1": {Properties: map[string]LockedField{}},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want ASCII validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "ascii") {
		t.Fatalf("FromRegistry() error = %q, want ASCII mention", err)
	}
}

func TestFromRegistryRejectsNonASCIINamespace(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acmé.storefront",
		Context:   map[string]registry.Field{},
		Events:    []registry.Event{{Name: "test", Version: 1, Properties: map[string]registry.Field{}}},
	}
	lock := Lock{
		Version: 1,
		Events: map[string]LockedEvent{
			"test@1": {Properties: map[string]LockedField{}},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want namespace validation error")
	}
}

func TestFromRegistryRejectsMessageNameCollisions(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context:   map[string]registry.Field{},
		Events: []registry.Event{
			{Name: "a.b_c", Version: 1, Properties: map[string]registry.Field{}},
			{Name: "a_b.c", Version: 1, Properties: map[string]registry.Field{}},
		},
	}
	lock := Lock{
		Version: 1,
		Events: map[string]LockedEvent{
			"a.b_c@1": {Properties: map[string]LockedField{}},
			"a_b.c@1": {Properties: map[string]LockedField{}},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want message name collision error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "collision") {
		t.Fatalf("FromRegistry() error = %q, want collision mention", err)
	}
	if !strings.Contains(err.Error(), "a.b_c") || !strings.Contains(err.Error(), "a_b.c") {
		t.Fatalf("FromRegistry() error = %q, want both event names mentioned", err)
	}
}

func TestFromRegistryRejectsMixedVersions(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context:   map[string]registry.Field{},
		Events: []registry.Event{
			{Name: "test", Version: 1, Properties: map[string]registry.Field{}},
			{Name: "test", Version: 2, Properties: map[string]registry.Field{}},
		},
	}
	lock := Lock{
		Version: 1,
		Events: map[string]LockedEvent{
			"test@1": {Properties: map[string]LockedField{}},
			"test@2": {Properties: map[string]LockedField{}},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want mixed version error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "version") {
		t.Fatalf("FromRegistry() error = %q, want version mention", err)
	}
}

func TestFromRegistryRejectsNoEvents(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme.storefront",
		Context:   map[string]registry.Field{},
		Events:    []registry.Event{},
	}
	lock := Lock{Version: 1}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want no events error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "event") {
		t.Fatalf("FromRegistry() error = %q, want event mention", err)
	}
}

func TestFromRegistryRejectsEmptyEventName(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme",
		Context:   map[string]registry.Field{},
		Events: []registry.Event{
			{
				Name:       "",
				Version:    1,
				Properties: map[string]registry.Field{},
			},
		},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{},
		Events: map[string]LockedEvent{
			"@1": {
				Properties: map[string]LockedField{},
			},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want error for empty event name")
	}
	if !strings.Contains(err.Error(), "event name") {
		t.Fatalf("FromRegistry() error = %q, want mention of event name", err)
	}
}

func TestFromRegistryRejectsUnrenderableEventName(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme",
		Context:   map[string]registry.Field{},
		Events: []registry.Event{
			{
				Name:       "---",
				Version:    1,
				Properties: map[string]registry.Field{},
			},
		},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{},
		Events: map[string]LockedEvent{
			"---@1": {
				Properties: map[string]LockedField{},
			},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want error for unrenderable event name")
	}
	if !strings.Contains(err.Error(), "event name") && !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("FromRegistry() error = %q, want mention of event name or invalid", err)
	}
}

func TestFromRegistryRejectsProtobufScalarKeywordAsFieldName(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme",
		Context: map[string]registry.Field{
			"string": {
				Name: "string",
				Type: registry.FieldTypeString,
			},
		},
		Events: []registry.Event{
			{
				Name:    "test.event",
				Version: 1,
				Properties: map[string]registry.Field{
					"bool": {
						Name: "bool",
						Type: registry.FieldTypeString,
					},
				},
			},
		},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"string": {StableID: "string", ProtoNumber: 1},
		},
		Events: map[string]LockedEvent{
			"test.event@1": {
				Properties: map[string]LockedField{
					"bool": {StableID: "bool", ProtoNumber: 1},
				},
			},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want error for scalar keyword field name")
	}
	if !strings.Contains(err.Error(), "reserved") && !strings.Contains(err.Error(), "keyword") {
		t.Fatalf("FromRegistry() error = %q, want mention of reserved/keyword", err)
	}
}

func TestFromRegistryRejectsContextEnumTypeNameCollision(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme",
		Context: map[string]registry.Field{
			"foo_bar": {
				Name:   "foo_bar",
				Type:   registry.FieldTypeEnum,
				Values: []string{"a", "b"},
			},
			"foo__bar": {
				Name:   "foo__bar",
				Type:   registry.FieldTypeEnum,
				Values: []string{"x", "y"},
			},
		},
		Events: []registry.Event{
			{
				Name:       "test.event",
				Version:    1,
				Properties: map[string]registry.Field{},
			},
		},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"foo_bar":  {StableID: "foo_bar", ProtoNumber: 1},
			"foo__bar": {StableID: "foo__bar", ProtoNumber: 2},
		},
		Events: map[string]LockedEvent{
			"test.event@1": {
				Properties: map[string]LockedField{},
			},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want enum type collision error")
	}
	if !strings.Contains(err.Error(), "collision") || !strings.Contains(err.Error(), "FooBar") {
		t.Fatalf("FromRegistry() error = %q, want collision mentioning FooBar", err)
	}
}

func TestFromRegistryRejectsPropertiesEnumTypeNameCollision(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme",
		Context:   map[string]registry.Field{},
		Events: []registry.Event{
			{
				Name:    "test.event",
				Version: 1,
				Properties: map[string]registry.Field{
					"payment_method": {
						Name:   "payment_method",
						Type:   registry.FieldTypeEnum,
						Values: []string{"card", "cash"},
					},
					"payment__method": {
						Name:   "payment__method",
						Type:   registry.FieldTypeEnum,
						Values: []string{"debit", "credit"},
					},
				},
			},
		},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{},
		Events: map[string]LockedEvent{
			"test.event@1": {
				Properties: map[string]LockedField{
					"payment_method":  {StableID: "payment_method", ProtoNumber: 1},
					"payment__method": {StableID: "payment__method", ProtoNumber: 2},
				},
			},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want enum type collision error")
	}
	if !strings.Contains(err.Error(), "collision") || !strings.Contains(err.Error(), "PaymentMethod") {
		t.Fatalf("FromRegistry() error = %q, want collision mentioning PaymentMethod", err)
	}
}

func TestFromRegistryRejectsLeadingUnderscoreInFieldName(t *testing.T) {
	reg := registry.Registry{
		Namespace: "com.acme",
		Context: map[string]registry.Field{
			"_tenant_id": {
				Name: "_tenant_id",
				Type: registry.FieldTypeString,
			},
		},
		Events: []registry.Event{
			{
				Name:       "test.event",
				Version:    1,
				Properties: map[string]registry.Field{},
			},
		},
	}
	lock := Lock{
		Version: 1,
		Context: map[string]LockedField{
			"_tenant_id": {StableID: "_tenant_id", ProtoNumber: 1},
		},
		Events: map[string]LockedEvent{
			"test.event@1": {
				Properties: map[string]LockedField{},
			},
		},
	}

	_, err := FromRegistry(reg, lock)
	if err == nil {
		t.Fatalf("FromRegistry() error = nil, want error for leading underscore")
	}
	if !strings.Contains(err.Error(), "start") || !strings.Contains(err.Error(), "letter") {
		t.Fatalf("FromRegistry() error = %q, want mention of start with letter", err)
	}
}
