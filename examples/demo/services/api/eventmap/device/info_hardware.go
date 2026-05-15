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

var sensorTypeByName = map[string]devicepb.DeviceInfoHardwareV1Properties_SensorType{
	"unspecified": devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_UNSPECIFIED,
	"co":          devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_CO,
	"alcohol":     devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_ALCOHOL,
	"oxygen":      devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_OXYGEN,
	"fuel_cell":   devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_FUEL_CELL,
}

// ToProto builds a DeviceInfoHardwareV1 protobuf with a fresh envelope.
// Callers must invoke Validate() first; ManufacturingTimestamp is parsed
// as RFC3339 (Validate already proved it parses).
func (r InfoHardwareRequest) ToProto() eventmap.EnvelopeMessage {
	t, _ := time.Parse(time.RFC3339, *r.ManufacturingTimestamp)
	props := &devicepb.DeviceInfoHardwareV1Properties{
		UniqueId:               proto.String(*r.UniqueID),
		ManufacturingTimestamp: timestamppb.New(t),
		SensorType:             sensorTypeByName[*r.SensorType].Enum(),
		EepromFormatVersion: &devicepb.DeviceInfoHardwareV1Properties_EepromFormatVersion{
			Major: proto.Int64(*r.EepromFormatVersion.Major),
			Minor: proto.Int64(*r.EepromFormatVersion.Minor),
		},
		ModulePcbVersion: &devicepb.DeviceInfoHardwareV1Properties_ModulePcbVersion{
			Major: proto.Int64(*r.ModulePcbVersion.Major),
			Minor: proto.Int64(*r.ModulePcbVersion.Minor),
		},
	}
	if r.FuelCellLotNumber != "" {
		props.FuelCellLotNumber = proto.String(r.FuelCellLotNumber)
	}
	if r.FuelCellVendor != "" {
		props.FuelCellVendor = proto.String(r.FuelCellVendor)
	}
	return &devicepb.DeviceInfoHardwareV1{
		EventName:    InfoHardwareV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties:   props,
	}
}
