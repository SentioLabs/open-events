<!-- arc-review: kind=legacy id=plan.095lgp -->
# OpenEvents — MVP Design and Milestone 1 Front-End Plan

Status: Draft v0.2  
Audience: Codex / implementation agents  
Primary implementation language: Go  
Planning stance: Full MVP at a high level; Milestone 1 specified in detail.

## 1. Purpose

OpenEvents is a Git-first, vendor-neutral event taxonomy compiler for analytics, product, and data pipeline events.

The project turns a YAML event registry into typed and reviewable artifacts used across a Go producer backend, queue JSON, Python/Dagster consumers, and Snowflake warehouse tables.

```text
OpenEvents YAML registry
        ↓
Go CLI compiler front-end
        ↓
normalized registry model
        ↓
validation, snapshot, diff, and generators
        ↓
Go producer types | Python consumer models | JSON Schema | Snowflake DDL | Markdown docs
```

The immediate implementation goal is **Milestone 1: Parse and Validate Registry**. Milestone 1 should not implement code generation, snapshots, or compatibility diffing, but it must establish the compiler front-end that those later milestones reuse.

## 2. Problem Statement

The current analytics/data event contract is implicit and spread across mobile payloads, Go structs, queue JSON, Python parsing, Dagster jobs, and Snowflake tables.

This causes several failure modes:

- Producer and consumer code drift.
- Event names, versions, fields, allowed values, and ownership are not governed in one place.
- PII classification is ad hoc.
- Breaking changes are easy to introduce accidentally.
- Warehouse shape is inferred downstream instead of specified at the contract boundary.

OpenEvents treats analytics events as a typed public API. The YAML registry is the source of truth, and every generated artifact should derive from the same normalized model.

## 3. MVP Scope

### In scope

- YAML registry format for event taxonomy.
- Deterministic parser and validator.
- Normalized internal registry model.
- Snapshot and breaking-change diff against a previous registry snapshot.
- Go producer code generation.
- Python Pydantic consumer code generation.
- JSON Schema export per event version.
- Snowflake DDL export.
- Markdown event catalog generation.
- CI-friendly commands and deterministic output.

### Out of scope for the MVP

- Event broker implementation.
- Queue-specific emitters for Kafka, SQS, NATS, Pub/Sub, or RabbitMQ.
- Hosted governance UI.
- Runtime analytics dashboards.
- Full lineage tracking.
- Mobile SDK generation.
- AsyncAPI, dbt, Dagster, OpenTelemetry, Segment, Amplitude, RudderStack, Snowplow, Avro, or Protobuf export.

Those integrations can be added later, but the MVP should stay focused on making the event contract explicit, typed, versioned, and shared.

## 4. Core MVP Concepts

### Registry

A registry is one file or a directory of `.yaml` / `.yml` files that merge into one OpenEvents model.

```text
events/
  openevents.yaml
  events/
    user.yaml
    billing.yaml
    search.yaml
```

The registry is the compiler input. Later milestones must consume the normalized model from Milestone 1 rather than reparsing YAML directly.

### Event

An event is identified by `event_name + event_version`.

Example names:

```text
user.signed_up
user.logged_in
search.query_submitted
billing.invoice_paid
mobile.screen_viewed
```

Event names are lowercase, dot-separated, and stable. Breaking changes create a new version.

### Envelope

The queue payload uses a shared envelope plus event-specific properties.

```json
{
  "event_name": "user.signed_up",
  "event_version": 1,
  "event_id": "018f48cf-3cb2-7e8e-bc88-2fd84b48e1f2",
  "event_ts": "2026-05-11T16:30:00Z",
  "tenant_id": "tenant_123",
  "client": {
    "source": "ios",
    "version": "5.56.0",
    "sdk": "openevents-go"
  },
  "context": {
    "user_id": "user_123",
    "session_id": "session_456",
    "platform": "ios"
  },
  "properties": {
    "signup_method": "apple",
    "plan": "pro"
  }
}
```

