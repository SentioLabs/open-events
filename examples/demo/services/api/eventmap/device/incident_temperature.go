package device

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

var breachTypeByName = map[string]devicepb.DeviceIncidentTemperatureV1Properties_BreachType{
	"over":  devicepb.DeviceIncidentTemperatureV1Properties_BREACH_TYPE_OVER,
	"under": devicepb.DeviceIncidentTemperatureV1Properties_BREACH_TYPE_UNDER,
}

// ToProto builds a DeviceIncidentTemperatureV1 protobuf with a fresh envelope.
// Callers must invoke Validate() first.
func (r IncidentTemperatureRequest) ToProto() eventmap.EnvelopeMessage {
	return &devicepb.DeviceIncidentTemperatureV1{
		EventName:    IncidentTemperatureV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceIncidentTemperatureV1Properties{
			DegreesC:   proto.Float64(*r.DegreesC),
			ThresholdC: proto.Float64(*r.ThresholdC),
			BreachType: breachTypeByName[*r.BreachType].Enum(),
		},
	}
}
