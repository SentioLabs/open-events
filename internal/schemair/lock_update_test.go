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

func TestUpdateLockReaddedSameNamePropertyPreservesReservedHistory(t *testing.T) {
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

	key := eventKey(event)
	removedNumber := lock.Events[key].Properties["coupon_code"].ProtoNumber
	delete(event.Properties, "coupon_code")
	reg.Events = []registry.Event{event}

	lock, err = UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after removal error = %v", err)
	}

	event.Properties["coupon_code"] = registry.Field{Name: "coupon_code"}
	reg.Events = []registry.Event{event}

	updated, err := UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after re-add error = %v", err)
	}

	updatedEvent := updated.Events[key]
	if updatedEvent.Properties["coupon_code"].ProtoNumber != removedNumber+1 {
		t.Fatalf("re-added coupon_code ProtoNumber = %d, want %d", updatedEvent.Properties["coupon_code"].ProtoNumber, removedNumber+1)
	}
	wantReserved := []ReservedField{{
		Name:        "coupon_code",
		StableID:    "coupon_code",
		ProtoNumber: removedNumber,
		Reason:      "field removed",
	}}
	if !reflect.DeepEqual(updatedEvent.Reserved, wantReserved) {
		t.Fatalf("Reserved = %#v, want %#v", updatedEvent.Reserved, wantReserved)
	}
}

func TestUpdateLockRejectsExistingProtobufReservedNumbers(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount": {Name: "amount"},
		},
	}
	key := eventKey(event)
	tests := []struct {
		name     string
		existing Lock
		reg      registry.Registry
		wantPath string
	}{
		{
			name: "context",
			existing: Lock{Context: map[string]LockedField{
				"tenant_id": {StableID: "tenant_id", ProtoNumber: 19000},
			}},
			reg:      registry.Registry{Context: map[string]registry.Field{"tenant_id": {Name: "tenant_id"}}},
			wantPath: "context.tenant_id",
		},
		{
			name: "property",
			existing: Lock{Events: map[string]LockedEvent{
				key: {Properties: map[string]LockedField{
					"amount": {StableID: "amount", ProtoNumber: 19001},
				}},
			}},
			reg:      registry.Registry{Events: []registry.Event{event}},
			wantPath: "events.checkout.completed@1.properties.amount",
		},
		{
			name: "reserved",
			existing: Lock{Events: map[string]LockedEvent{
				key: {Reserved: []ReservedField{{
					Name:        "coupon_code",
					StableID:    "coupon_code",
					ProtoNumber: 19999,
					Reason:      "field removed",
				}}},
			}},
			reg:      registry.Registry{Events: []registry.Event{event}},
			wantPath: "events.checkout.completed@1.reserved.coupon_code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdateLock(tt.existing, tt.reg)
			if err == nil {
				t.Fatalf("UpdateLock() error = nil, want protobuf reserved number error")
			}
			if !strings.Contains(err.Error(), tt.wantPath) || !strings.Contains(err.Error(), "19000..19999") {
				t.Fatalf("UpdateLock() error = %q, want path %q and reserved range", err, tt.wantPath)
			}
		})
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

func TestCheckLockRejectsMissingReservedHistoryGapAfterReadd(t *testing.T) {
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
	delete(event.Properties, "coupon_code")
	reg.Events = []registry.Event{event}
	lock, err = UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after removal error = %v", err)
	}
	event.Properties["coupon_code"] = registry.Field{Name: "coupon_code"}
	reg.Events = []registry.Event{event}
	lock, err = UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after re-add error = %v", err)
	}

	key := eventKey(event)
	lock.Events[key] = LockedEvent{
		Envelope:   lock.Events[key].Envelope,
		Properties: lock.Events[key].Properties,
		Reserved:   nil,
	}

	err = CheckLock(lock, reg)
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want missing reserved history gap error")
	}
	if !strings.Contains(err.Error(), "events.checkout.completed@1.properties") || !strings.Contains(err.Error(), "missing proto number 2") {
		t.Fatalf("CheckLock() error = %q, want property history gap", err)
	}
}

