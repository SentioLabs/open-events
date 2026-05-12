package schemair

import (
	"fmt"
	"sort"

	"github.com/sentiolabs/open-events/internal/registry"
)

const LockVersion = 1

var envelopeNumbers = map[string]int{
	"event_name":    1,
	"event_version": 2,
	"event_id":      3,
	"event_ts":      4,
	"client":        5,
	"context":       6,
	"properties":    7,
}

func UpdateLock(existing Lock, reg registry.Registry) (Lock, error) {
	updated := Lock{
		Version: LockVersion,
		Context: make(map[string]LockedField, len(reg.Context)),
		Events:  make(map[string]LockedEvent, len(reg.Events)),
	}

	contextMax := 0
	for _, name := range sortedLockedFieldNames(existing.Context) {
		locked := existing.Context[name]
		updated.Context[name] = locked
		if locked.ProtoNumber > contextMax {
			contextMax = locked.ProtoNumber
		}
	}

	for _, name := range sortedFieldNames(reg.Context) {
		if _, ok := updated.Context[name]; ok {
			continue
		}
		number := nextSequentialNumber(contextMax)
		contextMax = number
		updated.Context[name] = LockedField{StableID: name, ProtoNumber: number}
	}

	seenEventKeys := make(map[string]struct{}, len(reg.Events))
	for _, event := range reg.Events {
		key := eventKey(event)
		if _, exists := seenEventKeys[key]; exists {
			return Lock{}, fmt.Errorf("schema registry has duplicate event key %s", key)
		}
		seenEventKeys[key] = struct{}{}

		existingEvent := existing.Events[key]
		updatedEvent := LockedEvent{
			Envelope:   make(map[string]LockedField, len(envelopeNumbers)),
			Properties: make(map[string]LockedField, len(event.Properties)),
			Reserved:   make([]ReservedField, 0, len(existingEvent.Reserved)),
		}

		for _, name := range sortedEnvelopeFieldNames() {
			number := envelopeNumbers[name]
			stableID := name
			if locked, ok := existingEvent.Envelope[name]; ok && locked.StableID != "" {
				stableID = locked.StableID
			}
			updatedEvent.Envelope[name] = LockedField{StableID: stableID, ProtoNumber: number}
		}

		propertiesMax := 0
		for _, locked := range existingEvent.Properties {
			if locked.ProtoNumber > propertiesMax {
				propertiesMax = locked.ProtoNumber
			}
		}
		for _, reserved := range existingEvent.Reserved {
			if reserved.ProtoNumber > propertiesMax {
				propertiesMax = reserved.ProtoNumber
			}
		}

		for _, name := range sortedFieldNames(event.Properties) {
			if locked, ok := existingEvent.Properties[name]; ok {
				updatedEvent.Properties[name] = locked
				continue
			}
			number := nextSequentialNumber(propertiesMax)
			propertiesMax = number
			updatedEvent.Properties[name] = LockedField{StableID: name, ProtoNumber: number}
		}

		for _, reserved := range existingEvent.Reserved {
			if _, exists := updatedEvent.Properties[reserved.Name]; exists {
				continue
			}
			updatedEvent.Reserved = append(updatedEvent.Reserved, reserved)
		}

		for _, name := range sortedLockedFieldNames(existingEvent.Properties) {
			if _, ok := event.Properties[name]; ok {
				continue
			}
			locked := existingEvent.Properties[name]
			updatedEvent.Reserved = append(updatedEvent.Reserved, ReservedField{
				Name:        name,
				StableID:    locked.StableID,
				ProtoNumber: locked.ProtoNumber,
				Reason:      "field removed",
			})
		}
		sort.Slice(updatedEvent.Reserved, func(i, j int) bool {
			return lessReservedField(updatedEvent.Reserved[i], updatedEvent.Reserved[j])
		})

		updated.Events[key] = updatedEvent
	}

	return updated, nil
}

