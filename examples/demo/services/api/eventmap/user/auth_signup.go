package user

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

// AuthSignupRequest is the JSON body for POST /v1/events/user/auth/signup.
type AuthSignupRequest struct {
	Context UserContext `json:"context"`
	Method  string      `json:"method"` // "email"|"google"|"apple"
	Plan    string      `json:"plan,omitempty"`
}

var signupMethodByName = map[string]userpb.UserAuthSignupV1Properties_Method{
	"email":  userpb.UserAuthSignupV1Properties_METHOD_EMAIL,
	"google": userpb.UserAuthSignupV1Properties_METHOD_GOOGLE,
	"apple":  userpb.UserAuthSignupV1Properties_METHOD_APPLE,
}

// Validate returns field-level errors for the request, empty on success.
func (r AuthSignupRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if _, ok := signupMethodByName[r.Method]; !ok {
		errs = append(errs, eventmap.FieldError{Field: "method", Message: "must be one of email|google|apple"})
	}
	return errs
}

// ToProto builds a UserAuthSignupV1 protobuf with a fresh envelope.
func (r AuthSignupRequest) ToProto() eventmap.EnvelopeMessage {
	props := &userpb.UserAuthSignupV1Properties{
		Method: signupMethodByName[r.Method].Enum(),
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
