package user

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

// signupMethodByName maps the JSON enum string to the proto enum. Validation
// of the registry value set is handled by the generated Validate method in
// auth_signup_request.go; this map exists so ToProto can convert.
var signupMethodByName = map[string]userpb.UserAuthSignupV1Properties_Method{
	"email":  userpb.UserAuthSignupV1Properties_METHOD_EMAIL,
	"google": userpb.UserAuthSignupV1Properties_METHOD_GOOGLE,
	"apple":  userpb.UserAuthSignupV1Properties_METHOD_APPLE,
}

// ToProto builds a UserAuthSignupV1 protobuf with a fresh envelope. Callers
// must invoke Validate() first; ToProto assumes required fields are set.
func (r AuthSignupRequest) ToProto() eventmap.EnvelopeMessage {
	props := &userpb.UserAuthSignupV1Properties{
		Method: signupMethodByName[*r.Method].Enum(),
	}
	if r.Plan != "" {
		props.Plan = proto.String(r.Plan)
	}
	return &userpb.UserAuthSignupV1{
		EventName:    AuthSignupV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties:   props,
	}
}
