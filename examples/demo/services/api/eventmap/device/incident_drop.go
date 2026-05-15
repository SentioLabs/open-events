package device

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

var axisTypeByName = map[string]devicepb.DeviceIncidentDropV1Properties_Axis{
	"x": devicepb.DeviceIncidentDropV1Properties_AXIS_X,
	"y": devicepb.DeviceIncidentDropV1Properties_AXIS_Y,
	"z": devicepb.DeviceIncidentDropV1Properties_AXIS_Z,
}

// ToProto builds a DeviceIncidentDropV1 protobuf with a fresh envelope. Callers
// must invoke Validate() first.
func (r IncidentDropRequest) ToProto() eventmap.EnvelopeMessage {
	return &devicepb.DeviceIncidentDropV1{
		EventName:    IncidentDropV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceIncidentDropV1Properties{
			PeakAccelerationG: proto.Float64(*r.PeakAccelerationG),
			Axis:              axisTypeByName[*r.Axis].Enum(),
			DurationMs:        proto.Int64(*r.DurationMs),
		},
	}
}
