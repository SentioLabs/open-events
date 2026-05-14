package device

import (
	"google.golang.org/protobuf/proto"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

// validateContext validates the device request context.
func validateContext(c DeviceContext) []eventmap.FieldError {
	var errs []eventmap.FieldError
	if c.TenantId == "" {
		errs = append(errs, eventmap.FieldError{Field: "context.tenant_id", Message: "required"})
	}
	if c.DeviceId == "" {
		errs = append(errs, eventmap.FieldError{Field: "context.device_id", Message: "required"})
	}
	return errs
}

// contextToProto converts a JSON-binding DeviceContext to the proto DeviceContext.
func contextToProto(c DeviceContext) *devicepb.DeviceContext {
	out := &devicepb.DeviceContext{
		TenantId: proto.String(c.TenantId),
		DeviceId: proto.String(c.DeviceId),
	}
	if c.SerialNumber != "" {
		out.SerialNumber = proto.String(c.SerialNumber)
	}
	if c.FirmwareVersion != "" {
		out.FirmwareVersion = proto.String(c.FirmwareVersion)
	}
	return out
}