func TestCheckLockRejectsTamperedReservedProtoNumber(t *testing.T) {
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
	delete(event.Properties, "coupon_code")
	reg.Events = []registry.Event{event}
	lock, err = UpdateLock(lock, reg)
	if err != nil {
		t.Fatalf("UpdateLock() after removal error = %v", err)
	}

	key := eventKey(event)
	lock.Events[key].Reserved[0].ProtoNumber += 100

	err = CheckLock(lock, reg)
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want tampered reserved number error")
	}
	if !strings.Contains(err.Error(), "events.checkout.completed@1.properties") || !strings.Contains(err.Error(), "missing proto number 2") {
		t.Fatalf("CheckLock() error = %q, want property history gap", err)
	}
}

func TestCheckLockRejectsVersionMismatch(t *testing.T) {
	reg := registry.Registry{Context: map[string]registry.Field{"tenant_id": {Name: "tenant_id"}}}
	lock, err := UpdateLock(Lock{}, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}
	lock.Version = LockVersion + 1

	err = CheckLock(lock, reg)
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want version mismatch error")
	}
	if !strings.Contains(err.Error(), "schema lock version") || !strings.Contains(err.Error(), "want 1") {
		t.Fatalf("CheckLock() error = %q, want version mismatch", err)
	}
}

func TestCheckLockRejectsProtobufReservedNumbers(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount":      {Name: "amount"},
			"coupon_code": {Name: "coupon_code"},
		},
	}
	baseReg := registry.Registry{
		Context: map[string]registry.Field{"tenant_id": {Name: "tenant_id"}},
		Events:  []registry.Event{event},
	}
	baseLock, err := UpdateLock(Lock{}, baseReg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}

	removedEvent := event
	removedEvent.Properties = map[string]registry.Field{
		"amount": {Name: "amount"},
	}
	reservedReg := registry.Registry{Events: []registry.Event{removedEvent}}
	reservedLock, err := UpdateLock(baseLock, reservedReg)
	if err != nil {
		t.Fatalf("UpdateLock() after removal error = %v", err)
	}

	tests := []struct {
		name     string
		lock     Lock
		reg      registry.Registry
		wantPath string
	}{
		{
			name: "context",
			lock: func() Lock {
				lock := cloneLockForTest(baseLock)
				lock.Context["tenant_id"] = LockedField{StableID: "tenant_id", ProtoNumber: 19000}
				return lock
			}(),
			reg:      baseReg,
			wantPath: "context.tenant_id",
		},
		{
			name: "property",
			lock: func() Lock {
				lock := cloneLockForTest(baseLock)
				key := eventKey(event)
				lock.Events[key].Properties["amount"] = LockedField{StableID: "amount", ProtoNumber: 19001}
				return lock
			}(),
			reg:      baseReg,
			wantPath: "events.checkout.completed@1.properties.amount",
		},
		{
			name: "reserved",
			lock: func() Lock {
				lock := cloneLockForTest(reservedLock)
				key := eventKey(removedEvent)
				lock.Events[key].Reserved[0].ProtoNumber = 19999
				return lock
			}(),
			reg:      reservedReg,
			wantPath: "events.checkout.completed@1.reserved.coupon_code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckLock(tt.lock, tt.reg)
			if err == nil {
				t.Fatalf("CheckLock() error = nil, want protobuf reserved number error")
			}
			if !strings.Contains(err.Error(), tt.wantPath) || !strings.Contains(err.Error(), "19000..19999") {
				t.Fatalf("CheckLock() error = %q, want path %q and reserved range", err, tt.wantPath)
			}
		})
	}
}

