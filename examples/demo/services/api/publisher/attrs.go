package publisher

// SQS message attribute names used by the OpenEvents demo wire format.
// The Python consumer reads the same names from the same locations.
const (
	AttrEventName = "event_name" // "checkout.started@1"
	AttrSchema    = "schema"     // SchemaValue below
)

// SchemaValue identifies the registry namespace + version this demo emits.
const SchemaValue = "openevents:com.acme.storefront/v1"