`event_id` is the canonical OpenEvents envelope identifier. `event_ts` is the single canonical timestamp for when the event was generated. Client metadata is nested under `client`; `client.source` records the originating platform or service, `client.version` records the app/web/service version, and `client.sdk` records the OpenEvents library identifier used by the emitter. Common `context` stays separate from event-specific `properties` so shared fields remain governed consistently.

Event definitions can still include `producer` and `sources` as taxonomy/governance metadata. The emitted JSON representation should use the nested `client` object for runtime origin metadata.

### Context fields

Context fields are common across events and should support required/optional semantics and PII metadata.

Examples:

- `tenant_id`
- `user_id`
- `anonymous_id`
- `session_id`
- `device_id`
- `app_version`
- `platform`
- `locale`
- `request_id`
- `trace_id`

### Properties

Properties are event-specific fields. They use the same portable field type system as context fields.

### Ownership and governance

Owners and PII metadata are preserved in the model from Milestone 1 onward. Policy enforcement can come later; the MVP only needs validation, preservation, docs, and generated artifacts.

Supported MVP PII classifications:

```text
none
pseudonymous
personal
sensitive
```

## 5. Target YAML Shape

Milestone 1 validates this syntax. Later milestones generate from the normalized model produced from this syntax.

```yaml
openevents: 0.1.0

namespace: com.example.product

package:
  go: github.com/example/product/events
  python: example_product.events

defaults:
  queue: product-events
  snowflake:
    database: ANALYTICS
    schema: EVENTS

owners:
  - team: data-platform
    email: data-platform@example.com

context:
  tenant_id:
    type: string
    required: true
    pii: none
    description: Stable tenant identifier.

  user_id:
    type: string
    required: false
    pii: pseudonymous
    description: Stable internal user identifier.

  platform:
    type: enum
    values: [ios, android, web, backend]
    required: true
    pii: none

events:
  user.signed_up:
    version: 1
    status: active
    description: User completed account signup.
    owner: growth
    producer: api
    sources: [ios, android, web]
    destination:
      queue: product-events
      snowflake_table: fact_user_signed_up
    properties:
      signup_method:
        type: enum
        values: [email, google, apple]
        required: true
        pii: none
      plan:
        type: string
        required: false
        pii: none
```

## 6. Portable Type System

Milestone 1 should validate these field types:

```text
string
integer
number
boolean
timestamp
date
uuid
enum
object
array
```

Every field can include:

```yaml
type: string
required: true
description: Human-readable description.
pii: none
deprecated: false
default: null
examples: []
```

Additional rules:

- `enum` fields require a non-empty `values` list with unique values.
- `array` fields require `items`.
- `object` fields require `properties` for MVP use.
- Field names are snake_case.
- Deeply nested object support can be limited in generator milestones, but Milestone 1 should still parse and validate the recursive shape consistently.

## 7. Compatibility Rules for Later Milestones

Milestone 2 will implement snapshot and diff. Milestone 1 should preserve enough normalized information to support these rules later.

| Change | Classification |
|--------|----------------|
| Add optional property | Non-breaking |
| Add required property | Breaking |
| Remove property | Breaking |
| Change property type | Breaking |
| Remove enum value | Breaking |
| Add enum value | Potentially breaking |
| Increase PII sensitivity | Review required |
| Rename event | Breaking |
| Change event version | New contract |

The baseline comparison command should eventually be:

```bash
openevents diff ./events --against ./registry.snapshot.json
```

## 8. MVP Milestone Roadmap

### Milestone 1 — Parse and Validate Registry

Goal: establish the compiler front-end.

Implemented commands:

```bash
openevents validate ./events
```

Outputs: validation success/failure and deterministic diagnostics.

### Milestone 2 — Snapshot and Diff

Goal: create normalized registry snapshots and classify compatibility changes.

Commands:

