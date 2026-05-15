package user

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/common/v1"
	userpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/user/v1"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

// ToProto builds a UserCartItemAddedV1 protobuf with a fresh envelope. Callers
// must invoke Validate() first.
func (r CartItemAddedRequest) ToProto() eventmap.EnvelopeMessage {
	return &userpb.UserCartItemAddedV1{
		EventName:    CartItemAddedV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &userpb.UserCartItemAddedV1Properties{
			CartId:   proto.String(*r.CartID),
			Sku:      proto.String(*r.SKU),
			Quantity: proto.Int64(*r.Quantity),
		},
	}
}
