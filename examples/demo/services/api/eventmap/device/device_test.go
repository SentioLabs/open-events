package device_test

import (
	"slices"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/device"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

func validDeviceContext() device.DeviceContext {
	return device.DeviceContext{
		TenantID: "tenant-1",
		DeviceID: "device-1",
	}
}

func containsField(errs []eventmap.FieldError, field string) bool {
	return slices.ContainsFunc(errs, func(e eventmap.FieldError) bool {
		return e.Field == field
	})
}

func float64p(v float64) *float64 { return &v }
func int64p(v int64) *int64       { return &v }
func stringp(v string) *string    { return &v }

// --- InfoHardware ---

func TestInfoHardware_Validate_RejectsMissingDeviceID(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:                device.DeviceContext{TenantID: "t"},
		UniqueID:               stringp("uid"),
		ManufacturingTimestamp: stringp("2024-01-01T00:00:00Z"),
		SensorType:             stringp("co"),
	}
	errs := req.Validate()
	if !containsField(errs, "context.device_id") {
		t.Fatalf("expected context.device_id error, got %+v", errs)
	}
}

func TestInfoHardware_Validate_RejectsBadSensorType(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:                validDeviceContext(),
		UniqueID:               stringp("uid"),
		ManufacturingTimestamp: stringp("2024-01-01T00:00:00Z"),
		SensorType:             stringp("invalid"),
	}
	errs := req.Validate()
	if !containsField(errs, "sensor_type") {
		t.Fatalf("expected sensor_type error, got %+v", errs)
	}
}

// TestInfoHardware_Validate_RejectsMissingUniqueID guards against the slop-review
// M-2 regression: required string fields that used zero-value `== ""` checks
// would silently accept `{}` POSTs.
func TestInfoHardware_Validate_RejectsMissingUniqueID(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:                validDeviceContext(),
		ManufacturingTimestamp: stringp("2024-01-01T00:00:00Z"),
		SensorType:             stringp("co"),
	}
	errs := req.Validate()
	if !containsField(errs, "unique_id") {
		t.Fatalf("expected unique_id error, got %+v", errs)
	}
}

// TestInfoHardware_Validate_RejectsMissingManufacturingTs guards against the
// slop-review M-3 regression: the registry marks manufacturing_timestamp as
// required, but the old handler silently no-op-ed when it was missing.
func TestInfoHardware_Validate_RejectsMissingManufacturingTs(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:    validDeviceContext(),
		UniqueID:   stringp("abc123"),
		SensorType: stringp("co"),
	}
	errs := req.Validate()
	if !containsField(errs, "manufacturing_timestamp") {
		t.Fatalf("expected manufacturing_timestamp error, got %+v", errs)
	}
}

// TestInfoHardware_Validate_RejectsUnparseableManufacturingTs guards against
// the slop-review M-3 regression: the old handler swallowed time.Parse errors
// silently, so a bad timestamp would publish an envelope with no proto field.
func TestInfoHardware_Validate_RejectsUnparseableManufacturingTs(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:                validDeviceContext(),
		UniqueID:               stringp("abc123"),
		ManufacturingTimestamp: stringp("not-a-timestamp"),
		SensorType:             stringp("co"),
	}
	errs := req.Validate()
	if !containsField(errs, "manufacturing_timestamp") {
		t.Fatalf("expected manufacturing_timestamp error, got %+v", errs)
	}
}

func TestInfoHardware_ToProto_RoundTrip(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:                validDeviceContext(),
		UniqueID:               stringp("abc123"),
		ManufacturingTimestamp: stringp("2024-01-01T00:00:00Z"),
		SensorType:             stringp("oxygen"),
		EepromFormatVersion:    &device.EepromFormatVersionRequest{Major: int64p(1), Minor: int64p(0)},
		ModulePcbVersion:       &device.ModulePcbVersionRequest{Major: int64p(2), Minor: int64p(1)},
	}
	env := req.ToProto()
	wire, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got devicepb.DeviceInfoHardwareV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetEventName() != device.InfoHardwareV1 {
		t.Errorf("EventName: %q", got.GetEventName())
	}
	if got.GetProperties().GetUniqueId() != "abc123" {
		t.Errorf("UniqueId: %q", got.GetProperties().GetUniqueId())
	}
	if got.GetProperties().GetSensorType() != devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_OXYGEN {
		t.Errorf("SensorType: %v", got.GetProperties().GetSensorType())
	}
	if got.GetProperties().GetEepromFormatVersion().GetMajor() != 1 {
		t.Errorf("EepromFormatVersion.Major: %d", got.GetProperties().GetEepromFormatVersion().GetMajor())
	}
}

// --- IncidentTemperature ---

func TestIncidentTemperature_Validate_RejectsBadBreachType(t *testing.T) {
	req := device.IncidentTemperatureRequest{
		Context:    validDeviceContext(),
		DegreesC:   float64p(45.0),
		ThresholdC: float64p(40.0),
		BreachType: stringp("extreme"),
	}
	errs := req.Validate()
	if !containsField(errs, "breach_type") {
		t.Fatalf("expected breach_type error, got %+v", errs)
	}
}

func TestIncidentTemperature_Validate_RejectsMissingDegreesC(t *testing.T) {
	req := device.IncidentTemperatureRequest{
		Context:    validDeviceContext(),
		ThresholdC: float64p(40.0),
		BreachType: stringp("over"),
	}
	errs := req.Validate()
	if !containsField(errs, "degrees_c") {
		t.Fatalf("expected degrees_c error, got %+v", errs)
	}
}