func TestCheckLockRejectsTamperedActiveStableIDs(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount": {Name: "amount"},
		},
	}
	reg := registry.Registry{
		Context: map[string]registry.Field{
			"tenant_id": {Name: "tenant_id"},
		},
		Events: []registry.Event{event},
	}
	baseLock, err := UpdateLock(Lock{}, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}
	key := eventKey(event)

	tests := []struct {
		name     string
		tamper   func(Lock) Lock
		wantPath string
	}{
		{
			name: "context",
			tamper: func(lock Lock) Lock {
				field := lock.Context["tenant_id"]
				field.StableID = "evil"
				lock.Context["tenant_id"] = field
				return lock
			},
			wantPath: "context.tenant_id",
		},
		{
			name: "envelope",
			tamper: func(lock Lock) Lock {
				field := lock.Events[key].Envelope["event_name"]
				field.StableID = "evil"
				lock.Events[key].Envelope["event_name"] = field
				return lock
			},
			wantPath: "events.checkout.completed@1.envelope.event_name",
		},
		{
			name: "property",
			tamper: func(lock Lock) Lock {
				field := lock.Events[key].Properties["amount"]
				field.StableID = "evil"
				lock.Events[key].Properties["amount"] = field
				return lock
			},
			wantPath: "events.checkout.completed@1.properties.amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lock := tt.tamper(cloneLockForTest(baseLock))
			err := CheckLock(lock, reg)
			if err == nil {
				t.Fatalf("CheckLock() error = nil, want tampered StableID error")
			}
			if !strings.Contains(err.Error(), tt.wantPath) || !strings.Contains(strings.ToLower(err.Error()), "stable") {
				t.Fatalf("CheckLock() error = %q, want stable ID error at %q", err, tt.wantPath)
			}
		})
	}
}

func TestUpdateLockRejectsTamperedActiveStableIDs(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount": {Name: "amount"},
		},
	}
	reg := registry.Registry{
		Context: map[string]registry.Field{
			"tenant_id": {Name: "tenant_id"},
		},
		Events: []registry.Event{event},
	}
	baseLock, err := UpdateLock(Lock{}, reg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}
	key := eventKey(event)

	tests := []struct {
		name     string
		tamper   func(Lock) Lock
		wantPath string
	}{
		{
			name: "context",
			tamper: func(lock Lock) Lock {
				field := lock.Context["tenant_id"]
				field.StableID = "evil"
				lock.Context["tenant_id"] = field
				return lock
			},
			wantPath: "context.tenant_id",
		},
		{
			name: "envelope",
			tamper: func(lock Lock) Lock {
				field := lock.Events[key].Envelope["event_name"]
				field.StableID = "evil"
				lock.Events[key].Envelope["event_name"] = field
				return lock
			},
			wantPath: "events.checkout.completed@1.envelope.event_name",
		},
		{
			name: "property",
			tamper: func(lock Lock) Lock {
				field := lock.Events[key].Properties["amount"]
				field.StableID = "evil"
				lock.Events[key].Properties["amount"] = field
				return lock
			},
			wantPath: "events.checkout.completed@1.properties.amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lock := tt.tamper(cloneLockForTest(baseLock))
			_, err := UpdateLock(lock, reg)
			if err == nil {
				t.Fatalf("UpdateLock() error = nil, want tampered StableID error")
			}
			if !strings.Contains(err.Error(), tt.wantPath) || !strings.Contains(strings.ToLower(err.Error()), "stable") {
				t.Fatalf("UpdateLock() error = %q, want stable ID error at %q", err, tt.wantPath)
			}
		})
	}
}

