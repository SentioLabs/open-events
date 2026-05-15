package device

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/common/v1"
	devicepb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/device/v1"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

// ToProto builds a DeviceInfoCalibrationV1 protobuf with a fresh envelope.
// Callers must invoke Validate() first; Timestamp is parsed as RFC3339 per
// the registry's `type: timestamp` convention.
func (r InfoCalibrationRequest) ToProto() eventmap.EnvelopeMessage {
	// Validate already proved the timestamp parses; discard the error
	// deliberately.
	t, _ := time.Parse(time.RFC3339, *r.Timestamp)
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
			Timestamp:     timestamppb.New(t),
		},
	}
}
