package user

import (
	"google.golang.org/protobuf/proto"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
)

const (
	clientName    = "demo-api"
	clientVersion = "0.1.0"
)

// platformByName maps JSON platform strings to proto enum values.
var platformByName = map[string]userpb.UserContext_Platform{
	"ios":     userpb.UserContext_PLATFORM_IOS,
	"android": userpb.UserContext_PLATFORM_ANDROID,
	"web":     userpb.UserContext_PLATFORM_WEB,
}

// validateContext validates the request context and returns field-level errors.
func validateContext(c UserContext) []eventmap.FieldError {
	var errs []eventmap.FieldError
	if c.TenantId == "" {
		errs = append(errs, eventmap.FieldError{Field: "context.tenant_id", Message: "required"})
	}
	if _, ok := platformByName[c.Platform]; !ok {
		errs = append(errs, eventmap.FieldError{Field: "context.platform", Message: "must be one of ios|android|web"})
	}
	return errs
}

// contextToProto converts a JSON-binding UserContext to the proto UserContext.
func contextToProto(c UserContext) *userpb.UserContext {
	out := &userpb.UserContext{
		Platform: platformByName[c.Platform].Enum(),
		TenantId: proto.String(c.TenantId),
	}
	if c.UserId != "" {
		out.UserId = proto.String(c.UserId)
	}
	if c.SessionId != "" {
		out.SessionId = proto.String(c.SessionId)
	}
	return out
}
