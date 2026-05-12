package schemair

type Lock struct {
	Version int
	Context map[string]LockedField
	Events  map[string]LockedEvent
}

type LockedEvent struct {
	Envelope   map[string]LockedField
	Properties map[string]LockedField
	Reserved   []ReservedField
}

type LockedField struct {
	StableID    string
	ProtoNumber int
}

type ReservedField struct {
	Name        string
	StableID    string
	ProtoNumber int
	Reason      string
}
