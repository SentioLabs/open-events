package user

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

// CartItemAddedRequest is the JSON body for POST /v1/events/user/cart/item_added.
type CartItemAddedRequest struct {
	Context  UserContext `json:"context"`
	CartID   string      `json:"cart_id"`
	SKU      string      `json:"sku"`
	Quantity int64       `json:"quantity"`
}

// Validate returns field-level errors for the request, empty on success.
func (r CartItemAddedRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.CartID == "" {
		errs = append(errs, eventmap.FieldError{Field: "cart_id", Message: "required"})
	}
	if r.SKU == "" {
		errs = append(errs, eventmap.FieldError{Field: "sku", Message: "required"})
	}
	return errs
}

// ToProto builds a UserCartItemAddedV1 protobuf with a fresh envelope.
func (r CartItemAddedRequest) ToProto() eventmap.EnvelopeMessage {
	return &userpb.UserCartItemAddedV1{
		EventName:    CartItemAddedV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &userpb.UserCartItemAddedV1Properties{
			CartId:   proto.String(r.CartID),
			Sku:      proto.String(r.SKU),
			Quantity: proto.Int64(r.Quantity),
		},
	}
}
