package schemair

import (
	"fmt"
	"sort"

	"github.com/sentiolabs/open-events/internal/registry"
)

const LockVersion = 1

const reservedFieldReasonRemoved = "field removed"

const (
	protobufReservedStart = 19000
	protobufReservedEnd   = 19999
)

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
	if err := validateLockNumbers(existing); err != nil {
		return Lock{}, err
	}
	if err := validateActiveStableIDs(existing, reg); err != nil {
		return Lock{}, err
	}
	if err := validateReservedFieldIdentities(existing, reg); err != nil {
		return Lock{}, err
	}
	if err := validateLockDuplicates(existing); err != nil {
		return Lock{}, err
	}
	if err := validateLockNumberHistory(existing); err != nil {
		return Lock{}, err
	}

	updated := Lock{
		Version: LockVersion,
		Domains: make(map[string]LockedDomain, len(reg.Domains)),
		// Start from a copy of existing.Events so tombstones (events present in
		// the lock but absent from the registry) are preserved verbatim.
		Events: copyLockedEvents(existing.Events),
	}

	// Allocate per-domain context proto numbers, preserving existing assignments.
	for _, domainName := range sortedDomainKeys(reg.Domains) {
		domain := reg.Domains[domainName]
		existingDomain := existing.Domains[domainName] // zero-value if absent

		contextMax := 0
		for _, name := range sortedLockedFieldNames(existingDomain.Context) {
			locked := existingDomain.Context[name]
			if locked.ProtoNumber > contextMax {
				contextMax = locked.ProtoNumber
			}
		}
		for _, reserved := range existingDomain.Reserved {
			if reserved.ProtoNumber > contextMax {
				contextMax = reserved.ProtoNumber
			}
		}

		// Build the active context: only fields still in the registry.
		lockedCtx := make(map[string]LockedField, len(domain.Context))
		for _, name := range sortedFieldNames(domain.Context) {
			if locked, ok := existingDomain.Context[name]; ok {
				// Preserve existing assignment.
				lockedCtx[name] = locked
			} else {
				// Allocate a new proto number for new context fields.
				number := nextSequentialNumber(contextMax)
				contextMax = number
				lockedCtx[name] = LockedField{StableID: name, ProtoNumber: number}
			}
		}

		// Move removed context fields into Reserved (tombstones).
		domainReserved := append([]ReservedField(nil), existingDomain.Reserved...)
		for _, name := range sortedLockedFieldNames(existingDomain.Context) {
			if _, ok := domain.Context[name]; ok {
				continue // still active
			}
			locked := existingDomain.Context[name]
			domainReserved = append(domainReserved, ReservedField{
				Name:        name,
				StableID:    locked.StableID,
				ProtoNumber: locked.ProtoNumber,
				Reason:      reservedFieldReasonRemoved,
			})
		}
		sort.Slice(domainReserved, func(i, j int) bool {
			return lessReservedField(domainReserved[i], domainReserved[j])
		})

		updated.Domains[domainName] = LockedDomain{Context: lockedCtx, Reserved: domainReserved}
	}

	// Update or add events from the registry. Events already in updated.Events
	// that are absent from reg.Events remain as tombstones (no action needed).
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
			updatedEvent.Envelope[name] = LockedField{StableID: name, ProtoNumber: envelopeNumbers[name]}
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

		updatedEvent.Reserved = append(updatedEvent.Reserved, existingEvent.Reserved...)

		for _, name := range sortedLockedFieldNames(existingEvent.Properties) {
			if _, ok := event.Properties[name]; ok {
				continue
			}
			locked := existingEvent.Properties[name]
			updatedEvent.Reserved = append(updatedEvent.Reserved, ReservedField{
				Name:        name,
				StableID:    locked.StableID,
				ProtoNumber: locked.ProtoNumber,
				Reason:      reservedFieldReasonRemoved,
			})
		}
		sort.Slice(updatedEvent.Reserved, func(i, j int) bool {
			return lessReservedField(updatedEvent.Reserved[i], updatedEvent.Reserved[j])
		})

		updated.Events[key] = updatedEvent
	}

	return updated, nil
}

// copyLockedEvents returns a shallow copy of the events map. This is the
// starting point for UpdateLock so that events not present in reg.Events
// (tombstones) are preserved verbatim in the output.
func copyLockedEvents(events map[string]LockedEvent) map[string]LockedEvent {
	if events == nil {
		return make(map[string]LockedEvent)
	}
	cp := make(map[string]LockedEvent, len(events))
	for k, v := range events {
		cp[k] = v
	}
	return cp
}

