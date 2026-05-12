package schemair

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
)

func TestUpdateLockAllocatesStableContextNumbers(t *testing.T) {
	reg := registry.Registry{
		Context: map[string]registry.Field{
			"tenant_id": {Name: "tenant_id"},
			"region":    {Name: "region"},
		},
	}

	lock, err := UpdateLock(Lock{}, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}

	if lock.Context["region"].ProtoNumber != 1 {
		t.Fatalf("region ProtoNumber = %d, want 1", lock.Context["region"].ProtoNumber)
	}
	if lock.Context["tenant_id"].ProtoNumber != 2 {
		t.Fatalf("tenant_id ProtoNumber = %d, want 2", lock.Context["tenant_id"].ProtoNumber)
	}

	reg.Context["country"] = registry.Field{Name: "country"}

	updated, err := UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() second error = %v", err)
	}

	if updated.Context["region"].ProtoNumber != lock.Context["region"].ProtoNumber {
		t.Fatalf("region ProtoNumber changed from %d to %d", lock.Context["region"].ProtoNumber, updated.Context["region"].ProtoNumber)
	}
	if updated.Context["tenant_id"].ProtoNumber != lock.Context["tenant_id"].ProtoNumber {
		t.Fatalf("tenant_id ProtoNumber changed from %d to %d", lock.Context["tenant_id"].ProtoNumber, updated.Context["tenant_id"].ProtoNumber)
	}
	if updated.Context["country"].ProtoNumber != 3 {
		t.Fatalf("country ProtoNumber = %d, want 3", updated.Context["country"].ProtoNumber)
	}
}

func TestUpdateLockRejectsDuplicateRegistryEventKeys(t *testing.T) {
	reg := registry.Registry{Events: []registry.Event{
		{
			Name:    "checkout.completed",
			Version: 1,
			Properties: map[string]registry.Field{
				"amount": {Name: "amount"},
			},
		},
		{
			Name:    "checkout.completed",
			Version: 1,
			Properties: map[string]registry.Field{
				"order_id": {Name: "order_id"},
			},
		},
	}}

	_, err := UpdateLock(Lock{}, reg)
	if err == nil {
		t.Fatalf("UpdateLock() error = nil, want duplicate event key error")
	}
	if !strings.Contains(err.Error(), "duplicate event key checkout.completed@1") {
		t.Fatalf("UpdateLock() error = %q, want duplicate event key", err)
	}
}

func TestUpdateLockPreservesExistingPropertyNumbers(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"order_id": {Name: "order_id"},
			"amount":   {Name: "amount"},
		},
	}
	reg := registry.Registry{Events: []registry.Event{event}}

	lock, err := UpdateLock(Lock{}, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}

	event.Properties["coupon_code"] = registry.Field{Name: "coupon_code"}
	reg.Events = []registry.Event{event}

	updated, err := UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() second error = %v", err)
	}

	key := eventKey(event)
	if updated.Events[key].Properties["amount"].ProtoNumber != lock.Events[key].Properties["amount"].ProtoNumber {
		t.Fatalf("amount ProtoNumber changed")
	}
	if updated.Events[key].Properties["order_id"].ProtoNumber != lock.Events[key].Properties["order_id"].ProtoNumber {
		t.Fatalf("order_id ProtoNumber changed")
	}
	if updated.Events[key].Properties["coupon_code"].ProtoNumber != 3 {
		t.Fatalf("coupon_code ProtoNumber = %d, want 3", updated.Events[key].Properties["coupon_code"].ProtoNumber)
	}
}

func TestUpdateLockDoesNotReuseReservedNumbers(t *testing.T) {
	existing := Lock{
		Context: map[string]LockedField{
			"tenant_id": {StableID: "tenant_id", ProtoNumber: 1},
		},
	}
	reg := registry.Registry{Context: map[string]registry.Field{"region": {Name: "region"}}}

	updated, err := UpdateLock(existing, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}

	if updated.Context["region"].ProtoNumber != 2 {
		t.Fatalf("region ProtoNumber = %d, want 2", updated.Context["region"].ProtoNumber)
	}
}

