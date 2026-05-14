package device

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

// IncidentDropRequest is the JSON body for POST /v1/events/device/incident/drop.
type IncidentDropRequest struct {
	Context           DeviceContext `json:"context"`
	PeakAccelerationG float64       `json:"peak_acceleration_g"`
	Axis              string        `json:"axis"` // "x"|"y"|"z"
	DurationMs        int64         `json:"duration_ms"`
}

var axisTypeByName = map[string]devicepb.DeviceIncidentDropV1Properties_Axis{
	"x": devicepb.DeviceIncidentDropV1Properties_AXIS_X,
	"y": devicepb.DeviceIncidentDropV1Properties_AXIS_Y,
	"z": devicepb.DeviceIncidentDropV1Properties_AXIS_Z,
}

// Validate returns field-level errors for the request, empty on success.
func (r IncidentDropRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if _, ok := axisTypeByName[r.Axis]; !ok {
		errs = append(errs, eventmap.FieldError{Field: "axis", Message: "must be one of x|y|z"})
	}
	return errs
}

// ToProto builds a DeviceIncidentDropV1 protobuf with a fresh envelope.
func (r IncidentDropRequest) ToProto() eventmap.EnvelopeMessage {
	return &devicepb.DeviceIncidentDropV1{
		EventName:    IncidentDropV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(clientName), Version: proto.String(clientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceIncidentDropV1Properties{
			PeakAccelerationG: proto.Float64(r.PeakAccelerationG),
			Axis:              axisTypeByName[r.Axis].Enum(),
			DurationMs:        proto.Int64(r.DurationMs),
		},
	}
}