func CheckLock(lock Lock, reg registry.Registry) error {
	if lock.Version != LockVersion {
		return fmt.Errorf("schema lock version mismatch: got %d want %d", lock.Version, LockVersion)
	}
	if err := validateLockNumbers(lock); err != nil {
		return err
	}
	if err := validateActiveStableIDs(lock, reg); err != nil {
		return err
	}
	if err := validateReservedFieldIdentities(lock, reg); err != nil {
		return err
	}
	if err := validateLockDuplicates(lock); err != nil {
		return err
	}
	if err := validateLockNumberHistory(lock); err != nil {
		return err
	}

	expected, err := UpdateLock(lock, reg)
	if err != nil {
		return err
	}

	// Verify all registry domains have matching lock domain entries.
	for _, domainName := range sortedLockedDomainKeys(expected.Domains) {
		expDomain := expected.Domains[domainName]
		actualDomain, ok := lock.Domains[domainName]
		if !ok {
			return fmt.Errorf("schema lock is stale: domains.%s is missing", domainName)
		}
		for _, name := range sortedLockedFieldNames(expDomain.Context) {
			exp := expDomain.Context[name]
			actual, ok := actualDomain.Context[name]
			if !ok {
				return fmt.Errorf("schema lock is stale: domains.%s.context.%s is missing", domainName, name)
			}
			if err := compareLockedField("domains."+domainName+".context."+name, actual, exp); err != nil {
				return err
			}
		}
		// Check domain reserved fields match.
		if err := compareReservedFields("domains."+domainName, actualDomain.Reserved, expDomain.Reserved); err != nil {
			return err
		}
	}

	// Check events: registry events must all be in the lock; lock-only entries
	// (tombstones) are allowed and do not cause an error.
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
			if err := compareLockedField("events."+key+".envelope."+name, actual, exp); err != nil {
				return err
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
			if err := compareLockedField("events."+key+".properties."+name, actual, exp); err != nil {
				return err
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

func compareLockedField(path string, actual LockedField, expected LockedField) error {
	if actual.StableID != expected.StableID {
		return fmt.Errorf("schema lock is stale: %s stable ID mismatch: got %q want %q", path, actual.StableID, expected.StableID)
	}
	if actual.ProtoNumber != expected.ProtoNumber {
		return fmt.Errorf("schema lock is stale: %s number mismatch: got %d want %d", path, actual.ProtoNumber, expected.ProtoNumber)
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

func sortedDomainKeys(domains map[string]registry.Domain) []string {
	keys := make([]string, 0, len(domains))
	for k := range domains {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedLockedDomainKeys(domains map[string]LockedDomain) []string {
	keys := make([]string, 0, len(domains))
	for k := range domains {
		keys = append(keys, k)
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
	if !equalReservedFields(actual, sortedReservedFields(actual)) {
		return fmt.Errorf("schema lock is stale: %s order mismatch", path)
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

func equalReservedFields(left []ReservedField, right []ReservedField) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func validateActiveStableIDs(lock Lock, reg registry.Registry) error {
	for _, domainName := range sortedLockedDomainKeys(lock.Domains) {
		domain := lock.Domains[domainName]
		for _, name := range sortedLockedFieldNames(domain.Context) {
			if err := validateStableID("domains."+domainName+".context."+name, domain.Context[name].StableID, name); err != nil {
				return err
			}
		}
	}

	for _, event := range reg.Events {
		key := eventKey(event)
		existingEvent, ok := lock.Events[key]
		if !ok {
			continue
		}
		for _, name := range sortedEnvelopeFieldNames() {
			locked, ok := existingEvent.Envelope[name]
			if !ok {
				continue
			}
			if err := validateStableID("events."+key+".envelope."+name, locked.StableID, name); err != nil {
				return err
			}
		}
		for _, name := range sortedLockedFieldNames(existingEvent.Properties) {
			locked := existingEvent.Properties[name]
			if err := validateStableID("events."+key+".properties."+name, locked.StableID, name); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateStableID(path string, actual string, expected string) error {
	if actual != expected {
		return fmt.Errorf("schema lock has invalid stable ID at %s: got %q want %q", path, actual, expected)
	}
	return nil
}

func validateReservedFieldIdentities(lock Lock, reg registry.Registry) error {
	for _, event := range reg.Events {
		key := eventKey(event)
		lockedEvent, ok := lock.Events[key]
		if !ok {
			continue
		}
		for _, reserved := range sortedReservedFields(lockedEvent.Reserved) {
			path := "events." + key + ".reserved"
			if reserved.Name == "" {
				return fmt.Errorf("schema lock has invalid reserved field at %s: name must be non-empty", path)
			}
			fieldPath := path + "." + reserved.Name
			if err := validateStableID(fieldPath, reserved.StableID, reserved.Name); err != nil {
				return err
			}
			if reserved.Reason != reservedFieldReasonRemoved {
				return fmt.Errorf("schema lock has invalid reserved reason at %s: got %q want %q", fieldPath, reserved.Reason, reservedFieldReasonRemoved)
			}
		}
	}
	return nil
}

func validateLockNumbers(lock Lock) error {
	for _, domainName := range sortedLockedDomainKeys(lock.Domains) {
		domain := lock.Domains[domainName]
		for _, name := range sortedLockedFieldNames(domain.Context) {
			if err := validateProtoNumber("domains."+domainName+".context."+name, domain.Context[name].ProtoNumber); err != nil {
				return err
			}
		}
		for _, reserved := range sortedReservedFields(domain.Reserved) {
			if err := validateProtoNumber("domains."+domainName+".reserved."+reserved.Name, reserved.ProtoNumber); err != nil {
				return err
			}
		}
	}

	for _, key := range sortedLockedEventKeys(lock.Events) {
		event := lock.Events[key]
		for _, name := range sortedLockedFieldNames(event.Envelope) {
			if err := validateProtoNumber("events."+key+".envelope."+name, event.Envelope[name].ProtoNumber); err != nil {
				return err
			}
		}
		for _, name := range sortedLockedFieldNames(event.Properties) {
			if err := validateProtoNumber("events."+key+".properties."+name, event.Properties[name].ProtoNumber); err != nil {
				return err
			}
		}
		for _, reserved := range sortedReservedFields(event.Reserved) {
			if err := validateProtoNumber("events."+key+".reserved."+reserved.Name, reserved.ProtoNumber); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateProtoNumber(path string, number int) error {
	if number < 1 {
		return fmt.Errorf("schema lock has invalid proto number at %s: got %d, want >= 1", path, number)
	}
	if isProtobufReservedNumber(number) {
		return fmt.Errorf("schema lock has invalid proto number at %s: %d is in protobuf reserved range 19000..19999", path, number)
	}
	return nil
}

func validateLockNumberHistory(lock Lock) error {
	for _, domainName := range sortedLockedDomainKeys(lock.Domains) {
		domain := lock.Domains[domainName]
		numbers := make([]int, 0, len(domain.Context)+len(domain.Reserved))
		for _, name := range sortedLockedFieldNames(domain.Context) {
			numbers = append(numbers, domain.Context[name].ProtoNumber)
		}
		for _, reserved := range sortedReservedFields(domain.Reserved) {
			numbers = append(numbers, reserved.ProtoNumber)
		}
		if err := checkDenseNumberHistory("domains."+domainName+".context", numbers); err != nil {
			return err
		}
	}

	for _, key := range sortedLockedEventKeys(lock.Events) {
		event := lock.Events[key]
		numbers := make([]int, 0, len(event.Properties)+len(event.Reserved))
		for _, name := range sortedLockedFieldNames(event.Properties) {
			numbers = append(numbers, event.Properties[name].ProtoNumber)
		}
		for _, reserved := range sortedReservedFields(event.Reserved) {
			numbers = append(numbers, reserved.ProtoNumber)
		}
		if err := checkDenseNumberHistory("events."+key+".properties", numbers); err != nil {
			return err
		}
	}

	return nil
}

func checkDenseNumberHistory(path string, numbers []int) error {
	if len(numbers) == 0 {
		return nil
	}

	sorted := append([]int(nil), numbers...)
	sort.Ints(sorted)
	expected := 1
	for _, number := range sorted {
		if number < expected {
			continue
		}
		if number > expected {
			return fmt.Errorf("schema lock is stale: %s is missing proto number %d before %d", path, expected, number)
		}
		expected = nextExpectedProtoNumber(expected)
	}

	return nil
}

func nextExpectedProtoNumber(number int) int {
	next := number + 1
	if isProtobufReservedNumber(next) {
		return protobufReservedEnd + 1
	}
	return next
}

func isProtobufReservedNumber(number int) bool {
	return number >= protobufReservedStart && number <= protobufReservedEnd
}

func validateLockDuplicates(lock Lock) error {
	for _, domainName := range sortedLockedDomainKeys(lock.Domains) {
		domain := lock.Domains[domainName]
		if err := checkDuplicateNumbers("domains."+domainName+".context", domain.Context, domain.Reserved); err != nil {
			return err
		}
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
	if isProtobufReservedNumber(n) {
		return protobufReservedEnd + 1
	}
	return n
}