func TestIncidentTemperature_Validate_RejectsMissingThresholdC(t *testing.T) {
	req := device.IncidentTemperatureRequest{
		Context:    validDeviceContext(),
		DegreesC:   float64p(45.0),
		BreachType: stringp("over"),
	}
	errs := req.Validate()
	if !containsField(errs, "threshold_c") {
		t.Fatalf("expected threshold_c error, got %+v", errs)
	}
}

func TestIncidentTemperature_ToProto_RoundTrip(t *testing.T) {
	req := device.IncidentTemperatureRequest{
		Context:    validDeviceContext(),
		DegreesC:   float64p(45.5),
		ThresholdC: float64p(40.0),
		BreachType: stringp("over"),
	}
	env := req.ToProto()
	wire, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got devicepb.DeviceIncidentTemperatureV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.GetProperties().GetBreachType() != devicepb.DeviceIncidentTemperatureV1Properties_BREACH_TYPE_OVER {
		t.Errorf("BreachType: %v", got.GetProperties().GetBreachType())
	}
	if got.GetProperties().GetDegreesC() != 45.5 {
		t.Errorf("DegreesC: %f", got.GetProperties().GetDegreesC())
	}
}

// --- IncidentDrop ---

func TestIncidentDrop_Validate_RejectsMissingPeakAccelerationG(t *testing.T) {
	req := device.IncidentDropRequest{
		Context:    validDeviceContext(),
		DurationMs: int64p(100),
		Axis:       stringp("x"),
	}
	errs := req.Validate()
	if !containsField(errs, "peak_acceleration_g") {
		t.Fatalf("expected peak_acceleration_g error, got %+v", errs)
	}
}

func TestIncidentDrop_Validate_RejectsMissingDurationMs(t *testing.T) {
	req := device.IncidentDropRequest{
		Context:           validDeviceContext(),
		PeakAccelerationG: float64p(9.8),
		Axis:              stringp("x"),
	}
	errs := req.Validate()
	if !containsField(errs, "duration_ms") {
		t.Fatalf("expected duration_ms error, got %+v", errs)
	}
}

// --- InfoCalibration ---

func TestInfoCalibration_Validate_RejectsMissingConcentration(t *testing.T) {
	req := device.InfoCalibrationRequest{
		Context:   validDeviceContext(),
		Integral:  int64p(42),
		Timestamp: stringp("2024-01-01T00:00:00Z"),
	}
	errs := req.Validate()
	if !containsField(errs, "concentration") {
		t.Fatalf("expected concentration error, got %+v", errs)
	}
}

func TestInfoCalibration_Validate_RejectsMissingTimestamp(t *testing.T) {
	req := device.InfoCalibrationRequest{
		Context:       validDeviceContext(),
		Concentration: float64p(0.95),
		Integral:      int64p(42),
	}
	errs := req.Validate()
	if !containsField(errs, "timestamp") {
		t.Fatalf("expected timestamp error, got %+v", errs)
	}
}

// --- InfoSoftware ---

func TestInfoSoftware_Validate_RejectsMissingUniqueID(t *testing.T) {
	req := device.InfoSoftwareRequest{
		Context:                       validDeviceContext(),
		SerialNumber:                  stringp("SN-123"),
		ProductType:                   stringp("z9"),
		PcbaHwManufacturedTimestampMs: int64p(1700000000000),
	}
	errs := req.Validate()
	if !containsField(errs, "unique_id") {
		t.Fatalf("expected unique_id error, got %+v", errs)
	}
}

func TestInfoSoftware_Validate_RejectsMissingPcbaTimestamp(t *testing.T) {
	req := device.InfoSoftwareRequest{
		Context:      validDeviceContext(),
		SerialNumber: stringp("SN-123"),
		UniqueID:     int64p(42),
		ProductType:  stringp("z9"),
	}
	errs := req.Validate()
	if !containsField(errs, "pcba_hw_manufactured_timestamp_ms") {
		t.Fatalf("expected pcba_hw_manufactured_timestamp_ms error, got %+v", errs)
	}
}

// --- DiagnosticsStackUsage ---

func TestDiagnosticsStackUsage_Validate_RejectsMissingHighestUsageThread(t *testing.T) {
	req := device.DiagnosticsStackUsageRequest{
		Context:             validDeviceContext(),
		ThreadCount:         int64p(1),
		HighestUsagePercent: int64p(50),
	}
	errs := req.Validate()
	if !containsField(errs, "highest_usage_thread") {
		t.Fatalf("expected highest_usage_thread error, got %+v", errs)
	}
}

func TestDiagnosticsStackUsage_ToProto_ThreadMapping(t *testing.T) {
	req := device.DiagnosticsStackUsageRequest{
		Context:             validDeviceContext(),
		ThreadCount:         int64p(1),
		HighestUsagePercent: int64p(75),
		HighestUsageThread:  stringp("main"),
		Threads: []*device.ThreadRequest{
			{
				Name:           stringp("main"),
				StackSizeBytes: int64p(4096),
				StackUsedBytes: int64p(3072),
				UsagePercent:   int64p(75),
				Priority:       int64p(5),
				State:          stringp("running"),
			},
		},
	}
	env := req.ToProto()
	wire, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got devicepb.DeviceDiagnosticsStackUsageV1
	if err := proto.Unmarshal(wire, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	threads := got.GetProperties().GetThreads()
	if len(threads) != 1 {
		t.Fatalf("Threads: want 1, got %d", len(threads))
	}
	if threads[0].GetName() != "main" {
		t.Errorf("Thread.Name: %q", threads[0].GetName())
	}
	if threads[0].GetState() != devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_RUNNING {
		t.Errorf("Thread.State: %v", threads[0].GetState())
	}
}
