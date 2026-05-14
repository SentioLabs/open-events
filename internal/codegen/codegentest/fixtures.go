// Package codegentest provides shared test fixtures for the per-language
// emitter packages under internal/codegen/. It exists to keep the Go and
// Python emitter test files from duplicating identical 2-domain scaffolds.
package codegentest

import (
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
	"github.com/sentiolabs/open-events/internal/registry/testfx"
	"github.com/sentiolabs/open-events/internal/schemair"
)

// TwoDomainOptions tunes the optional fields of TwoDomainRegistry.
type TwoDomainOptions struct {
	// IncludeUserID adds an optional UUID user_id field to the user domain
	// context (with ProtoNumber 2 in the returned lock). The Python emitter
	// uses this to exercise the Optional[...] code path; the Go emitter
	// doesn't need it.
	IncludeUserID bool
}

// TwoDomainRegistry constructs a small 2-domain (user + device) registry on
// disk via testfx, loads it, and returns both the loaded registry and a
// minimal valid Lock covering the contexts and events.
func TwoDomainRegistry(t *testing.T, opts TwoDomainOptions) (registry.Registry, schemair.Lock) {
	t.Helper()

	userDomain := testfx.New().
		Namespace("com.acme.platform").
		Package("github.com/acme/platform/events", "acme.events").
		Domain("user").
		Description("user domain").
		Owner("growth").
		Context("tenant_id", registry.FieldTypeString, true, registry.PIINone)

	if opts.IncludeUserID {
		userDomain = userDomain.Context("user_id", registry.FieldTypeUUID, false, registry.PIIPseudonymous)
	}

	root := userDomain.
		Action([]string{"auth"}, "signup").
		Version(1).
		Status("active").
		Property("method", registry.FieldTypeString, true, registry.PIINone).
		Done().
		Done().
		Domain("device").
		Description("device domain").
		Owner("platform").
		Context("device_id", registry.FieldTypeString, true, registry.PIINone).
		Action([]string{"info"}, "hardware").
		Version(1).
		Status("active").
		Property("os", registry.FieldTypeString, true, registry.PIINone).
		Done().
		Done().
		Write(t)

	reg, diags := registry.Load(root)
	if diags.HasErrors() {
		t.Fatalf("registry.Load() diagnostics: %v", diags)
	}

	userContext := map[string]schemair.LockedField{
		"tenant_id": {StableID: "tenant_id", ProtoNumber: 1},
	}
	if opts.IncludeUserID {
		userContext["user_id"] = schemair.LockedField{StableID: "user_id", ProtoNumber: 2}
	}

	lock := schemair.Lock{
		Version: schemair.LockVersion,
		Domains: map[string]schemair.LockedDomain{
			"user":   {Context: userContext},
			"device": {Context: map[string]schemair.LockedField{"device_id": {StableID: "device_id", ProtoNumber: 1}}},
		},
		Events: map[string]schemair.LockedEvent{
			"user.auth.signup@1": {
				Properties: map[string]schemair.LockedField{"method": {StableID: "method", ProtoNumber: 1}},
			},
			"device.info.hardware@1": {
				Properties: map[string]schemair.LockedField{"os": {StableID: "os", ProtoNumber: 1}},
			},
		},
	}

	return reg, lock
}
