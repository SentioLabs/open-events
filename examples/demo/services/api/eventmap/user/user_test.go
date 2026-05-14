package user_test

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/user"
)

func int64p(v int64) *int64 { return &v }
func boolp(v bool) *bool    { return &v }

func validUserContext() user.UserContext {
	return user.UserContext{
		TenantId:  "tenant-1",
		UserId:    "user-1",
		SessionId: "session-1",
		Platform:  "web",
	}
}

func containsField(errs []eventmap.FieldError, field string) bool {
	return slices.ContainsFunc(errs, func(e eventmap.FieldError) bool {
		return e.Field == field
	})
}

// --- AuthSignup ---

func TestAuthSignup_Validate_RejectsMissingTenantID(t *testing.T) {
	req := user.AuthSignupRequest{
		Context: user.UserContext{Platform: "web"},
		Method:  "email",
	}
	errs := req.Validate()
	if !containsField(errs, "context.tenant_id") {
		t.Fatalf("expected context.tenant_id error, got %+v", errs)
	}
}

func TestAuthSignup_Validate_RejectsBadMethod(t *testing.T) {
	req := user.AuthSignupRequest{
		Context: validUserContext(),
		Method:  "fax",
	}
	errs := req.Validate()
	if !containsField(errs, "method") {
		t.Fatalf("expected method error, got %+v", errs)
	}
}

func TestAuthSignup_Validate_AcceptsValid(t *testing.T) {
	req := user.AuthSignupRequest{
		Context: validUserContext(),
		Method:  "google",
	}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}

func TestAuthSignup_ToProto_FillsEnvelope(t *testing.T) {
	before := time.Now().UTC().Add(-time.Second)
	req := user.AuthSignupRequest{
		Context: validUserContext(),
		Method:  "apple",
		Plan:    "enterprise",
	}
	envelope := req.ToProto()

	if envelope.GetEventId() == "" {
		t.Error("EventId is empty")
	}
	if _, err := uuid.Parse(envelope.GetEventId()); err != nil {
		t.Errorf("EventId %q is not a valid uuid: %v", envelope.GetEventId(), err)
	}

	// Type-assert to access the concrete proto type for timestamp check.
	msg, ok := envelope.(*userpb.UserAuthSignupV1)
	if !ok {
		t.Fatal("expected *userpb.UserAuthSignupV1")
	}
	if msg.GetEventTs() == nil {
		t.Fatal("EventTs is nil")
	}
	ts := msg.GetEventTs().AsTime()
	after := time.Now().UTC().Add(time.Second)
	if ts.Before(before) || ts.After(after) {
		t.Errorf("EventTs %v outside range", ts)
	}
}

func TestAuthSignup_ToProto_RoundTrip(t *testing.T) {
	req := user.AuthSignupRequest{
		Context: validUserContext(),
		Method:  "email",
		Plan:    "starter",
	}
	env := req.ToProto()
	wire, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got userpb.UserAuthSignupV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetEventName() != user.AuthSignupV1 {
		t.Errorf("EventName: got %q want %q", got.GetEventName(), user.AuthSignupV1)
	}
	if got.GetContext().GetTenantId() != "tenant-1" {
		t.Errorf("TenantId: %q", got.GetContext().GetTenantId())
	}
	if got.GetProperties().GetPlan() != "starter" {
		t.Errorf("Plan: %q", got.GetProperties().GetPlan())
	}
	if got.GetProperties().GetMethod() != userpb.UserAuthSignupV1Properties_METHOD_EMAIL {
		t.Errorf("Method: %v", got.GetProperties().GetMethod())
	}
}

// --- AuthLogin required primitive fields ---

func TestAuthLogin_Validate_RejectsMissingSuccess(t *testing.T) {
	req := user.AuthLoginRequest{
		Context: validUserContext(),
		Method:  "email",
		// Success omitted (nil)
	}
	errs := req.Validate()
	if !containsField(errs, "success") {
		t.Fatalf("expected success error for missing bool, got %+v", errs)
	}
}

func TestAuthLogin_Validate_AcceptsSuccessFalse(t *testing.T) {
	f := false
	req := user.AuthLoginRequest{
		Context: validUserContext(),
		Method:  "email",
		Success: &f,
	}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors for explicit false, got %+v", errs)
	}
}

func TestAuthLogin_Validate_AcceptsSuccessTrue(t *testing.T) {
	tr := true
	req := user.AuthLoginRequest{
		Context: validUserContext(),
		Method:  "email",
		Success: &tr,
	}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors for explicit true, got %+v", errs)
	}
}

// --- AuthLogout required primitive fields ---

func TestAuthLogout_Validate_RejectsMissingDuration(t *testing.T) {
	req := user.AuthLogoutRequest{
		Context: validUserContext(),
		// DurationSeconds omitted (nil)
	}
	errs := req.Validate()
	if !containsField(errs, "duration_seconds") {
		t.Fatalf("expected duration_seconds error for missing int64, got %+v", errs)
	}
}

// --- CartCheckout ---

func TestCartCheckout_Validate_RejectsMissingCartID(t *testing.T) {
	req := user.CartCheckoutRequest{
		Context:       validUserContext(),
		ItemCount:     int64p(1),
		SubtotalCents: int64p(100),
		Currency:      "USD",
	}
	errs := req.Validate()
	if !containsField(errs, "cart_id") {
		t.Fatalf("expected cart_id error, got %+v", errs)
	}
}

