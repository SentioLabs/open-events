package device

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/common/v1"
	devicepb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/device/v1"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

// ToProto builds a DeviceInfoSoftwareV1 protobuf with a fresh envelope.
// Callers must invoke Validate() first.
func (r InfoSoftwareRequest) ToProto() eventmap.EnvelopeMessage {
	return &devicepb.DeviceInfoSoftwareV1{
		EventName:    InfoSoftwareV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceInfoSoftwareV1Properties{
			SerialNumber:                  proto.String(*r.SerialNumber),
			UniqueId:                      proto.Int64(*r.UniqueID),
			ProductType:                   proto.String(*r.ProductType),
			PcbaHwVersion:                 proto.String(*r.PcbaHwVersion),
			PcbaHwManufacturedTimestampMs: proto.Int64(*r.PcbaHwManufacturedTimestampMs),
			Versions: &devicepb.DeviceInfoSoftwareV1Properties_Versions{
				ZephyrKernelVersion:      proto.String(*r.Versions.ZephyrKernelVersion),
				ZephyrKernelGitSha:       proto.String(*r.Versions.ZephyrKernelGitSha),
				AppSemanticVersion:       proto.String(*r.Versions.AppSemanticVersion),
				AppGitSha:                proto.String(*r.Versions.AppGitSha),
				CompileTime:              proto.String(*r.Versions.CompileTime),
				CompiledOnOs:             proto.String(*r.Versions.CompiledOnOS),
				CompiledBy:               proto.String(*r.Versions.CompiledBy),
				CompilerVersion:          proto.String(*r.Versions.CompilerVersion),
				HwPlatform:               proto.String(*r.Versions.HwPlatform),
				BuildType:                proto.String(*r.Versions.BuildType),
				BootloaderGitSha:         proto.String(*r.Versions.BootloaderGitSha),
				BootloaderBuildTimestamp: proto.Int64(*r.Versions.BootloaderBuildTimestamp),
			},
		},
	}
}
