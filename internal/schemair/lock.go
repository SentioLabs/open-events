package schemair

type Lock struct {
	Version int
	Domains map[string]LockedDomain
	Events  map[string]LockedEvent
}

type LockedDomain struct {
	Context map[string]LockedField
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
