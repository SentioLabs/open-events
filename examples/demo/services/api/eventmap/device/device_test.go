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
		TenantId: "tenant-1",
		DeviceId: "device-1",
	}
}

func containsField(errs []eventmap.FieldError, field string) bool {
	return slices.ContainsFunc(errs, func(e eventmap.FieldError) bool {
		return e.Field == field
	})
}

// --- InfoHardware ---

func TestInfoHardware_Validate_RejectsMissingDeviceID(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:    device.DeviceContext{TenantId: "t"},
		UniqueID:   "uid",
		SensorType: "co",
	}
	errs := req.Validate()
	if !containsField(errs, "context.device_id") {
		t.Fatalf("expected context.device_id error, got %+v", errs)
	}
}

func TestInfoHardware_Validate_RejectsBadSensorType(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:    validDeviceContext(),
		UniqueID:   "uid",
		SensorType: "invalid",
	}
	errs := req.Validate()
	if !containsField(errs, "sensor_type") {
		t.Fatalf("expected sensor_type error, got %+v", errs)
	}
}

func TestInfoHardware_ToProto_RoundTrip(t *testing.T) {
	req := device.InfoHardwareRequest{
		Context:             validDeviceContext(),
		UniqueID:            "abc123",
		ManufacturingTs:     "2024-01-01T00:00:00Z",
		SensorType:          "oxygen",
		EepromFormatVersion: device.EepromFormatVersionRequest{Major: 1, Minor: 0},
		ModulePcbVersion:    device.ModulePcbVersionRequest{Major: 2, Minor: 1},
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
		DegreesC:   45.0,
		ThresholdC: 40.0,
		BreachType: "extreme",
	}
	errs := req.Validate()
	if !containsField(errs, "breach_type") {
		t.Fatalf("expected breach_type error, got %+v", errs)
	}
}

func TestIncidentTemperature_ToProto_RoundTrip(t *testing.T) {
	req := device.IncidentTemperatureRequest{
		Context:    validDeviceContext(),
		DegreesC:   45.5,
		ThresholdC: 40.0,
		BreachType: "over",
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

// --- DiagnosticsStackUsage ---

func TestDiagnosticsStackUsage_Validate_RejectsMissingHighestUsageThread(t *testing.T) {
	req := device.DiagnosticsStackUsageRequest{
		Context:             validDeviceContext(),
		ThreadCount:         1,
		HighestUsagePercent: 50,
	}
	errs := req.Validate()
	if !containsField(errs, "highest_usage_thread") {
		t.Fatalf("expected highest_usage_thread error, got %+v", errs)
	}
}

func TestDiagnosticsStackUsage_ToProto_ThreadMapping(t *testing.T) {
	req := device.DiagnosticsStackUsageRequest{
		Context:             validDeviceContext(),
		ThreadCount:         1,
		HighestUsagePercent: 75,
		HighestUsageThread:  "main",
		Threads: []device.ThreadRequest{
			{
				Name:           "main",
				StackSizeBytes: 4096,
				StackUsedBytes: 3072,
				UsagePercent:   75,
				Priority:       5,
				State:          "running",
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