func TestUpdateLockSkipsProtobufReservedRange(t *testing.T) {
	existing := Lock{Context: map[string]LockedField{"field": {StableID: "field", ProtoNumber: 18999}}}
	reg := registry.Registry{Context: map[string]registry.Field{"field": {Name: "field"}, "next": {Name: "next"}}}

	updated, err := UpdateLock(existing, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}

	if updated.Context["next"].ProtoNumber != 20000 {
		t.Fatalf("next ProtoNumber = %d, want 20000", updated.Context["next"].ProtoNumber)
	}
}

func TestUpdateLockMovesRemovedPropertyToReserved(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount":      {Name: "amount"},
			"coupon_code": {Name: "coupon_code"},
		},
	}
	reg := registry.Registry{Events: []registry.Event{event}}

	lock, err := UpdateLock(Lock{}, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}

	removedNumber := lock.Events[eventKey(event)].Properties["coupon_code"].ProtoNumber
	delete(event.Properties, "coupon_code")
	reg.Events = []registry.Event{event}

	updated, err := UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after removal error = %v", err)
	}

	updatedEvent := updated.Events[eventKey(event)]
	if _, ok := updatedEvent.Properties["coupon_code"]; ok {
		t.Fatalf("coupon_code remained active after removal")
	}
	wantReserved := ReservedField{
		Name:        "coupon_code",
		StableID:    "coupon_code",
		ProtoNumber: removedNumber,
		Reason:      "field removed",
	}
	if !reflect.DeepEqual(updatedEvent.Reserved, []ReservedField{wantReserved}) {
		t.Fatalf("Reserved = %#v, want %#v", updatedEvent.Reserved, []ReservedField{wantReserved})
	}

	event.Properties["tax"] = registry.Field{Name: "tax"}
	reg.Events = []registry.Event{event}

	updated, err = UpdateLock(updated, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after adding property error = %v", err)
	}

	if updated.Events[eventKey(event)].Properties["tax"].ProtoNumber != removedNumber+1 {
		t.Fatalf("tax ProtoNumber = %d, want %d", updated.Events[eventKey(event)].Properties["tax"].ProtoNumber, removedNumber+1)
	}
}

func TestUpdateLockPreservesRemovedContextNumbers(t *testing.T) {
	reg := registry.Registry{
		Context: map[string]registry.Field{
			"account_id": {Name: "account_id"},
			"tenant_id":  {Name: "tenant_id"},
		},
	}

	lock, err := UpdateLock(Lock{}, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}

	removedNumber := lock.Context["tenant_id"].ProtoNumber
	delete(reg.Context, "tenant_id")

	updated, err := UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after removal error = %v", err)
	}
	if updated.Context["tenant_id"].ProtoNumber != removedNumber {
		t.Fatalf("removed tenant_id ProtoNumber = %d, want preserved %d", updated.Context["tenant_id"].ProtoNumber, removedNumber)
	}

	reg.Context["region"] = registry.Field{Name: "region"}
	updated, err = UpdateLock(updated, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after adding context field error = %v", err)
	}

	if updated.Context["region"].ProtoNumber != removedNumber+1 {
		t.Fatalf("region ProtoNumber = %d, want %d", updated.Context["region"].ProtoNumber, removedNumber+1)
	}
}

func TestCheckLockRejectsMissingField(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"coupon_code": {Name: "coupon_code"},
		},
	}
	reg := registry.Registry{Events: []registry.Event{event}}

	lock := Lock{
		Version: LockVersion,
		Events: map[string]LockedEvent{
			eventKey(event): {
				Envelope: map[string]LockedField{
					"event_name":    {StableID: "event_name", ProtoNumber: 1},
					"event_version": {StableID: "event_version", ProtoNumber: 2},
					"event_id":      {StableID: "event_id", ProtoNumber: 3},
					"event_ts":      {StableID: "event_ts", ProtoNumber: 4},
					"client":        {StableID: "client", ProtoNumber: 5},
					"context":       {StableID: "context", ProtoNumber: 6},
					"properties":    {StableID: "properties", ProtoNumber: 7},
				},
				Properties: map[string]LockedField{},
			},
		},
	}

	err := CheckLock(lock, reg)
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "events.checkout.completed@1.properties.coupon_code is missing") {
		t.Fatalf("CheckLock() error = %q", err)
	}
}

