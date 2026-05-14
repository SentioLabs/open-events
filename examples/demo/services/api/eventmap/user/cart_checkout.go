package user

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

// CartCheckoutRequest is the JSON body for POST /v1/events/user/cart/checkout.
// Required primitive fields use pointer types so the validator can distinguish
// "field omitted from JSON" from "field set to its zero value".
type CartCheckoutRequest struct {
	Context       UserContext `json:"context"`
	CartID        *string     `json:"cart_id"`        // required
	ItemCount     *int64      `json:"item_count"`     // required
	SubtotalCents *int64      `json:"subtotal_cents"` // required
	Currency      string      `json:"currency"`       // "USD"|"EUR"|"GBP"
}

var checkoutCurrencyByName = map[string]userpb.UserCartCheckoutV1Properties_Currency{
	"USD": userpb.UserCartCheckoutV1Properties_CURRENCY_USD,
	"EUR": userpb.UserCartCheckoutV1Properties_CURRENCY_EUR,
	"GBP": userpb.UserCartCheckoutV1Properties_CURRENCY_GBP,
}

// Validate returns field-level errors for the request, empty on success.
func (r CartCheckoutRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.CartID == nil {
		errs = append(errs, eventmap.FieldError{Field: "cart_id", Message: "required"})
	}
	if r.ItemCount == nil {
		errs = append(errs, eventmap.FieldError{Field: "item_count", Message: "required"})
	}
	if r.SubtotalCents == nil {
		errs = append(errs, eventmap.FieldError{Field: "subtotal_cents", Message: "required"})
	}
	if _, ok := checkoutCurrencyByName[r.Currency]; !ok {
		errs = append(errs, eventmap.FieldError{Field: "currency", Message: "must be one of USD|EUR|GBP"})
	}
	return errs
}

// ToProto builds a UserCartCheckoutV1 protobuf with a fresh envelope.
func (r CartCheckoutRequest) ToProto() eventmap.EnvelopeMessage {
	return &userpb.UserCartCheckoutV1{
		EventName:    CartCheckoutV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &userpb.UserCartCheckoutV1Properties{
			CartId:        proto.String(*r.CartID),
			ItemCount:     proto.Int64(*r.ItemCount),
			SubtotalCents: proto.Int64(*r.SubtotalCents),
			Currency:      checkoutCurrencyByName[r.Currency].Enum(),
		},
	}
}