func CheckLock(lock Lock, reg registry.Registry) error {
	if err := validateLockDuplicates(lock); err != nil {
		return err
	}

	expected, err := UpdateLock(lock, reg)
	if err != nil {
		return err
	}

	for _, name := range sortedLockedFieldNames(expected.Context) {
		exp := expected.Context[name]
		actual, ok := lock.Context[name]
		if !ok {
			return fmt.Errorf("schema lock is stale: context.%s is missing", name)
		}
		if actual.ProtoNumber != exp.ProtoNumber {
			return fmt.Errorf("schema lock is stale: context.%s number mismatch: got %d want %d", name, actual.ProtoNumber, exp.ProtoNumber)
		}
	}

	for _, key := range sortedLockedEventKeys(lock.Events) {
		if _, ok := expected.Events[key]; !ok {
			return fmt.Errorf("schema lock is stale: events.%s is not in registry", key)
		}
	}

	for _, event := range reg.Events {
		key := eventKey(event)
		expectedEvent, ok := expected.Events[key]
		if !ok {
			return fmt.Errorf("schema lock is stale: events.%s is missing", key)
		}
		actualEvent, ok := lock.Events[key]
		if !ok {
			return fmt.Errorf("schema lock is stale: events.%s is missing", key)
		}
		for _, name := range sortedLockedFieldNames(expectedEvent.Envelope) {
			exp := expectedEvent.Envelope[name]
			actual, ok := actualEvent.Envelope[name]
			if !ok {
				return fmt.Errorf("schema lock is stale: events.%s.envelope.%s is missing", key, name)
			}
			if actual.ProtoNumber != exp.ProtoNumber {
				return fmt.Errorf("schema lock is stale: events.%s.envelope.%s number mismatch: got %d want %d", key, name, actual.ProtoNumber, exp.ProtoNumber)
			}
		}
		for _, name := range sortedLockedFieldNames(actualEvent.Envelope) {
			if _, ok := expectedEvent.Envelope[name]; !ok {
				return fmt.Errorf("schema lock is stale: events.%s.envelope.%s is not in registry", key, name)
			}
		}
		for _, name := range sortedLockedFieldNames(expectedEvent.Properties) {
			exp := expectedEvent.Properties[name]
			actual, ok := actualEvent.Properties[name]
			if !ok {
				return fmt.Errorf("schema lock is stale: events.%s.properties.%s is missing", key, name)
			}
			if actual.ProtoNumber != exp.ProtoNumber {
				return fmt.Errorf("schema lock is stale: events.%s.properties.%s number mismatch: got %d want %d", key, name, actual.ProtoNumber, exp.ProtoNumber)
			}
		}
		if err := compareReservedFields(key, actualEvent.Reserved, expectedEvent.Reserved); err != nil {
			return err
		}
		for _, name := range sortedLockedFieldNames(actualEvent.Properties) {
			if _, ok := expectedEvent.Properties[name]; !ok {
				return fmt.Errorf("schema lock is stale: events.%s.properties.%s should be reserved", key, name)
			}
		}
	}

	return nil
}

func eventKey(event registry.Event) string {
	return fmt.Sprintf("%s@%d", event.Name, event.Version)
}