func TestCompareReservedFieldsRejectsDuplicateReservedNumbers(t *testing.T) {
	actual := []ReservedField{
		{Name: "legacy_coupon_code", StableID: "legacy_coupon_code", ProtoNumber: 9, Reason: "field removed"},
		{Name: "coupon_code", StableID: "coupon_code", ProtoNumber: 9, Reason: "field removed"},
	}
	expected := []ReservedField{
		{Name: "coupon_code", StableID: "coupon_code", ProtoNumber: 9, Reason: "field removed"},
	}

	err := compareReservedFields("checkout.completed@1", actual, expected)
	if err == nil {
		t.Fatalf("compareReservedFields() error = nil, want duplicate reserved number error")
	}
	if !strings.Contains(err.Error(), "duplicate proto numbers") || !strings.Contains(err.Error(), "events.checkout.completed@1.reserved") {
		t.Fatalf("compareReservedFields() error = %q, want duplicate reserved number path", err)
	}
}

func TestCheckDuplicateNumbersReportsSortedFieldNames(t *testing.T) {
	for i := 0; i < 100; i++ {
		fields := map[string]LockedField{
			"zulu":  {StableID: "zulu", ProtoNumber: 3},
			"alpha": {StableID: "alpha", ProtoNumber: 3},
		}

		err := checkDuplicateNumbers("message", fields, nil)
		if err == nil {
			t.Fatalf("checkDuplicateNumbers() error = nil, want duplicate error")
		}
		want := "schema lock has duplicate proto numbers in message: alpha and zulu share 3"
		if err.Error() != want {
			t.Fatalf("checkDuplicateNumbers() error = %q, want %q", err, want)
		}
	}
}

func TestSortedReservedFieldsUsesStableTieBreakers(t *testing.T) {
	fields := []ReservedField{
		{Name: "coupon_code", StableID: "zeta", ProtoNumber: 9, Reason: "field removed"},
		{Name: "coupon_code", StableID: "alpha", ProtoNumber: 9, Reason: "renamed"},
		{Name: "coupon_code", StableID: "alpha", ProtoNumber: 9, Reason: "field removed"},
	}

	got := sortedReservedFields(fields)
	want := []ReservedField{fields[2], fields[1], fields[0]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedReservedFields() = %#v, want %#v", got, want)
	}
}

func TestCheckLockRejectsDuplicateNumbersWithinMessage(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount":   {Name: "amount"},
			"order_id": {Name: "order_id"},
		},
	}
	reg := registry.Registry{Events: []registry.Event{event}}

	lock := Lock{
		Version: LockVersion,
		Events: map[string]LockedEvent{
			eventKey(event): {
				Envelope: map[string]LockedField{
					"event_name":    {StableID: "event_name", ProtoNumber: 1},
					"event_version": {StableID: "event_version", ProtoNumber: 2},
					"event_id":      {StableID: "event_id", ProtoNumber: 3},
					"event_ts":      {StableID: "event_ts", ProtoNumber: 4},
					"client":        {StableID: "client", ProtoNumber: 5},
					"context":       {StableID: "context", ProtoNumber: 6},
					"properties":    {StableID: "properties", ProtoNumber: 7},
				},
				Properties: map[string]LockedField{
					"amount":   {StableID: "amount", ProtoNumber: 1},
					"order_id": {StableID: "order_id", ProtoNumber: 1},
				},
			},
		},
	}

	err := CheckLock(lock, reg)
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("CheckLock() error = %q, want duplicate error", err)
	}
}

