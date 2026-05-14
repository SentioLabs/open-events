package device

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

// VersionsRequest is the wire shape for the versions nested object.
type VersionsRequest struct {
	ZephyrKernelVersion string `json:"zephyr_kernel_version"`
	ZephyrKernelGitSha  string `json:"zephyr_kernel_git_sha"`
	AppSemanticVersion  string `json:"app_semantic_version"`
	AppGitSha           string `json:"app_git_sha"`
	CompileTime         string `json:"compile_time"`
	CompiledOnOs        string `json:"compiled_on_os"`
	CompiledBy          string `json:"compiled_by"`
	CompilerVersion     string `json:"compiler_version"`
	HwPlatform          string `json:"hw_platform"`
	BuildType           string `json:"build_type"`
	BootloaderGitSha    string `json:"bootloader_git_sha"`
	BootloaderBuildTs   int64  `json:"bootloader_build_timestamp"`
}

// InfoSoftwareRequest is the JSON body for POST /v1/events/device/info/software.
type InfoSoftwareRequest struct {
	Context                DeviceContext   `json:"context"`
	SerialNumber           string          `json:"serial_number"`
	UniqueID               int64           `json:"unique_id"`
	ProductType            string          `json:"product_type"`
	PcbaHwVersion          string          `json:"pcba_hw_version"`
	PcbaHwManufacturedTsMs int64           `json:"pcba_hw_manufactured_timestamp_ms"`
	Versions               VersionsRequest `json:"versions"`
}

// Validate returns field-level errors for the request, empty on success.
func (r InfoSoftwareRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.SerialNumber == "" {
		errs = append(errs, eventmap.FieldError{Field: "serial_number", Message: "required"})
	}
	if r.ProductType == "" {
		errs = append(errs, eventmap.FieldError{Field: "product_type", Message: "required"})
	}
	return errs
}

// ToProto builds a DeviceInfoSoftwareV1 protobuf with a fresh envelope.
func (r InfoSoftwareRequest) ToProto() eventmap.EnvelopeMessage {
	return &devicepb.DeviceInfoSoftwareV1{
		EventName:    InfoSoftwareV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(clientName), Version: proto.String(clientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceInfoSoftwareV1Properties{
			SerialNumber:                  proto.String(r.SerialNumber),
			UniqueId:                      proto.Int64(r.UniqueID),
			ProductType:                   proto.String(r.ProductType),
			PcbaHwVersion:                 proto.String(r.PcbaHwVersion),
			PcbaHwManufacturedTimestampMs: proto.Int64(r.PcbaHwManufacturedTsMs),
			Versions: &devicepb.DeviceInfoSoftwareV1Properties_Versions{
				ZephyrKernelVersion:      proto.String(r.Versions.ZephyrKernelVersion),
				ZephyrKernelGitSha:       proto.String(r.Versions.ZephyrKernelGitSha),
				AppSemanticVersion:       proto.String(r.Versions.AppSemanticVersion),
				AppGitSha:                proto.String(r.Versions.AppGitSha),
				CompileTime:              proto.String(r.Versions.CompileTime),
				CompiledOnOs:             proto.String(r.Versions.CompiledOnOs),
				CompiledBy:               proto.String(r.Versions.CompiledBy),
				CompilerVersion:          proto.String(r.Versions.CompilerVersion),
				HwPlatform:               proto.String(r.Versions.HwPlatform),
				BuildType:                proto.String(r.Versions.BuildType),
				BootloaderGitSha:         proto.String(r.Versions.BootloaderGitSha),
				BootloaderBuildTimestamp: proto.Int64(r.Versions.BootloaderBuildTs),
			},
		},
	}
}
