package device

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

// EepromFormatVersionRequest is the wire shape for the eeprom_format_version object.
type EepromFormatVersionRequest struct {
	Major int64 `json:"major"`
	Minor int64 `json:"minor"`
}

// ModulePcbVersionRequest is the wire shape for the module_pcb_version object.
type ModulePcbVersionRequest struct {
	Major int64 `json:"major"`
	Minor int64 `json:"minor"`
}

// InfoHardwareRequest is the JSON body for POST /v1/events/device/info/hardware.
type InfoHardwareRequest struct {
	Context             DeviceContext              `json:"context"`
	UniqueID            string                     `json:"unique_id"`
	ManufacturingTs     string                     `json:"manufacturing_timestamp"`
	EepromFormatVersion EepromFormatVersionRequest `json:"eeprom_format_version"`
	ModulePcbVersion    ModulePcbVersionRequest    `json:"module_pcb_version"`
	SensorType          string                     `json:"sensor_type"` // "co"|"alcohol"|"oxygen"|"fuel_cell"
	FuelCellLotNumber   string                     `json:"fuel_cell_lot_number,omitempty"`
	FuelCellVendor      string                     `json:"fuel_cell_vendor,omitempty"`
}

var sensorTypeByName = map[string]devicepb.DeviceInfoHardwareV1Properties_SensorType{
	"unspecified": devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_UNSPECIFIED,
	"co":          devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_CO,
	"alcohol":     devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_ALCOHOL,
	"oxygen":      devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_OXYGEN,
	"fuel_cell":   devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_FUEL_CELL,
}

// Validate returns field-level errors for the request, empty on success.
func (r InfoHardwareRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.UniqueID == "" {
		errs = append(errs, eventmap.FieldError{Field: "unique_id", Message: "required"})
	}
	if _, ok := sensorTypeByName[r.SensorType]; !ok {
		errs = append(errs, eventmap.FieldError{Field: "sensor_type", Message: "must be one of unspecified|co|alcohol|oxygen|fuel_cell"})
	}
	return errs
}

// ToProto builds a DeviceInfoHardwareV1 protobuf with a fresh envelope.
func (r InfoHardwareRequest) ToProto() eventmap.EnvelopeMessage {
	props := &devicepb.DeviceInfoHardwareV1Properties{
		UniqueId:   proto.String(r.UniqueID),
		SensorType: sensorTypeByName[r.SensorType].Enum(),
		EepromFormatVersion: &devicepb.DeviceInfoHardwareV1Properties_EepromFormatVersion{
			Major: proto.Int64(r.EepromFormatVersion.Major),
			Minor: proto.Int64(r.EepromFormatVersion.Minor),
		},
		ModulePcbVersion: &devicepb.DeviceInfoHardwareV1Properties_ModulePcbVersion{
			Major: proto.Int64(r.ModulePcbVersion.Major),
			Minor: proto.Int64(r.ModulePcbVersion.Minor),
		},
	}
	if r.ManufacturingTs != "" {
		props.ManufacturingTimestamp = timestamppb.Now() // demo: timestamp string parsing is out of scope for this task
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
		Client:       &commonpb.Client{Name: proto.String(clientName), Version: proto.String(clientVersion)},
		Context:      contextToProto(r.Context),
		Properties:   props,
	}
}
