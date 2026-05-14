package device

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

// InfoCalibrationRequest is the JSON body for POST /v1/events/device/info/calibration.
type InfoCalibrationRequest struct {
	Context       DeviceContext `json:"context"`
	Concentration *float64      `json:"concentration"` // required; pointer distinguishes 0.0 from omitted
	Integral      *int64        `json:"integral"`      // required; pointer distinguishes 0 from omitted
	Timestamp     *int64        `json:"timestamp"`     // required unix epoch seconds; pointer distinguishes 0 from omitted
}

// Validate returns field-level errors for the request, empty on success.
func (r InfoCalibrationRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.Concentration == nil {
		errs = append(errs, eventmap.FieldError{Field: "concentration", Message: "required"})
	}
	if r.Integral == nil {
		errs = append(errs, eventmap.FieldError{Field: "integral", Message: "required"})
	}
	if r.Timestamp == nil {
		errs = append(errs, eventmap.FieldError{Field: "timestamp", Message: "required"})
	}
	return errs
}

// ToProto builds a DeviceInfoCalibrationV1 protobuf with a fresh envelope.
func (r InfoCalibrationRequest) ToProto() eventmap.EnvelopeMessage {
	return &devicepb.DeviceInfoCalibrationV1{
		EventName:    InfoCalibrationV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceInfoCalibrationV1Properties{
			Concentration: proto.Float64(*r.Concentration),
			Integral:      proto.Int64(*r.Integral),
			Timestamp:     timestamppb.New(time.Unix(*r.Timestamp, 0)),
		},
	}
}
