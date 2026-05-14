package device

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	commonpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/common"
	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
)

// ThreadRequest is the wire shape for a single thread in the stack usage snapshot.
type ThreadRequest struct {
	Name           string `json:"name"`
	StackSizeBytes *int64 `json:"stack_size_bytes"` // required; pointer distinguishes 0 from omitted
	StackUsedBytes *int64 `json:"stack_used_bytes"` // required; pointer distinguishes 0 from omitted
	UsagePercent   *int64 `json:"usage_percent"`    // required; pointer distinguishes 0 from omitted
	Priority       *int64 `json:"priority"`         // required; pointer distinguishes 0 from omitted
	State          string `json:"state"`            // "running"|"ready"|"pending"|"suspended"|"dead"
}

// DiagnosticsStackUsageRequest is the JSON body for POST /v1/events/device/diagnostics/stack_usage.
type DiagnosticsStackUsageRequest struct {
	Context             DeviceContext   `json:"context"`
	ThreadCount         *int64          `json:"thread_count"`          // required; pointer distinguishes 0 from omitted
	HighestUsagePercent *int64          `json:"highest_usage_percent"` // required; pointer distinguishes 0 from omitted
	HighestUsageThread  string          `json:"highest_usage_thread"`
	Threads             []ThreadRequest `json:"threads"`
}

var threadStateByName = map[string]devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_State{
	"running":   devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_RUNNING,
	"ready":     devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_READY,
	"pending":   devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_PENDING,
	"suspended": devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_SUSPENDED,
	"dead":      devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_DEAD,
}

// Validate returns field-level errors for the request, empty on success.
func (r DiagnosticsStackUsageRequest) Validate() []eventmap.FieldError {
	errs := validateContext(r.Context)
	if r.ThreadCount == nil {
		errs = append(errs, eventmap.FieldError{Field: "thread_count", Message: "required"})
	}
	if r.HighestUsagePercent == nil {
		errs = append(errs, eventmap.FieldError{Field: "highest_usage_percent", Message: "required"})
	}
	if r.HighestUsageThread == "" {
		errs = append(errs, eventmap.FieldError{Field: "highest_usage_thread", Message: "required"})
	}
	return errs
}

// ToProto builds a DeviceDiagnosticsStackUsageV1 protobuf with a fresh envelope.
func (r DiagnosticsStackUsageRequest) ToProto() eventmap.EnvelopeMessage {
	threads := make([]*devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads, 0, len(r.Threads))
	for _, t := range r.Threads {
		state := devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads_STATE_UNSPECIFIED
		if s, ok := threadStateByName[t.State]; ok {
			state = s
		}
		thread := &devicepb.DeviceDiagnosticsStackUsageV1Properties_Threads{
			Name:  proto.String(t.Name),
			State: state.Enum(),
		}
		if t.StackSizeBytes != nil {
			thread.StackSizeBytes = proto.Int64(*t.StackSizeBytes)
		}
		if t.StackUsedBytes != nil {
			thread.StackUsedBytes = proto.Int64(*t.StackUsedBytes)
		}
		if t.UsagePercent != nil {
			thread.UsagePercent = proto.Int64(*t.UsagePercent)
		}
		if t.Priority != nil {
			thread.Priority = proto.Int64(*t.Priority)
		}
		threads = append(threads, thread)
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
			HighestUsageThread:  proto.String(r.HighestUsageThread),
			Threads:             threads,
		},
	}
}
