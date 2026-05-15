package user

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

var checkoutCurrencyByName = map[string]userpb.UserCartCheckoutV1Properties_Currency{
	"USD": userpb.UserCartCheckoutV1Properties_CURRENCY_USD,
	"EUR": userpb.UserCartCheckoutV1Properties_CURRENCY_EUR,
	"GBP": userpb.UserCartCheckoutV1Properties_CURRENCY_GBP,
}

// ToProto builds a UserCartCheckoutV1 protobuf with a fresh envelope. Callers
// must invoke Validate() first.
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
			Currency:      checkoutCurrencyByName[*r.Currency].Enum(),
		},
	}
}