func TestCheckLockRejectsRemovedPropertyStillActive(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount": {Name: "amount"},
		},
	}
	reg := registry.Registry{Events: []registry.Event{event}}

	lock := Lock{
		Version: LockVersion,
		Events: map[string]LockedEvent{
			eventKey(event): {
				Envelope: lockedEnvelopeForTest(),
				Properties: map[string]LockedField{
					"amount":      {StableID: "amount", ProtoNumber: 1},
					"coupon_code": {StableID: "coupon_code", ProtoNumber: 2},
				},
			},
		},
	}

	err := CheckLock(lock, reg)
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "events.checkout.completed@1.reserved.coupon_code is missing") {
		t.Fatalf("CheckLock() error = %q", err)
	}
}

func TestCheckLockRejectsStaleActiveEvent(t *testing.T) {
	event := registry.Event{Name: "checkout.completed", Version: 1}
	lock := Lock{
		Version: LockVersion,
		Events: map[string]LockedEvent{
			eventKey(event): {
				Envelope:   lockedEnvelopeForTest(),
				Properties: map[string]LockedField{},
			},
		},
	}

	err := CheckLock(lock, registry.Registry{})
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "events.checkout.completed@1 is not in registry") {
		t.Fatalf("CheckLock() error = %q", err)
	}
}

func TestCheckLockDoesNotMutateLock(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"coupon_code": {Name: "coupon_code"},
		},
	}
	reg := registry.Registry{Events: []registry.Event{event}}
	lock := Lock{
		Version: LockVersion,
		Context: map[string]LockedField{
			"tenant_id": {StableID: "tenant_id", ProtoNumber: 1},
		},
		Events: map[string]LockedEvent{
			eventKey(event): {
				Envelope:   lockedEnvelopeForTest(),
				Properties: map[string]LockedField{},
				Reserved:   nil,
			},
		},
	}
	before := cloneLockForTest(lock)

	err := CheckLock(lock, reg)
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want stale lock error")
	}
	if lock.Events[eventKey(event)].Reserved != nil {
		t.Fatalf("CheckLock() changed Reserved from nil to %#v", lock.Events[eventKey(event)].Reserved)
	}
	if !reflect.DeepEqual(lock, before) {
		t.Fatalf("CheckLock() mutated lock: got %#v want %#v", lock, before)
	}
}

func lockedEnvelopeForTest() map[string]LockedField {
	return map[string]LockedField{
		"event_name":    {StableID: "event_name", ProtoNumber: 1},
		"event_version": {StableID: "event_version", ProtoNumber: 2},
		"event_id":      {StableID: "event_id", ProtoNumber: 3},
		"event_ts":      {StableID: "event_ts", ProtoNumber: 4},
		"client":        {StableID: "client", ProtoNumber: 5},
		"context":       {StableID: "context", ProtoNumber: 6},
		"properties":    {StableID: "properties", ProtoNumber: 7},
	}
}

func cloneLockForTest(lock Lock) Lock {
	clone := Lock{
		Version: lock.Version,
		Context: cloneLockedFieldsForTest(lock.Context),
		Events:  make(map[string]LockedEvent, len(lock.Events)),
	}
	if lock.Events == nil {
		clone.Events = nil
	}
	for key, event := range lock.Events {
		clone.Events[key] = LockedEvent{
			Envelope:   cloneLockedFieldsForTest(event.Envelope),
			Properties: cloneLockedFieldsForTest(event.Properties),
			Reserved:   cloneReservedFieldsForTest(event.Reserved),
		}
	}
	return clone
}

func cloneLockedFieldsForTest(fields map[string]LockedField) map[string]LockedField {
	if fields == nil {
		return nil
	}
	clone := make(map[string]LockedField, len(fields))
	for name, field := range fields {
		clone[name] = field
	}
	return clone
}

func cloneReservedFieldsForTest(fields []ReservedField) []ReservedField {
	if fields == nil {
		return nil
	}
	clone := make([]ReservedField, len(fields))
	copy(clone, fields)
	return clone
}