```bash
openevents snapshot ./events --out registry.snapshot.json
openevents diff ./events --against registry.snapshot.json
```

### Milestone 3 — Go Codegen

Goal: generate typed Go producer models, validators, enums, envelope types, event registry lookup, and an `Emitter` interface without broker-specific implementations.

### Milestone 4 — Python Codegen

Goal: generate Pydantic models, Literal constraints, registry dispatch, and `decode_event(raw)` for consumer pipelines.

Use Pydantic v2 unless a future implementation task discovers a strong compatibility reason not to.

### Milestone 5 — JSON Schema and Snowflake

Goal: generate one JSON Schema file per event version and one Snowflake DDL file per event.

The Snowflake MVP uses one table per event with envelope columns (`event_id`, `event_name`, `event_version`, `event_ts`, `tenant_id`, and flattened `client_*` metadata), context columns, property columns, `raw_event variant`, and `loaded_at timestamp_ntz`.

### Milestone 6 — Docs

Goal: generate a readable Markdown event catalog with owners, descriptions, versions, producers, destinations, fields, PII classification, examples, and deprecated metadata.

## 9. Milestone 1 Detailed Design

### Goal

Build a Go CLI that can load, normalize, and validate an OpenEvents registry from a file or directory.

### Non-goals

Milestone 1 must not implement:

- Go code generation.
- Python code generation.
- JSON Schema generation.
- Snowflake generation.
- Markdown docs generation.
- Snapshot or diff.
- Runtime queue emitters.

### Repository layout after Milestone 1

```text
cmd/
  openevents/
    main.go

internal/
  cli/
    root.go
    validate.go
  registry/
    diagnostic.go
    load.go
    model.go
    validate.go
    yaml.go
    load_test.go
    validate_test.go
    model_contract_test.go

examples/
  basic/
    openevents.yaml

go.mod
README.md
```

### CLI UX

```bash
openevents validate <path>
```

Behavior:

- `<path>` can be a YAML file or a directory.
- Directory loading recursively discovers `.yaml` and `.yml` files.
- File discovery order is lexical and deterministic.
- Validation succeeds with exit code `0`.
- Parse, load, or validation errors print deterministic diagnostics and exit non-zero.

Example output:

```text
ok: registry valid (2 events, 3 context fields)
```

Example diagnostics:

```text
examples/basic/openevents.yaml: events.user.signed_up.properties.signup_method.values: enum fields must define at least one value
examples/basic/openevents.yaml: events.search.query_submitted.version: version must be positive
```

### Loading and merge semantics

- A file input loads exactly that file.
- A directory input loads all `.yaml` and `.yml` files below it, sorted by relative path.
- Multiple files merge into one registry.
- Singleton fields such as `openevents`, `namespace`, `package`, and `defaults` may repeat only if they have the same value.
- `owners`, `context`, and `events` merge across files.
- Duplicate context field names are invalid.
- Duplicate `event_name + version` definitions are invalid.
- Unknown fields are invalid. Strict parsing is intentional so taxonomy typos do not silently disappear.

### Internal model boundary

Milestone 1 should separate YAML parsing DTOs from the normalized internal model.

Reasons:

- Later generators should not depend on YAML layout details.
- Validation can annotate fields with normalized names and defaults.
- Snapshot/diff can serialize a stable model rather than source YAML.
- Future importers can create the same model without going through YAML.

### Validation rules

Milestone 1 validates:

- Supported `openevents` version.
- Non-empty namespace.
- Valid package names when present.
- Event names are lowercase dot-separated identifiers.
- Event versions are positive integers.
- Field names are snake_case identifiers.
- Field types are in the supported MVP type list.
- Required flags are explicit booleans after normalization.
- Enum fields have unique non-empty values.
- Array fields have an `items` type.
- Object fields have valid nested properties.
- PII classification is one of the supported values.
- Event statuses are known values: `active`, `deprecated`, or `experimental`.
- Duplicate event names/versions are rejected.
- Duplicate context/property names are rejected by YAML structure or explicit validation.
- Required top-level fields are present.

