package publisher

import (
	"context"
	"testing"
)

func TestFakePublisher_RecordsCalls(t *testing.T) {
	f := &FakePublisher{}
	attrs := map[string]string{AttrEventName: "checkout.started@1", AttrSchema: SchemaValue}

	id, err := f.Publish(context.Background(), "payload", attrs)
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
	if call.Body != "payload" {
		t.Errorf("Body: got %q want %q", call.Body, "payload")
	}
	if call.Attrs[AttrEventName] != "checkout.started@1" {
		t.Errorf("Attrs[%q]: got %q want %q", AttrEventName, call.Attrs[AttrEventName], "checkout.started@1")
	}
	if call.Attrs[AttrSchema] != SchemaValue {
		t.Errorf("Attrs[%q]: got %q want %q", AttrSchema, call.Attrs[AttrSchema], SchemaValue)
	}
}

var _ Publisher = (*SQSPublisher)(nil)
var _ Publisher = (*FakePublisher)(nil)
