package user

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

// AuthLogoutRequest is the JSON body for POST /v1/events/user/auth/logout.
type AuthLogoutRequest struct {
	Context         UserContext `json:"context"`
	DurationSeconds *int64      `json:"duration_seconds"` // required; pointer distinguishes 0 from omitted
}

// Validate returns field-level errors for the request, empty on success.
func (r AuthLogoutRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.DurationSeconds == nil {
		errs = append(errs, eventmap.FieldError{Field: "duration_seconds", Message: "required"})
	}
	return errs
}

// ToProto builds a UserAuthLogoutV1 protobuf with a fresh envelope.
func (r AuthLogoutRequest) ToProto() eventmap.EnvelopeMessage {
	return &userpb.UserAuthLogoutV1{
		EventName:    AuthLogoutV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(clientName), Version: proto.String(clientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &userpb.UserAuthLogoutV1Properties{
			DurationSeconds: proto.Int64(*r.DurationSeconds),
		},
	}
}
