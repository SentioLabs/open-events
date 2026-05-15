package device

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/common/v1"
	devicepb "github.com/sentiolabs/open-events/examples/demo/gen/go/com/acme/platform/device/v1"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

var threadStateByName = map[string]devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_State{
	"running":   devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_RUNNING,
	"ready":     devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_READY,
	"pending":   devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_PENDING,
	"suspended": devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_SUSPENDED,
	"dead":      devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_DEAD,
}

// ToProto builds a DeviceDiagnosticsStackUsageV1 protobuf with a fresh envelope.
// Callers must invoke Validate() first.
func (r DiagnosticsStackUsageRequest) ToProto() eventmap.EnvelopeMessage {
	threads := make([]*devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads, 0, len(r.Threads))
	for _, t := range r.Threads {
		threads = append(threads, &devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads{
			Name:           proto.String(*t.Name),
			StackSizeBytes: proto.Int64(*t.StackSizeBytes),
			StackUsedBytes: proto.Int64(*t.StackUsedBytes),
			UsagePercent:   proto.Int64(*t.UsagePercent),
			Priority:       proto.Int64(*t.Priority),
			State:          threadStateByName[*t.State].Enum(),
		})
	}
	return &devicepb.DeviceDiagnosticsStackUsageV1{
		EventName:    DiagnosticsStackUsageV1,
		EventVersion: 1,
		EventId:      uuid.NewString(),
		EventTs:      timestamppb.Now(),
		Client:       &commonpb.Client{Name: proto.String(eventmap.ClientName), Version: proto.String(eventmap.ClientVersion)},
		Context:      contextToProto(r.Context),
		Properties: &devicepb.DeviceDiagnosticsStackUsageV1Properties{
			ThreadCount:         proto.Int64(*r.ThreadCount),
			HighestUsagePercent: proto.Int64(*r.HighestUsagePercent),
			HighestUsageThread:  proto.String(*r.HighestUsageThread),
			Threads:             threads,
		},
	}
}
