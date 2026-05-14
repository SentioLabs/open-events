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
	Concentration float64       `json:"concentration"`
	Integral      int64         `json:"integral"`
	Timestamp     int64         `json:"timestamp"` // unix epoch seconds
}

// Validate returns field-level errors for the request, empty on success.
func (r InfoCalibrationRequest) Validate() []eventmap.FieldError {
	return validateContext(r.Context)
}

// ToProto builds a DeviceInfoCalibrationV1 protobuf with a fresh envelope.
func (r InfoCalibrationRequest) ToProto() eventmap.EnvelopeMessage {
	return &devicepb.DeviceInfoCalibrationV1{
		EventName:    InfoCalibrationV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(clientName), Version: proto.String(clientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceInfoCalibrationV1Properties{
			Concentration: proto.Float64(r.Concentration),
			Integral:      proto.Int64(r.Integral),
			Timestamp:     timestamppb.New(time.Unix(r.Timestamp, 0)),
		},
	}
}
