package user

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

// AuthLoginRequest is the JSON body for POST /v1/events/user/auth/login.
type AuthLoginRequest struct {
	Context UserContext `json:"context"`
	Method  string      `json:"method"`  // "email"|"google"|"apple"
	Success *bool       `json:"success"` // required; pointer distinguishes false from omitted
}

var loginMethodByName = map[string]userpb.UserAuthLoginV1Properties_Method{
	"email":  userpb.UserAuthLoginV1Properties_METHOD_EMAIL,
	"google": userpb.UserAuthLoginV1Properties_METHOD_GOOGLE,
	"apple":  userpb.UserAuthLoginV1Properties_METHOD_APPLE,
}

// Validate returns field-level errors for the request, empty on success.
func (r AuthLoginRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if _, ok := loginMethodByName[r.Method]; !ok {
		errs = append(errs, eventmap.FieldError{Field: "method", Message: "must be one of email|google|apple"})
	}
	if r.Success == nil {
		errs = append(errs, eventmap.FieldError{Field: "success", Message: "required"})
	}
	return errs
}

// ToProto builds a UserAuthLoginV1 protobuf with a fresh envelope.
func (r AuthLoginRequest) ToProto() eventmap.EnvelopeMessage {
	return &userpb.UserAuthLoginV1{
		EventName:    AuthLoginV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &userpb.UserAuthLoginV1Properties{
			Method:  loginMethodByName[r.Method].Enum(),
			Success: proto.Bool(*r.Success),
		},
	}
}