func sortedEnvelopeFieldNames() []string {
	names := make([]string, 0, len(envelopeNumbers))
	for name := range envelopeNumbers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedFieldNames(fields map[string]registry.Field) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedLockedFieldNames(fields map[string]LockedField) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedLockedEventKeys(events map[string]LockedEvent) []string {
	keys := make([]string, 0, len(events))
	for key := range events {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func compareReservedFields(eventKey string, actual []ReservedField, expected []ReservedField) error {
	path := "events." + eventKey + ".reserved"
	if err := checkDuplicateReservedNumbers(path, actual); err != nil {
		return err
	}
	if err := checkDuplicateReservedNumbers(path, expected); err != nil {
		return err
	}

	actualByKey := make(map[reservedFieldKey]ReservedField, len(actual))
	for _, reserved := range actual {
		actualByKey[reservedFieldKey{protoNumber: reserved.ProtoNumber, name: reserved.Name}] = reserved
	}
	expectedByKey := make(map[reservedFieldKey]ReservedField, len(expected))
	for _, reserved := range expected {
		expectedByKey[reservedFieldKey{protoNumber: reserved.ProtoNumber, name: reserved.Name}] = reserved
	}

	for _, exp := range sortedReservedFields(expected) {
		key := reservedFieldKey{protoNumber: exp.ProtoNumber, name: exp.Name}
		actual, ok := actualByKey[key]
		if !ok {
			return fmt.Errorf("schema lock is stale: events.%s.reserved.%s is missing", eventKey, exp.Name)
		}
		if actual != exp {
			return fmt.Errorf("schema lock is stale: events.%s.reserved.%s mismatch", eventKey, exp.Name)
		}
	}
	for _, actual := range sortedReservedFields(actual) {
		key := reservedFieldKey{protoNumber: actual.ProtoNumber, name: actual.Name}
		if _, ok := expectedByKey[key]; !ok {
			return fmt.Errorf("schema lock is stale: events.%s.reserved.%s is not expected", eventKey, actual.Name)
		}
	}

	return nil
}

type reservedFieldKey struct {
	protoNumber int
	name        string
}

func sortedReservedFields(fields []ReservedField) []ReservedField {
	sorted := append([]ReservedField(nil), fields...)
	sort.Slice(sorted, func(i, j int) bool {
		return lessReservedField(sorted[i], sorted[j])
	})
	return sorted
}

func lessReservedField(left ReservedField, right ReservedField) bool {
	if left.ProtoNumber != right.ProtoNumber {
		return left.ProtoNumber < right.ProtoNumber
	}
	if left.Name != right.Name {
		return left.Name < right.Name
	}
	if left.StableID != right.StableID {
		return left.StableID < right.StableID
	}
	return left.Reason < right.Reason
}

func validateLockDuplicates(lock Lock) error {
	if err := checkDuplicateNumbers("context", lock.Context, nil); err != nil {
		return err
	}

	for _, key := range sortedLockedEventKeys(lock.Events) {
		event := lock.Events[key]
		if err := checkDuplicateNumbers("events."+key+".envelope", event.Envelope, nil); err != nil {
			return err
		}
		if err := checkDuplicateNumbers("events."+key+".properties", event.Properties, event.Reserved); err != nil {
			return err
		}
	}

	return nil
}

func checkDuplicateNumbers(path string, fields map[string]LockedField, reserved []ReservedField) error {
	byNumber := map[int]string{}
	for _, name := range sortedLockedFieldNames(fields) {
		field := fields[name]
		if prior, exists := byNumber[field.ProtoNumber]; exists {
			return fmt.Errorf("schema lock has duplicate proto numbers in %s: %s and %s share %d", path, prior, name, field.ProtoNumber)
		}
		byNumber[field.ProtoNumber] = name
	}
	for _, item := range sortedReservedFields(reserved) {
		if prior, exists := byNumber[item.ProtoNumber]; exists {
			return fmt.Errorf("schema lock has duplicate proto numbers in %s: %s and %s share %d", path, prior, item.Name, item.ProtoNumber)
		}
		byNumber[item.ProtoNumber] = item.Name
	}
	return nil
}

func checkDuplicateReservedNumbers(path string, reserved []ReservedField) error {
	byNumber := map[int]string{}
	for _, item := range sortedReservedFields(reserved) {
		if prior, exists := byNumber[item.ProtoNumber]; exists {
			return fmt.Errorf("schema lock has duplicate proto numbers in %s: %s and %s share %d", path, prior, item.Name, item.ProtoNumber)
		}
		byNumber[item.ProtoNumber] = item.Name
	}
	return nil
}

func nextSequentialNumber(maxUsed int) int {
	n := maxUsed + 1
	if n >= 19000 && n <= 19999 {
		return 20000
	}
	return n
}
