package device

import (
	"github.com/labstack/echo/v4"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

// Routes returns all device-domain event routes.
func Routes() []eventmap.Route {
	return []eventmap.Route{
		{Path: "/v1/events/device/info/hardware", EventName: InfoHardwareV1, Build: buildInfoHardware},
		{Path: "/v1/events/device/info/software", EventName: InfoSoftwareV1, Build: buildInfoSoftware},
		{Path: "/v1/events/device/info/calibration", EventName: InfoCalibrationV1, Build: buildInfoCalibration},
		{Path: "/v1/events/device/incident/temperature", EventName: IncidentTemperatureV1, Build: buildIncidentTemperature},
		{Path: "/v1/events/device/incident/drop", EventName: IncidentDropV1, Build: buildIncidentDrop},
		{Path: "/v1/events/device/diagnostics/stack_usage", EventName: DiagnosticsStackUsageV1, Build: buildDiagnosticsStackUsage},
	}
}

func buildInfoHardware(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[InfoHardwareRequest](c, func(r InfoHardwareRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildInfoSoftware(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[InfoSoftwareRequest](c, func(r InfoSoftwareRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildInfoCalibration(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[InfoCalibrationRequest](c, func(r InfoCalibrationRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildIncidentTemperature(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[IncidentTemperatureRequest](c, func(r IncidentTemperatureRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildIncidentDrop(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[IncidentDropRequest](c, func(r IncidentDropRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildDiagnosticsStackUsage(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[DiagnosticsStackUsageRequest](c, func(r DiagnosticsStackUsageRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}