### Error model

The validator should accumulate errors rather than fail fast. This is important for CLI UX and later CI integration.

Diagnostics must be stable in order and include enough location information to find the problem.

### Test strategy

Milestone 1 should include:

- Valid fixture under `examples/basic/openevents.yaml`.
- Loader tests for file input, directory input, deterministic merge, duplicate detection, and unknown fields.
- Validator tests for each field type and common invalid registries.
- CLI smoke test for `openevents validate ./examples/basic` if practical without overbuilding test harnesses.

## 10. Future Milestone Constraints on Milestone 1

Milestone 1 decisions should serve the future roadmap:

- The normalized model must preserve ownership, destinations, PII classification, descriptions, examples, deprecated flags, enum values, and nested field structure.
- The normalized model must order maps deterministically when serialized or reported.
- Validation should be independent from CLI printing so generators and tests can reuse it.
- Error messages should be deterministic because CI and tests will assert on them.
- YAML parsing should be strict because later generated code should not ignore misspelled contract fields.
- The model should not include broker-specific runtime behavior.

## 11. Open Decisions Deferred Beyond Milestone 1

These decisions should not block Milestone 1 unless implementation reveals a direct conflict:

- Whether event versions are per event only or also tied to a registry semantic version.
- Whether `tenant_id` is both envelope-level and context-level, or envelope-level only.
- Whether `client.source` values become global enums.
- The exact naming and format for `client.sdk` values, such as `openevents-go` versus `go/openevents`.
- Whether future generated queue JSON supports flattened formats in addition to the MVP's nested `context` and `properties`.
- Whether Snowflake always flattens all fields or can preserve `properties` as `VARIANT`.
- Whether Go optional scalar fields use pointers, generic option wrappers, or zero-value plus presence metadata.
- Whether Go validation returns one aggregated error type or a diagnostic list mirroring the CLI.
- Whether enum expansion is always potentially breaking or configurable per consumer strictness.

## Parallel Readiness

### T0 Foundation Decision

Before parallel implementation, land one sequential foundation task that creates the Go module, the normalized registry model, diagnostics, and the valid example taxonomy.

T0 exists because loader, validator, CLI, and later generator tasks all depend on the same model vocabulary. It should not implement the full loader or validator. It should define the contracts that make the next tasks safe to run independently.

Shared contract for `internal/registry/model.go`:

```go
package registry

const SupportedVersion = "0.1.0"

type Registry struct {
	Version   string
	Namespace string
	Package   PackageConfig
	Defaults  Defaults
	Owners    []Owner
	Context   map[string]Field
	Events    []Event
}

type PackageConfig struct {
	Go     string
	Python string
}

type Defaults struct {
	Queue     string
	Snowflake SnowflakeDefaults
}

type SnowflakeDefaults struct {
	Database string
	Schema   string
}

type Owner struct {
	Team  string
	Slack string
	Email string
}

type Destination struct {
	Queue          string
	SnowflakeTable string
}

type Event struct {
	Name        string
	Version     int
	Status      string
	Description string
	Owner       string
	Producer    string
	Sources     []string
	Destination Destination
	Properties  map[string]Field
}

type Field struct {
	Name        string
	Type        FieldType
	Required    bool
	Description string
	PII         PIIClassification
	Deprecated  bool
	Default     any
	Examples    []any
	Values      []string
	Items       *Field
	Properties  map[string]Field
}

type FieldType string

const (
	FieldTypeString    FieldType = "string"
	FieldTypeInteger   FieldType = "integer"
	FieldTypeNumber    FieldType = "number"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeTimestamp FieldType = "timestamp"
	FieldTypeDate      FieldType = "date"
	FieldTypeUUID      FieldType = "uuid"
	FieldTypeEnum      FieldType = "enum"
	FieldTypeObject    FieldType = "object"
	FieldTypeArray     FieldType = "array"
)

type PIIClassification string

const (
	PIINone          PIIClassification = "none"
	PIIPseudonymous  PIIClassification = "pseudonymous"
	PIIPersonal      PIIClassification = "personal"
	PIISensitive     PIIClassification = "sensitive"
)
```

