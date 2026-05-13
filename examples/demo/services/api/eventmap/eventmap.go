package eventmap

import (
	"slices"
	"time"

	eventspb "github.com/acme/storefront/events"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Context is the wire shape clients post under the "context" key.
type Context struct {
	TenantID  string `json:"tenant_id"`
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Platform  string `json:"platform"` // "WEB" | "IOS" | "ANDROID" | "BACKEND"
}

var platformByName = map[string]eventspb.Context_Platform{
	"WEB":     eventspb.Context_PLATFORM_WEB,
	"IOS":     eventspb.Context_PLATFORM_IOS,
	"ANDROID": eventspb.Context_PLATFORM_ANDROID,
	"BACKEND": eventspb.Context_PLATFORM_BACKEND,
}

func (c Context) validate() []FieldError {
	var errs []FieldError
	if c.TenantID == "" {
		errs = append(errs, FieldError{Field: "context.tenant_id", Message: "required"})
	}
	if _, ok := platformByName[c.Platform]; !ok {
		errs = append(errs, FieldError{Field: "context.platform", Message: "must be one of WEB|IOS|ANDROID|BACKEND"})
	}
	return errs
}

func (c Context) toProto() *eventspb.Context {
	out := &eventspb.Context{
		Platform: platformByName[c.Platform].Enum(),
		TenantId: proto.String(c.TenantID),
	}
	if c.UserID != "" {
		out.UserId = proto.String(c.UserID)
	}
	if c.SessionID != "" {
		out.SessionId = proto.String(c.SessionID)
	}
	return out
}

const (
	clientName    = "demo-api"
	clientVersion = "0.1.0"
)

func newClient() *eventspb.Client {
	return &eventspb.Client{
		Name:    proto.String(clientName),
		Version: proto.String(clientVersion),
	}
}

func newEventID() string { return uuid.NewString() }

func newTimestamp() *timestamppb.Timestamp { return timestamppb.New(time.Now().UTC()) }

// --- CheckoutStarted ---

// CheckoutStartedRequest is the JSON body for POST /v1/events/checkout-started.
type CheckoutStartedRequest struct {
	Context       Context `json:"context"`
	CartID        string  `json:"cart_id"`
	ItemCount     int64   `json:"item_count"`
	SubtotalCents int64   `json:"subtotal_cents"`
	Currency      string  `json:"currency"` // "USD"|"EUR"|"GBP"
}

var currencyByName = map[string]eventspb.CheckoutStartedV1Properties_Currency{
	"USD": eventspb.CheckoutStartedV1Properties_CURRENCY_USD,
	"EUR": eventspb.CheckoutStartedV1Properties_CURRENCY_EUR,
	"GBP": eventspb.CheckoutStartedV1Properties_CURRENCY_GBP,
}

// Validate returns field-level errors for the request, empty on success.
func (r CheckoutStartedRequest) Validate() []FieldError {
	errs := r.Context.validate()
	if r.CartID == "" {
		errs = append(errs, FieldError{Field: "cart_id", Message: "required"})
	}
	if _, ok := currencyByName[r.Currency]; !ok {
		errs = append(errs, FieldError{Field: "currency", Message: "must be one of USD|EUR|GBP"})
	}
	return errs
}

// ToProto builds a CheckoutStartedV1 protobuf with a fresh envelope.
func (r CheckoutStartedRequest) ToProto() *eventspb.CheckoutStartedV1 {
	return &eventspb.CheckoutStartedV1{
		EventName:    CheckoutStartedV1,
		EventVersion: 1,
		EventId:      newEventID(),
		EventTs:      newTimestamp(),
		Client:       newClient(),
		Context:      r.Context.toProto(),
		Properties: &eventspb.CheckoutStartedV1Properties{
			CartId:        proto.String(r.CartID),
			ItemCount:     proto.Int64(r.ItemCount),
			SubtotalCents: proto.Int64(r.SubtotalCents),
			Currency:      currencyByName[r.Currency].Enum(),
		},
	}
}

// --- CheckoutCompleted ---

// CheckoutCompletedRequest is the JSON body for POST /v1/events/checkout-completed.
type CheckoutCompletedRequest struct {
	Context       Context `json:"context"`
	CartID        string  `json:"cart_id"`
	OrderID       string  `json:"order_id"`
	TotalCents    int64   `json:"total_cents"`
	PaymentMethod string  `json:"payment_method"` // "CARD"|"APPLE_PAY"|"GOOGLE_PAY"
	CouponCode    string  `json:"coupon_code,omitempty"`
}

var paymentMethodByName = map[string]eventspb.CheckoutCompletedV1Properties_PaymentMethod{
	"CARD":       eventspb.CheckoutCompletedV1Properties_PAYMENT_METHOD_CARD,
	"APPLE_PAY":  eventspb.CheckoutCompletedV1Properties_PAYMENT_METHOD_APPLE_PAY,
	"GOOGLE_PAY": eventspb.CheckoutCompletedV1Properties_PAYMENT_METHOD_GOOGLE_PAY,
}

// Validate returns field-level errors for the request, empty on success.
func (r CheckoutCompletedRequest) Validate() []FieldError {
	errs := r.Context.validate()
	if r.CartID == "" {
		errs = append(errs, FieldError{Field: "cart_id", Message: "required"})
	}
	if r.OrderID == "" {
		errs = append(errs, FieldError{Field: "order_id", Message: "required"})
	}
	if _, ok := paymentMethodByName[r.PaymentMethod]; !ok {
		errs = append(errs, FieldError{Field: "payment_method", Message: "must be one of CARD|APPLE_PAY|GOOGLE_PAY"})
	}
	return errs
}

// ToProto builds a CheckoutCompletedV1 protobuf with a fresh envelope.
func (r CheckoutCompletedRequest) ToProto() *eventspb.CheckoutCompletedV1 {
	props := &eventspb.CheckoutCompletedV1Properties{
		CartId:        proto.String(r.CartID),
		OrderId:       proto.String(r.OrderID),
		TotalCents:    proto.Int64(r.TotalCents),
		PaymentMethod: paymentMethodByName[r.PaymentMethod].Enum(),
	}
	if r.CouponCode != "" {
		props.CouponCode = proto.String(r.CouponCode)
	}
	return &eventspb.CheckoutCompletedV1{
		EventName:    CheckoutCompletedV1,
		EventVersion: 1,
		EventId:      newEventID(),
		EventTs:      newTimestamp(),
		Client:       newClient(),
		Context:      r.Context.toProto(),
		Properties:   props,
	}
}

// --- SearchPerformed ---

// SearchPerformedRequest is the JSON body for POST /v1/events/search-performed.
type SearchPerformedRequest struct {
	Context     Context  `json:"context"`
	Query       string   `json:"query"`
	ResultCount int64    `json:"result_count"`
	Filters     []string `json:"filters,omitempty"`
}

// Validate returns field-level errors for the request, empty on success.
func (r SearchPerformedRequest) Validate() []FieldError {
	errs := r.Context.validate()
	if r.Query == "" {
		errs = append(errs, FieldError{Field: "query", Message: "required"})
	}
	return errs
}

// ToProto builds a SearchPerformedV1 protobuf with a fresh envelope.
func (r SearchPerformedRequest) ToProto() *eventspb.SearchPerformedV1 {
	props := &eventspb.SearchPerformedV1Properties{
		Query:       proto.String(r.Query),
		ResultCount: proto.Int64(r.ResultCount),
	}
	if len(r.Filters) > 0 {
		props.Filters = slices.Clone(r.Filters)
	}
	return &eventspb.SearchPerformedV1{
		EventName:    SearchPerformedV1,
		EventVersion: 1,
		EventId:      newEventID(),
		EventTs:      newTimestamp(),
		Client:       newClient(),
		Context:      r.Context.toProto(),
		Properties:   props,
	}
}