func TestUpdateLockRejectsDuplicateExistingNumbers(t *testing.T) {
	event := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount":   {Name: "amount"},
			"order_id": {Name: "order_id"},
		},
	}
	key := eventKey(event)

	tests := []struct {
		name     string
		existing Lock
		reg      registry.Registry
		wantPath string
	}{
		{
			name: "context active active",
			existing: Lock{Context: map[string]LockedField{
				"tenant_id": {StableID: "tenant_id", ProtoNumber: 1},
				"region":    {StableID: "region", ProtoNumber: 1},
			}},
			reg: registry.Registry{Context: map[string]registry.Field{
				"tenant_id": {Name: "tenant_id"},
				"region":    {Name: "region"},
			}},
			wantPath: "context",
		},
		{
			name: "property active active",
			existing: Lock{Events: map[string]LockedEvent{
				key: {Properties: map[string]LockedField{
					"amount":   {StableID: "amount", ProtoNumber: 1},
					"order_id": {StableID: "order_id", ProtoNumber: 1},
				}},
			}},
			reg:      registry.Registry{Events: []registry.Event{event}},
			wantPath: "events.checkout.completed@1.properties",
		},
		{
			name: "property active reserved",
			existing: Lock{Events: map[string]LockedEvent{
				key: {
					Properties: map[string]LockedField{
						"amount": {StableID: "amount", ProtoNumber: 1},
					},
					Reserved: []ReservedField{{
						Name:        "coupon_code",
						StableID:    "coupon_code",
						ProtoNumber: 1,
						Reason:      "field removed",
					}},
				},
			}},
			reg: registry.Registry{Events: []registry.Event{{
				Name:    event.Name,
				Version: event.Version,
				Properties: map[string]registry.Field{
					"amount": {Name: "amount"},
				},
			}}},
			wantPath: "events.checkout.completed@1.properties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdateLock(tt.existing, tt.reg)
			if err == nil {
				t.Fatalf("UpdateLock() error = nil, want duplicate number error")
			}
			if !strings.Contains(err.Error(), tt.wantPath) || !strings.Contains(err.Error(), "duplicate") {
				t.Fatalf("UpdateLock() error = %q, want duplicate error at %q", err, tt.wantPath)
			}
		})
	}
}

func TestCheckLockDoesNotMutateValidLockWithNilReserved(t *testing.T) {
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
					"amount": {StableID: "amount", ProtoNumber: 1},
				},
				Reserved: nil,
			},
		},
	}
	before := cloneLockForTest(lock)

	if err := CheckLock(lock, reg); err != nil {
		t.Fatalf("CheckLock() error = %v, want nil", err)
	}
	if lock.Events[eventKey(event)].Reserved != nil {
		t.Fatalf("CheckLock() changed Reserved from nil to %#v", lock.Events[eventKey(event)].Reserved)
	}
	if !reflect.DeepEqual(lock, before) {
		t.Fatalf("CheckLock() mutated lock: got %#v want %#v", lock, before)
	}
}

func TestCheckLockRejectsUnsortedReservedEntries(t *testing.T) {
	baseEvent := registry.Event{
		Name:    "checkout.completed",
		Version: 1,
		Properties: map[string]registry.Field{
			"amount":      {Name: "amount"},
			"coupon_code": {Name: "coupon_code"},
			"discount":    {Name: "discount"},
		},
	}
	baseReg := registry.Registry{Events: []registry.Event{baseEvent}}
	lock, err := UpdateLock(Lock{}, baseReg)
	if err != nil {
		t.Fatalf("UpdateLock() error = %v", err)
	}

	removedEvent := registry.Event{
		Name:    baseEvent.Name,
		Version: baseEvent.Version,
		Properties: map[string]registry.Field{
			"amount": {Name: "amount"},
		},
	}
	removedReg := registry.Registry{Events: []registry.Event{removedEvent}}
	lock, err = UpdateLock(lock, removedReg)
	if err != nil {
		t.Fatalf("UpdateLock() after removal error = %v", err)
	}

	key := eventKey(removedEvent)
	reserved := lock.Events[key].Reserved
	if len(reserved) != 2 {
		t.Fatalf("Reserved length = %d, want 2", len(reserved))
	}
	lock.Events[key] = LockedEvent{
		Envelope:   lock.Events[key].Envelope,
		Properties: lock.Events[key].Properties,
		Reserved:   []ReservedField{reserved[1], reserved[0]},
	}

	err = CheckLock(lock, removedReg)
	if err == nil {
		t.Fatalf("CheckLock() error = nil, want reserved ordering error")
	}
	if !strings.Contains(err.Error(), "events.checkout.completed@1.reserved") || !strings.Contains(err.Error(), "order") {
		t.Fatalf("CheckLock() error = %q, want reserved order error", err)
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
