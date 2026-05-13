package eventmap

import "testing"

// --- Contract assertions ---
// Mirror in examples/demo/services/consumer/tests/test_contract_event_names.py.

func TestContractEventNames(t *testing.T) {
	want := []string{
		"checkout.started@1",
		"checkout.completed@1",
		"search.performed@1",
	}
	got := AllEventNames()
	if len(got) != len(want) {
		t.Fatalf("AllEventNames length: got %d want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("AllEventNames[%d]: got %q want %q", i, got[i], w)
		}
	}
}
