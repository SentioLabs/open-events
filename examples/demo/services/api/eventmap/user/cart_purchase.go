package user

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

// CartPurchaseRequest is the JSON body for POST /v1/events/user/cart/purchase.
type CartPurchaseRequest struct {
	Context       UserContext `json:"context"`
	CartID        string      `json:"cart_id"`
	OrderID       string      `json:"order_id"`
	TotalCents    int64       `json:"total_cents"`
	PaymentMethod string      `json:"payment_method"` // "card"|"apple_pay"|"google_pay"
	CouponCode    string      `json:"coupon_code,omitempty"`
}

var purchasePaymentMethodByName = map[string]userpb.UserCartPurchaseV1Properties_PaymentMethod{
	"card":       userpb.UserCartPurchaseV1Properties_PAYMENT_METHOD_CARD,
	"apple_pay":  userpb.UserCartPurchaseV1Properties_PAYMENT_METHOD_APPLE_PAY,
	"google_pay": userpb.UserCartPurchaseV1Properties_PAYMENT_METHOD_GOOGLE_PAY,
}

// Validate returns field-level errors for the request, empty on success.
func (r CartPurchaseRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.CartID == "" {
		errs = append(errs, eventmap.FieldError{Field: "cart_id", Message: "required"})
	}
	if r.OrderID == "" {
		errs = append(errs, eventmap.FieldError{Field: "order_id", Message: "required"})
	}
	if _, ok := purchasePaymentMethodByName[r.PaymentMethod]; !ok {
		errs = append(errs, eventmap.FieldError{Field: "payment_method", Message: "must be one of card|apple_pay|google_pay"})
	}
	return errs
}

// ToProto builds a UserCartPurchaseV1 protobuf with a fresh envelope.
func (r CartPurchaseRequest) ToProto() eventmap.EnvelopeMessage {
	props := &userpb.UserCartPurchaseV1Properties{
		CartId:        proto.String(r.CartID),
		OrderId:       proto.String(r.OrderID),
		TotalCents:    proto.Int64(r.TotalCents),
		PaymentMethod: purchasePaymentMethodByName[r.PaymentMethod].Enum(),
	}
	if r.CouponCode != "" {
		props.CouponCode = proto.String(r.CouponCode)
	}
	return &userpb.UserCartPurchaseV1{
		EventName:    CartPurchaseV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(clientName), Version: proto.String(clientVersion)},
		Context:      contextToProto(r.Context),
		Properties:   props,
	}
}
