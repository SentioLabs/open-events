package schemair

import "testing"

func TestLockContract(t *testing.T) {
	var _ int = Lock{}.Version
	var _ map[string]LockedField = Lock{}.Context
	var _ map[string]LockedEvent = Lock{}.Events
	var _ map[string]LockedField = LockedEvent{}.Envelope
	var _ map[string]LockedField = LockedEvent{}.Properties
	var _ []ReservedField = LockedEvent{}.Reserved
	var _ string = LockedField{}.StableID
	var _ int = LockedField{}.ProtoNumber
	var _ string = ReservedField{}.Reason
}
