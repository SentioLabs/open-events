package eventmap

import (
	"slices"
	"testing"
	"time"

	eventspb "github.com/acme/storefront/events"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func validContext() Context {
	return Context{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		SessionID: "session-1",
		Platform:  "WEB",
	}
}

// --- CheckoutStarted ---

func TestCheckoutStarted_Validate_RejectsMissingTenantID(t *testing.T) {
	req := CheckoutStartedRequest{
		Context:       Context{Platform: "WEB"},
		CartID:        "cart-1",
		ItemCount:     1,
		SubtotalCents: 100,
		Currency:      "USD",
	}
	errs := req.Validate()
	if !containsField(errs, "context.tenant_id") {
		t.Fatalf("expected context.tenant_id error, got %+v", errs)
	}
}

func TestCheckoutStarted_Validate_RejectsBadPlatform(t *testing.T) {
	req := CheckoutStartedRequest{
		Context:       Context{TenantID: "t", Platform: "MAINFRAME"},
		CartID:        "cart-1",
		ItemCount:     1,
		SubtotalCents: 100,
		Currency:      "USD",
	}
	errs := req.Validate()
	if !containsField(errs, "context.platform") {
		t.Fatalf("expected context.platform error, got %+v", errs)
	}
}

func TestCheckoutStarted_Validate_RejectsMissingCartID(t *testing.T) {
	req := CheckoutStartedRequest{
		Context:       validContext(),
		ItemCount:     1,
		SubtotalCents: 100,
		Currency:      "USD",
	}
	errs := req.Validate()
	if !containsField(errs, "cart_id") {
		t.Fatalf("expected cart_id error, got %+v", errs)
	}
}

func TestCheckoutStarted_Validate_RejectsBadCurrency(t *testing.T) {
	req := CheckoutStartedRequest{
		Context:       validContext(),
		CartID:        "cart-1",
		ItemCount:     1,
		SubtotalCents: 100,
		Currency:      "BTC",
	}
	errs := req.Validate()
	if !containsField(errs, "currency") {
		t.Fatalf("expected currency error, got %+v", errs)
	}
}

func TestCheckoutStarted_Validate_AcceptsValid(t *testing.T) {
	req := CheckoutStartedRequest{
		Context:       validContext(),
		CartID:        "cart-1",
		ItemCount:     2,
		SubtotalCents: 1234,
		Currency:      "EUR",
	}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}

func TestCheckoutStarted_ToProto_FillsEnvelope(t *testing.T) {
	before := time.Now().UTC().Add(-time.Second)
	req := CheckoutStartedRequest{
		Context:       validContext(),
		CartID:        "cart-1",
		ItemCount:     3,
		SubtotalCents: 4500,
		Currency:      "USD",
	}
	msg := req.ToProto()

	if msg.GetEventName() != CheckoutStartedV1 {
		t.Errorf("EventName: got %q want %q", msg.GetEventName(), CheckoutStartedV1)
	}
	if msg.GetEventVersion() != 1 {
		t.Errorf("EventVersion: got %d want 1", msg.GetEventVersion())
	}
	if _, err := uuid.Parse(msg.GetEventId()); err != nil {
		t.Errorf("EventId %q is not a valid uuid: %v", msg.GetEventId(), err)
	}
	if msg.GetEventTs() == nil {
		t.Fatal("EventTs is nil")
	}
	ts := msg.GetEventTs().AsTime()
	after := time.Now().UTC().Add(time.Second)
	if ts.Before(before) || ts.After(after) {
		t.Errorf("EventTs %v outside [%v, %v]", ts, before, after)
	}
	if msg.GetClient().GetName() != "demo-api" {
		t.Errorf("Client.Name: got %q want demo-api", msg.GetClient().GetName())
	}
	if msg.GetClient().GetVersion() == "" {
		t.Error("Client.Version is empty")
	}
}

func TestCheckoutStarted_ToProto_RoundTrip(t *testing.T) {
	req := CheckoutStartedRequest{
		Context:       validContext(),
		CartID:        "cart-42",
		ItemCount:     7,
		SubtotalCents: 1999,
		Currency:      "GBP",
	}
	msg := req.ToProto()
	wire, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got eventspb.CheckoutStartedV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetEventName() != CheckoutStartedV1 {
		t.Errorf("EventName: %q", got.GetEventName())
	}
	if got.GetEventId() != msg.GetEventId() {
		t.Errorf("EventId mismatch: got %q want %q", got.GetEventId(), msg.GetEventId())
	}
	if got.GetContext().GetTenantId() != "tenant-1" {
		t.Errorf("Context.TenantId: %q", got.GetContext().GetTenantId())
	}
	if got.GetContext().GetPlatform() != eventspb.Context_PLATFORM_WEB {
		t.Errorf("Context.Platform: %v", got.GetContext().GetPlatform())
	}
	if got.GetProperties().GetCartId() != "cart-42" {
		t.Errorf("CartId: %q", got.GetProperties().GetCartId())
	}
	if got.GetProperties().GetItemCount() != 7 {
		t.Errorf("ItemCount: %d", got.GetProperties().GetItemCount())
	}
	if got.GetProperties().GetSubtotalCents() != 1999 {
		t.Errorf("SubtotalCents: %d", got.GetProperties().GetSubtotalCents())
	}
	if got.GetProperties().GetCurrency() != eventspb.CheckoutStartedV1Properties_CURRENCY_GBP {
		t.Errorf("Currency: %v", got.GetProperties().GetCurrency())
	}
}

// --- CheckoutCompleted ---

func TestCheckoutCompleted_Validate_RejectsMissingTenantID(t *testing.T) {
	req := CheckoutCompletedRequest{
		Context:       Context{Platform: "WEB"},
		CartID:        "cart-1",
		OrderID:       "order-1",
		TotalCents:    100,
		PaymentMethod: "CARD",
	}
	if !containsField(req.Validate(), "context.tenant_id") {
		t.Fatal("expected context.tenant_id error")
	}
}

func TestCheckoutCompleted_Validate_RejectsBadPaymentMethod(t *testing.T) {
	req := CheckoutCompletedRequest{
		Context:       validContext(),
		CartID:        "cart-1",
		OrderID:       "order-1",
		TotalCents:    100,
		PaymentMethod: "BITCOIN",
	}
	if !containsField(req.Validate(), "payment_method") {
		t.Fatalf("expected payment_method error, got %+v", req.Validate())
	}
}

func TestCheckoutCompleted_Validate_RejectsMissingOrderID(t *testing.T) {
	req := CheckoutCompletedRequest{
		Context:       validContext(),
		CartID:        "cart-1",
		TotalCents:    100,
		PaymentMethod: "CARD",
	}
	if !containsField(req.Validate(), "order_id") {
		t.Fatal("expected order_id error")
	}
}

func TestCheckoutCompleted_ToProto_RoundTrip(t *testing.T) {
	req := CheckoutCompletedRequest{
		Context:       validContext(),
		CartID:        "cart-9",
		OrderID:       "order-9",
		TotalCents:    9999,
		PaymentMethod: "APPLE_PAY",
		CouponCode:    "WELCOME",
	}
	msg := req.ToProto()
	wire, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got eventspb.CheckoutCompletedV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetEventName() != CheckoutCompletedV1 {
		t.Errorf("EventName: %q", got.GetEventName())
	}
	if got.GetProperties().GetOrderId() != "order-9" {
		t.Errorf("OrderId: %q", got.GetProperties().GetOrderId())
	}
	if got.GetProperties().GetCouponCode() != "WELCOME" {
		t.Errorf("CouponCode: %q", got.GetProperties().GetCouponCode())
	}
	if got.GetProperties().GetTotalCents() != 9999 {
		t.Errorf("TotalCents: %d", got.GetProperties().GetTotalCents())
	}
	if got.GetProperties().GetPaymentMethod() != eventspb.CheckoutCompletedV1Properties_PAYMENT_METHOD_APPLE_PAY {
		t.Errorf("PaymentMethod: %v", got.GetProperties().GetPaymentMethod())
	}
}

// --- SearchPerformed ---

func TestSearchPerformed_Validate_RejectsMissingTenantID(t *testing.T) {
	req := SearchPerformedRequest{
		Context:     Context{Platform: "WEB"},
		Query:       "shoes",
		ResultCount: 12,
	}
	if !containsField(req.Validate(), "context.tenant_id") {
		t.Fatal("expected context.tenant_id error")
	}
}

func TestSearchPerformed_Validate_RejectsMissingQuery(t *testing.T) {
	req := SearchPerformedRequest{
		Context:     validContext(),
		ResultCount: 1,
	}
	if !containsField(req.Validate(), "query") {
		t.Fatal("expected query error")
	}
}

func TestSearchPerformed_ToProto_RoundTrip(t *testing.T) {
	req := SearchPerformedRequest{
		Context:     validContext(),
		Query:       "running shoes",
		ResultCount: 25,
		Filters:     []string{"brand:acme", "size:10"},
	}
	msg := req.ToProto()
	wire, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got eventspb.SearchPerformedV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetEventName() != SearchPerformedV1 {
		t.Errorf("EventName: %q", got.GetEventName())
	}
	if got.GetProperties().GetQuery() != "running shoes" {
		t.Errorf("Query: %q", got.GetProperties().GetQuery())
	}
	if got.GetProperties().GetResultCount() != 25 {
		t.Errorf("ResultCount: %d", got.GetProperties().GetResultCount())
	}
	if want := []string{"brand:acme", "size:10"}; !equalStrings(got.GetProperties().GetFilters(), want) {
		t.Errorf("Filters: got %+v want %+v", got.GetProperties().GetFilters(), want)
	}
}

// --- helpers ---

func containsField(errs []FieldError, field string) bool {
	return slices.ContainsFunc(errs, func(e FieldError) bool { return e.Field == field })
}

func equalStrings(a, b []string) bool {
	return slices.Equal(a, b)
}
