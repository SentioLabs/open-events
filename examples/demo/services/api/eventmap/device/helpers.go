package device

import (
	"google.golang.org/protobuf/proto"

	devicepb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/device/v1"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

// validateContext validates the device request context.
func validateContext(c DeviceContext) []eventmap.FieldError {
	var errs []eventmap.FieldError
	if c.TenantID == "" {
		errs = append(errs, eventmap.FieldError{Field: "context.tenant_id", Message: "required"})
	}
	if c.DeviceID == "" {
		errs = append(errs, eventmap.FieldError{Field: "context.device_id", Message: "required"})
	}
	return errs
}

// contextToProto converts a JSON-binding DeviceContext to the proto DeviceContext.
func contextToProto(c DeviceContext) *devicepb.DeviceContext {
	out := &devicepb.DeviceContext{
		TenantId: proto.String(c.TenantID),
		DeviceId: proto.String(c.DeviceID),
	}
	if c.SerialNumber != "" {
		out.SerialNumber = proto.String(c.SerialNumber)
	}
	if c.FirmwareVersion != "" {
		out.FirmwareVersion = proto.String(c.FirmwareVersion)
	}
	return out
}
