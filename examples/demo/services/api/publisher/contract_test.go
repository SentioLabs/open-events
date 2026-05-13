package publisher

import "testing"

// --- Contract assertions ---
// Mirror in examples/demo/services/consumer/tests/test_contract_attrs.py.
// Do NOT change values without updating the approved plan.

func TestContractAttrNames(t *testing.T) {
	if AttrEventName != "event_name" {
		t.Fatalf("AttrEventName drift: %q", AttrEventName)
	}
	if AttrSchema != "schema" {
		t.Fatalf("AttrSchema drift: %q", AttrSchema)
	}
}

func TestContractSchemaValue(t *testing.T) {
	if SchemaValue != "openevents:com.acme.storefront/v1" {
		t.Fatalf("SchemaValue drift: %q", SchemaValue)
	}
}
