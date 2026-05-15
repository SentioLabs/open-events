package user

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/common/v1"
	userpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/user/v1"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

var purchasePaymentMethodByName = map[string]userpb.UserCartPurchaseV1Properties_PaymentMethod{
	"card":       userpb.UserCartPurchaseV1Properties_PAYMENT_METHOD_CARD,
	"apple_pay":  userpb.UserCartPurchaseV1Properties_PAYMENT_METHOD_APPLE_PAY,
	"google_pay": userpb.UserCartPurchaseV1Properties_PAYMENT_METHOD_GOOGLE_PAY,
}

// ToProto builds a UserCartPurchaseV1 protobuf with a fresh envelope. Callers
// must invoke Validate() first.
func (r CartPurchaseRequest) ToProto() eventmap.EnvelopeMessage {
	props := &userpb.UserCartPurchaseV1Properties{
		CartId:        proto.String(*r.CartID),
		OrderId:       proto.String(*r.OrderID),
		TotalCents:    proto.Int64(*r.TotalCents),
		PaymentMethod: purchasePaymentMethodByName[*r.PaymentMethod].Enum(),
	}
	if r.CouponCode != "" {
		props.CouponCode = proto.String(r.CouponCode)
	}
	return &userpb.UserCartPurchaseV1{
		EventName:    CartPurchaseV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties:   props,
	}
}
