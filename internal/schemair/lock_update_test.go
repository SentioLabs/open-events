package schemair

import (
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
