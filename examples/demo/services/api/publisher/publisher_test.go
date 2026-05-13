package publisher

import (
	"context"
	"testing"
)

func TestFakePublisher_RecordsCalls(t *testing.T) {
	f := &FakePublisher{}
	body := []byte("payload")
	attrs := map[string]string{AttrEventName: "checkout.started@1", AttrSchema: SchemaValue}

	id, err := f.Publish(context.Background(), "checkout.started@1", body, attrs)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if id == "" {
		t.Fatal("Publish returned empty message id")
	}
	if len(f.Calls) != 1 {
		t.Fatalf("Calls length: got %d want 1", len(f.Calls))
	}

	call := f.Calls[0]
	if call.EventName != "checkout.started@1" {
		t.Errorf("EventName: got %q want %q", call.EventName, "checkout.started@1")
	}
	if string(call.Body) != "payload" {
		t.Errorf("Body: got %q want %q", string(call.Body), "payload")
	}
	if call.Attrs[AttrEventName] != "checkout.started@1" {
		t.Errorf("Attrs[%q]: got %q want %q", AttrEventName, call.Attrs[AttrEventName], "checkout.started@1")
	}
	if call.Attrs[AttrSchema] != SchemaValue {
		t.Errorf("Attrs[%q]: got %q want %q", AttrSchema, call.Attrs[AttrSchema], SchemaValue)
	}
}

func TestFakePublisher_DefensiveCopies(t *testing.T) {
	f := &FakePublisher{}
	body := []byte("hello")
	attrs := map[string]string{AttrEventName: "test@1"}

	_, err := f.Publish(context.Background(), "test@1", body, attrs)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	// Mutate the original inputs.
	body[0] = 'X'
	attrs[AttrEventName] = "mutated"

	if string(f.Calls[0].Body) != "hello" {
		t.Errorf("Body should be defensively copied; got %q", string(f.Calls[0].Body))
	}
	if f.Calls[0].Attrs[AttrEventName] != "test@1" {
		t.Errorf("Attrs should be defensively copied; got %q", f.Calls[0].Attrs[AttrEventName])
	}
}

// Compile-time assertions that the public types implement the Publisher
// interface. If a future change drops the method, this stops the build.
var _ Publisher = (*SQSPublisher)(nil)
var _ Publisher = (*FakePublisher)(nil)
