package eventmap

// Canonical event-name strings as used in SQS message attributes,
// matching the openevents.lock.yaml key format "<name>@<version>".
const (
	CheckoutStartedV1   = "checkout.started@1"
	CheckoutCompletedV1 = "checkout.completed@1"
	SearchPerformedV1   = "search.performed@1"
)

// AllEventNames returns the canonical names in deterministic order.
func AllEventNames() []string {
	return []string{CheckoutStartedV1, CheckoutCompletedV1, SearchPerformedV1}
}
