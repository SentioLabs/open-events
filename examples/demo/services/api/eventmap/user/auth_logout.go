package user

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/common/v1"
	userpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/user/v1"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

// ToProto builds a UserAuthLogoutV1 protobuf with a fresh envelope. Callers
// must invoke Validate() first.
func (r AuthLogoutRequest) ToProto() eventmap.EnvelopeMessage {
	return &userpb.UserAuthLogoutV1{
		EventName:    AuthLogoutV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &userpb.UserAuthLogoutV1Properties{
			DurationSeconds: proto.Int64(*r.DurationSeconds),
		},
	}
}