func TestCartCheckout_Validate_RejectsBadCurrency(t *testing.T) {
	req := user.CartCheckoutRequest{
		Context:       validUserContext(),
		CartID:        "cart-1",
		ItemCount:     int64p(1),
		SubtotalCents: int64p(100),
		Currency:      "BTC",
	}
	errs := req.Validate()
	if !containsField(errs, "currency") {
		t.Fatalf("expected currency error, got %+v", errs)
	}
}

func TestCartCheckout_Validate_RejectsMissingItemCount(t *testing.T) {
	req := user.CartCheckoutRequest{
		Context:       validUserContext(),
		CartID:        "cart-1",
		SubtotalCents: int64p(100),
		Currency:      "USD",
	}
	errs := req.Validate()
	if !containsField(errs, "item_count") {
		t.Fatalf("expected item_count error, got %+v", errs)
	}
}

func TestCartCheckout_Validate_RejectsMissingSubtotalCents(t *testing.T) {
	req := user.CartCheckoutRequest{
		Context:   validUserContext(),
		CartID:    "cart-1",
		ItemCount: int64p(1),
		Currency:  "USD",
	}
	errs := req.Validate()
	if !containsField(errs, "subtotal_cents") {
		t.Fatalf("expected subtotal_cents error, got %+v", errs)
	}
}

func TestCartCheckout_ToProto_RoundTrip(t *testing.T) {
	req := user.CartCheckoutRequest{
		Context:       validUserContext(),
		CartID:        "cart-42",
		ItemCount:     int64p(7),
		SubtotalCents: int64p(1999),
		Currency:      "GBP",
	}
	env := req.ToProto()
	wire, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got userpb.UserCartCheckoutV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetEventName() != user.CartCheckoutV1 {
		t.Errorf("EventName: %q", got.GetEventName())
	}
	if got.GetProperties().GetCartId() != "cart-42" {
		t.Errorf("CartId: %q", got.GetProperties().GetCartId())
	}
	if got.GetProperties().GetItemCount() != 7 {
		t.Errorf("ItemCount: %d", got.GetProperties().GetItemCount())
	}
	if got.GetProperties().GetCurrency() != userpb.UserCartCheckoutV1Properties_CURRENCY_GBP {
		t.Errorf("Currency: %v", got.GetProperties().GetCurrency())
	}
}

// --- CartPurchase ---

func TestCartPurchase_Validate_RejectsMissingOrderID(t *testing.T) {
	req := user.CartPurchaseRequest{
		Context:       validUserContext(),
		CartID:        "cart-1",
		TotalCents:    100,
		PaymentMethod: "card",
	}
	errs := req.Validate()
	if !containsField(errs, "order_id") {
		t.Fatalf("expected order_id error, got %+v", errs)
	}
}

func TestCartPurchase_Validate_RejectsBadPaymentMethod(t *testing.T) {
	req := user.CartPurchaseRequest{
		Context:       validUserContext(),
		CartID:        "cart-1",
		OrderID:       "order-1",
		TotalCents:    100,
		PaymentMethod: "bitcoin",
	}
	errs := req.Validate()
	if !containsField(errs, "payment_method") {
		t.Fatalf("expected payment_method error, got %+v", errs)
	}
}

func TestCartPurchase_ToProto_RoundTrip(t *testing.T) {
	req := user.CartPurchaseRequest{
		Context:       validUserContext(),
		CartID:        "cart-9",
		OrderID:       "order-9",
		TotalCents:    9999,
		PaymentMethod: "apple_pay",
		CouponCode:    "WELCOME",
	}
	env := req.ToProto()
	wire, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got userpb.UserCartPurchaseV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetEventName() != user.CartPurchaseV1 {
		t.Errorf("EventName: %q", got.GetEventName())
	}
	if got.GetProperties().GetOrderId() != "order-9" {
		t.Errorf("OrderId: %q", got.GetProperties().GetOrderId())
	}
	if got.GetProperties().GetCouponCode() != "WELCOME" {
		t.Errorf("CouponCode: %q", got.GetProperties().GetCouponCode())
	}
	if got.GetProperties().GetPaymentMethod() != userpb.UserCartPurchaseV1Properties_PAYMENT_METHOD_APPLE_PAY {
		t.Errorf("PaymentMethod: %v", got.GetProperties().GetPaymentMethod())
	}
}

// --- CartItemAdded ---

func TestCartItemAdded_Validate_RejectsMissingSKU(t *testing.T) {
	req := user.CartItemAddedRequest{
		Context:  validUserContext(),
		CartID:   "cart-1",
		Quantity: 1,
	}
	errs := req.Validate()
	if !containsField(errs, "sku") {
		t.Fatalf("expected sku error, got %+v", errs)
	}
}

func TestCartItemAdded_ToProto_RoundTrip(t *testing.T) {
	req := user.CartItemAddedRequest{
		Context:  validUserContext(),
		CartID:   "cart-1",
		SKU:      "SKU-999",
		Quantity: 3,
	}
	env := req.ToProto()
	wire, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got userpb.UserCartItemAddedV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetEventName() != user.CartItemAddedV1 {
		t.Errorf("EventName: %q", got.GetEventName())
	}
	if got.GetProperties().GetSku() != "SKU-999" {
		t.Errorf("Sku: %q", got.GetProperties().GetSku())
	}
	if got.GetProperties().GetQuantity() != 3 {
		t.Errorf("Quantity: %d", got.GetProperties().GetQuantity())
	}
}
