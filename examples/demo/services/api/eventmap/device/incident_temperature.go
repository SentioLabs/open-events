package device

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

// IncidentTemperatureRequest is the JSON body for POST /v1/events/device/incident/temperature.
type IncidentTemperatureRequest struct {
	Context    DeviceContext `json:"context"`
	DegreesC   *float64      `json:"degrees_c"`   // required; pointer distinguishes 0.0 from omitted
	ThresholdC *float64      `json:"threshold_c"` // required; pointer distinguishes 0.0 from omitted
	BreachType string        `json:"breach_type"` // "over"|"under"
}

var breachTypeByName = map[string]devicepb.DeviceIncidentTemperatureV1Properties_BreachType{
	"over":  devicepb.DeviceIncidentTemperatureV1Properties_BREACH_TYPE_OVER,
	"under": devicepb.DeviceIncidentTemperatureV1Properties_BREACH_TYPE_UNDER,
}

// Validate returns field-level errors for the request, empty on success.
func (r IncidentTemperatureRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.DegreesC == nil {
		errs = append(errs, eventmap.FieldError{Field: "degrees_c", Message: "required"})
	}
	if r.ThresholdC == nil {
		errs = append(errs, eventmap.FieldError{Field: "threshold_c", Message: "required"})
	}
	if _, ok := breachTypeByName[r.BreachType]; !ok {
		errs = append(errs, eventmap.FieldError{Field: "breach_type", Message: "must be one of over|under"})
	}
	return errs
}

// ToProto builds a DeviceIncidentTemperatureV1 protobuf with a fresh envelope.
func (r IncidentTemperatureRequest) ToProto() eventmap.EnvelopeMessage {
	return &devicepb.DeviceIncidentTemperatureV1{
		EventName:    IncidentTemperatureV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(clientName), Version: proto.String(clientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceIncidentTemperatureV1Properties{
			DegreesC:   proto.Float64(*r.DegreesC),
			ThresholdC: proto.Float64(*r.ThresholdC),
			BreachType: breachTypeByName[r.BreachType].Enum(),
		},
	}
}
