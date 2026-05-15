package user

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

var loginMethodByName = map[string]userpb.UserAuthLoginV1Properties_Method{
	"email":  userpb.UserAuthLoginV1Properties_METHOD_EMAIL,
	"google": userpb.UserAuthLoginV1Properties_METHOD_GOOGLE,
	"apple":  userpb.UserAuthLoginV1Properties_METHOD_APPLE,
}

// ToProto builds a UserAuthLoginV1 protobuf with a fresh envelope. Callers
// must invoke Validate() first.
func (r AuthLoginRequest) ToProto() eventmap.EnvelopeMessage {
	return &userpb.UserAuthLoginV1{
		EventName:    AuthLoginV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &userpb.UserAuthLoginV1Properties{
			Method:  loginMethodByName[*r.Method].Enum(),
			Success: proto.Bool(*r.Success),
		},
	}
}