Shared contract for `internal/registry/diagnostic.go`:

```go
package registry

import "strings"

type Diagnostic struct {
	Location string
	Message  string
}

type Diagnostics []Diagnostic

func (d Diagnostics) HasErrors() bool {
	return len(d) > 0
}

func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}

	lines := make([]string, 0, len(d))
	for _, diagnostic := range d {
		if diagnostic.Location == "" {
			lines = append(lines, diagnostic.Message)
			continue
		}
		lines = append(lines, diagnostic.Location+": "+diagnostic.Message)
	}
	return strings.Join(lines, "\n")
}
```

Shared contract assertions for `internal/registry/model_contract_test.go`:

```go
package registry

import "testing"

func TestRegistryModelContracts(t *testing.T) {
	var _ string = Registry{}.Version
	var _ string = Registry{}.Namespace
	var _ PackageConfig = Registry{}.Package
	var _ Defaults = Registry{}.Defaults
	var _ []Owner = Registry{}.Owners
	var _ map[string]Field = Registry{}.Context
	var _ []Event = Registry{}.Events

	var _ string = Event{}.Name
	var _ int = Event{}.Version
	var _ Destination = Event{}.Destination
	var _ map[string]Field = Event{}.Properties

	var _ string = Field{}.Name
	var _ FieldType = Field{}.Type
	var _ bool = Field{}.Required
	var _ PIIClassification = Field{}.PII
	var _ []string = Field{}.Values
	var _ *Field = Field{}.Items
	var _ map[string]Field = Field{}.Properties
}

func TestDiagnosticContracts(t *testing.T) {
	var _ error = Diagnostics{}
	var _ string = Diagnostic{}.Location
	var _ string = Diagnostic{}.Message
}
```

Shared valid example for `examples/basic/openevents.yaml`:

```yaml
openevents: 0.1.0
namespace: com.example.product

package:
  go: github.com/example/product/events
  python: example_product.events

defaults:
  queue: product-events
  snowflake:
    database: ANALYTICS
    schema: EVENTS

owners:
  - team: data-platform
    email: data-platform@example.com
  - team: growth
    slack: "#team-growth"
    email: growth-data@example.com

context:
  tenant_id:
    type: string
    required: true
    pii: none
    description: Stable tenant identifier.
  user_id:
    type: string
    required: false
    pii: pseudonymous
    description: Stable internal user identifier.
  platform:
    type: enum
    values: [ios, android, web, backend]
    required: true
    pii: none

events:
  user.signed_up:
    version: 1
    status: active
    description: User completed account signup.
    owner: growth
    producer: api
    sources: [ios, android, web]
    destination:
      queue: product-events
      snowflake_table: fact_user_signed_up
    properties:
      signup_method:
        type: enum
        values: [email, google, apple]
        required: true
        pii: none
      plan:
        type: string
        required: false
        pii: none

  search.query_submitted:
    version: 1
    status: active
    description: User submitted a search query.
    owner: data-platform
    producer: api
    sources: [ios, android, web]
    destination:
      queue: product-events
      snowflake_table: fact_search_query_submitted
    properties:
      query_text:
        type: string
        required: true
        pii: personal
        description: Raw user query text.
      result_count:
        type: integer
        required: true
        pii: none
      latency_ms:
        type: integer
        required: false
        pii: none
```

### File Ownership Matrix

