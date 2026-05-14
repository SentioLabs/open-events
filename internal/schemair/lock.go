package schemair

type Lock struct {
	Version int
	Domains map[string]LockedDomain
	Events  map[string]LockedEvent
}

type LockedDomain struct {
	Context  map[string]LockedField
	Reserved []ReservedField
}

type LockedEvent struct {
	Envelope   map[string]LockedField
	Properties map[string]LockedField
	Reserved   []ReservedField
}

type LockedField struct {
	StableID    string
	ProtoNumber int
	// Properties and Reserved track nested subfield numbers for object-typed fields.
	// Absent for non-object fields; treated as "fresh — allocate now" for pre-existing
	// lockfiles that predate recursive locking (backward-compatible, no version bump).
	Properties map[string]LockedField
	Reserved   []ReservedField
}

type ReservedField struct {
	Name        string
	StableID    string
	ProtoNumber int
	Reason      string
}
