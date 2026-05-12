package schemair

import (
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
)

func TestProtoPackageAndFilePath(t *testing.T) {
	pkg, err := ProtoPackage("com.acme.storefront", 1)
	if err != nil {
		t.Fatalf("ProtoPackage() error = %v, want nil", err)
	}
	if pkg != "com.acme.storefront.v1" {
		t.Fatalf("ProtoPackage() = %q, want %q", pkg, "com.acme.storefront.v1")
	}

	path, err := ProtoFilePath("com.acme.storefront", 1)
	if err != nil {
		t.Fatalf("ProtoFilePath() error = %v, want nil", err)
	}
	if path != "com/acme/storefront/v1/events.proto" {
		t.Fatalf("ProtoFilePath() = %q, want %q", path, "com/acme/storefront/v1/events.proto")
	}
}

func TestProtoPackageRejectsInvalidNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      string
	}{
		{name: "empty", namespace: "", want: "must not be empty"},
		{name: "starts with digit", namespace: "1com.acme", want: "starts with a digit"},
		{name: "invalid character", namespace: "com.acme-storefront", want: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ProtoPackage(tt.namespace, 1)
			if err == nil {
				t.Fatalf("ProtoPackage() error = nil, want non-nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("ProtoPackage() error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestEventAndEnumNameHelpers(t *testing.T) {
	event := registry.Event{Name: "checkout.completed", Version: 1}
	if got := EventMessageName(event); got != "CheckoutCompletedV1" {
		t.Fatalf("EventMessageName() = %q, want %q", got, "CheckoutCompletedV1")
	}
	if got := PropertiesMessageName(event); got != "CheckoutCompletedV1Properties" {
		t.Fatalf("PropertiesMessageName() = %q, want %q", got, "CheckoutCompletedV1Properties")
	}
	if got := EnumTypeName("payment_method"); got != "PaymentMethod" {
		t.Fatalf("EnumTypeName() = %q, want %q", got, "PaymentMethod")
	}
}

func TestEnumValueName(t *testing.T) {
	got, err := EnumValueName("PaymentMethod", "apple_pay")
	if err != nil {
		t.Fatalf("EnumValueName() error = %v, want nil", err)
	}
	if got != "PAYMENT_METHOD_APPLE_PAY" {
		t.Fatalf("EnumValueName() = %q, want %q", got, "PAYMENT_METHOD_APPLE_PAY")
	}

	got, err = EnumValueName("Currency", "USD")
	if err != nil {
		t.Fatalf("EnumValueName() error = %v, want nil", err)
	}
	if got != "CURRENCY_USD" {
		t.Fatalf("EnumValueName() = %q, want %q", got, "CURRENCY_USD")
	}
}

func TestEnumValueNameRejectsInvalidValue(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "must not be empty"},
		{name: "starts with digit", raw: "1day", want: "starts with a digit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EnumValueName("Currency", tt.raw)
			if err == nil {
				t.Fatalf("EnumValueName() error = nil, want non-nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("EnumValueName() error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestBuildEnumValuesRejectsCollisions(t *testing.T) {
	_, err := buildEnumValues("PaymentMethod", []string{"apple-pay", "apple_pay"}, "events.checkout.completed@1.properties.payment_method")
	if err == nil {
		t.Fatalf("buildEnumValues() error = nil, want collision error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "collide") {
		t.Fatalf("buildEnumValues() error = %q, want collision message", err)
	}
}