| Task | Owns files | Notes |
|------|------------|-------|
| T0 Foundation | `go.mod`, `internal/registry/model.go`, `internal/registry/diagnostic.go`, `internal/registry/model_contract_test.go`, `examples/basic/openevents.yaml` | Sequential prerequisite. Defines shared contracts and valid fixture. |
| M1-A Loader | `internal/registry/yaml.go`, `internal/registry/load.go`, `internal/registry/load_test.go`, `internal/registry/testdata/load/**` | Parses YAML, discovers files, merges registry fragments, detects unknown fields and duplicate definitions. |
| M1-B Validator | `internal/registry/validate.go`, `internal/registry/validate_test.go`, `internal/registry/testdata/validate/**` | Validates normalized registry model and returns stable diagnostics. |
| M1-C CLI | `cmd/openevents/main.go`, `internal/cli/root.go`, `internal/cli/validate.go`, `internal/cli/*_test.go` | Wires Cobra command to loader and validator. Should wait for M1-A and M1-B APIs. |
| M1-D README polish | `README.md` | Documents Milestone 1 usage after CLI behavior is known. |

### Parallel Batch Manifest

| Batch | Prerequisites | Tasks | Independence proof | Validation |
|-------|---------------|-------|--------------------|------------|
| Batch 0 | None | T0 Foundation | Only task touching shared model contracts before other work begins. | `go test ./internal/registry` should compile contract tests. |
| Batch 1 | T0 Foundation | M1-A Loader, M1-B Validator | Loader owns YAML/file traversal; validator owns normalized model rules. Both depend only on T0 model and diagnostics. | `go test ./internal/registry` with loader and validator tests. |
| Batch 2 | M1-A Loader, M1-B Validator | M1-C CLI | CLI depends on stable loader and validator functions and owns separate files. | `go test ./...` and `go run ./cmd/openevents validate ./examples/basic`. |
| Batch 3 | M1-C CLI | M1-D README polish | Documentation follows implemented CLI behavior. | README command examples match working command output. |

### Validation Matrix

| Scope | Check | Proves |
|-------|-------|--------|
| T0 Foundation | `go test ./internal/registry` | Shared model and diagnostic contracts compile. |
| Loader | `go test ./internal/registry -run Load` | File discovery, YAML parsing, strict unknown-field detection, and merge semantics work. |
| Validator | `go test ./internal/registry -run Validate` | Registry validation rules return deterministic diagnostics. |
| CLI | `go test ./...` | All packages compile and command tests pass. |
| CLI smoke | `go run ./cmd/openevents validate ./examples/basic` | User-facing Milestone 1 command works on the canonical example. |

## 12. Acceptance Criteria for Milestone 1

Milestone 1 is complete when:

1. `go test ./...` passes.
2. `go run ./cmd/openevents validate ./examples/basic` exits `0`.
3. Invalid fixtures produce deterministic non-zero validation output.
4. The loader accepts a single YAML file and a directory of YAML files.
5. Duplicate event definitions are rejected.
6. Unknown YAML fields are rejected.
7. The normalized registry model preserves all fields needed by future milestones.
8. The README includes the working validate command.

## 13. Acceptance Criteria for the MVP

The full MVP is complete when:

1. A developer can define `user.signed_up` in YAML.
2. `openevents validate` catches invalid schemas.
3. `openevents snapshot` creates a deterministic baseline.
4. `openevents diff` classifies breaking changes for CI.
5. `openevents generate go` produces structs and validators.
6. Go can emit valid JSON for the event envelope and properties.
7. `openevents generate python` produces Pydantic models.
8. Python can decode the Go-produced event JSON.
9. JSON Schema can be exported per event version.
10. Snowflake DDL can be exported per event.
11. Markdown docs can be generated from the registry.

## 14. Practical Recommendation

Start with a deliberately boring compiler front-end:

1. Parse YAML strictly.
2. Normalize into an internal model.
3. Validate aggressively and accumulate stable diagnostics.
4. Keep output deterministic.
5. Make every later milestone depend on the normalized model, not source YAML.

This keeps Milestone 1 small enough to implement safely while preserving the full MVP direction.
