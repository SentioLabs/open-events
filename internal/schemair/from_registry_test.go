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

	_, err := FromRegistry(reg, Lock{})
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
