<!-- arc-review: kind=share-local id=t9xv2jvt -->
# OpenEvents — Backend-Neutral Codegen with Protobuf + Buf First

Status: Draft v0.3  
Audience: OpenEvents maintainers and implementation agents  
Source proposal: `~/Downloads/openevents-protobuf-buf-architecture.md`  
Decision: keep OpenEvents YAML/IR canonical, make schema technologies mostly invisible to developers, and use protobuf + Buf as the first durable backend for generated language object code.

## 1. Purpose

OpenEvents should remain a spec-first event taxonomy compiler. The product experience should be:

```text
Developer writes OpenEvents YAML
      ↓
Developer asks for target languages/artifacts
      ↓
OpenEvents generates usable event objects/helpers for those targets
```

Protobuf, Avro, JSON Schema, Buf, and schema-registry formats are **means to an end**, not the user-facing product. A developer defining analytics/product/data-pipeline events should not need to think in `.proto`, Avro schemas, or JSON Schema first. They should define events once in OpenEvents YAML and generate the code/artifacts they need.

The durable implementation path should change from direct language generators:

```text
OpenEvents YAML -> direct Go / Python / Rust / TypeScript / ... generators
```

to backend-driven generation:

```text
openevents.yaml
      ↓
OpenEvents parser + validator
      ↓
normalized OpenEvents IR
      ↓
schema/codegen backend adapters
      ├─ protobuf + Buf -> Go / Python / Rust / TypeScript / Swift / Kotlin DTOs
      ├─ Avro -> Kafka/data-platform schema-registry artifacts later
      └─ JSON Schema -> JSON validation / Snowflake / frontend artifacts later
      ↓
small OpenEvents runtime helpers where needed
```

The immediate reason for the pivot is practical: the direct Go/Python generator spike already exposed repeated language-specific edge cases — package keywords, generated identifier collisions, string literal escaping, nested enum support, Python keywords, enum naming, and cross-language optionality. Mature schema/codegen ecosystems already solve much of this.

OpenEvents should own event-taxonomy semantics, registry validation, deterministic normalization, backend-neutral schema IR, metadata/governance semantics, documentation, and evolution policy. Protobuf/Buf should own the first multi-language DTO generation path. Avro and JSON Schema should remain first-class future backends rather than afterthoughts.

## 2. Current Project Context

The current repository already has:

- A Go CLI with `validate` and recently added `generate go` / `generate python` commands.
- A registry model in `internal/registry` with portable field types, owners, package config, context fields, events, properties, PII, destinations, and examples.
- A documented demo registry in `examples/demo`.
- Integration coverage that validates the demo registry and proves generated Go/Python artifacts can exchange JSON.
- Existing roadmap issues for direct Go and Python codegen milestones.

The source proposal correctly identifies a strong implementation mechanism: `YAML -> IR -> .proto -> Buf -> generated language bindings -> OpenEvents runtime helpers`. This revised plan frames that as **protobuf first**, not **protobuf forever** and not **protobuf as the OpenEvents product model**.

## 3. Product Stance

### Developer-facing principle

The normal developer workflow should be language/artifact-oriented:

```bash
openevents validate ./events
openevents generate code ./events --target=go --out=gen/go
openevents generate code ./events --target=python --out=gen/python
openevents generate docs ./events --out=docs/events
```

Backend-specific commands can exist for debugging, CI, or advanced users:

```bash
openevents generate proto ./events ./build/openevents-proto
openevents generate avro ./events ./build/openevents-avro
openevents generate jsonschema ./events ./build/openevents-jsonschema
```

But docs should present protobuf/Avro/JSON Schema as implementation backends or export formats, not as the thing developers must author directly.

### Registry principle

The first OpenEvents registry should be low-tech and Git-first:

- YAML files are reviewed in pull requests.
- A committed lock/snapshot file records backend-stable IDs, field numbers, and compatibility metadata.
- `openevents diff` / CI detects breaking changes.
- Releases/tags identify published registry versions.

External schema registries can be added later as distribution/integration targets. OpenEvents should not require a hosted schema registry to be useful.

## 4. Alternatives Considered

### A. Continue direct language generators

Summary: keep writing first-party Go, Python, Rust, TypeScript, Swift, and Kotlin generators directly from the OpenEvents registry model.

Pros:

- Maximum control over idiomatic language APIs.
- No protobuf/Avro/JSON Schema dependency.
- Can preserve exact current JSON envelope ergonomics.

Cons:

- Every language repeats identifier, keyword, enum, optionality, packaging, escaping, and runtime compatibility work.
- Code review has already found several classes of generator bugs before the generators are mature.
- Breaking-change detection and schema evolution rules must be built largely from scratch.
- Harder to support many languages with a small project.

Assessment: not recommended for the durable path.

### B. Protobuf + Buf as the only canonical backend

Summary: make generated `.proto` the canonical event contract and use Buf/protobuf plugins for language DTOs.

Pros:

- Mature multi-language generation ecosystem.
- Strong fit for Go/Python/Rust/TypeScript/Swift/Kotlin DTOs.
- Buf provides linting, breaking checks, remote plugins, and a protobuf module registry.
- Smaller OpenEvents generator surface area than direct language generation.

Cons:

- Risks leaking protobuf concepts into OpenEvents YAML and user docs.
- Requires field-number, required/presence, envelope, and metadata decisions.
- Buf Schema Registry is useful for protobuf modules, but it is not the same as a Kafka/event-stream schema registry.
- Less ideal than Avro for some Kafka/data-lake schema-registry workflows.

Assessment: good implementation backend, but too narrow as the product architecture if treated as canonical.

### C. Avro as the first/canonical backend

Summary: generate Avro schemas first and use Avro tooling/schema registries for code and compatibility.

Pros:

- Strong fit for Kafka, Confluent-style schema registry, data pipelines, and warehouse/lake ingestion.
- Schema evolution semantics are familiar in data platforms: defaults, aliases, unions/nullability.
- Avoids protobuf field-number exposure.
- Confluent Schema Registry has first-class Avro support.

Cons:

- Generated application DTO ergonomics are generally weaker than protobuf across many target languages.
- Mobile/client/service codegen ecosystem is less compelling than protobuf + Buf for the desired broad language list.
- Still requires OpenEvents-specific runtime helpers and metadata mapping.
- Does not by itself solve docs, JSON validation, or all governance needs.

Assessment: important future backend, especially for Kafka/data-platform integration, but not the best first backend for broad object-code generation.

### D. JSON Schema as the first/canonical backend

Summary: generate JSON Schema first and use it as the primary event contract.

Pros:

- Natural fit for current JSON envelope workflows.
- Strong for validation, Snowflake-ish ingestion paths, frontend tooling, docs, and human-readable schemas.
- No binary encoding or field-number model.

Cons:

- Weaker generated strongly typed object-code story across Go/Python/Rust/Swift/Kotlin.
- Schema evolution and compatibility policies vary by tooling.
- Runtime DTO ergonomics are less standardized than protobuf.

Assessment: valuable output backend, but not the best first object-code backend.

### E. Backend-neutral OpenEvents IR with protobuf + Buf first

Summary: keep OpenEvents YAML/IR canonical; use protobuf + Buf as the first backend for generated language DTOs; preserve Avro and JSON Schema as future backends for schema registry, event streams, validation, and warehouse tooling.

Pros:

- Matches the intended product: users define OpenEvents YAML and generate target artifacts.
- Keeps protobuf mostly invisible while still leveraging Buf for codegen.
- Leaves room for Avro/schema-registry and JSON Schema/Snowflake without a later architectural rewrite.
- Lets Git + lock/snapshot files provide a low-tech registry/versioning story before any hosted registry exists.

Cons:

- Requires a cleaner internal boundary than a proto-only generator.
- Requires deciding which semantics belong in OpenEvents IR versus backend-specific adapters.
- More design work up front than a one-off `.proto` renderer.

Assessment: recommended.

## 5. Recommended Architecture

Choose **Approach E: backend-neutral OpenEvents IR with protobuf + Buf first**.

The next durable milestone can still be:

```bash
openevents generate proto ./examples/demo <output-dir>
```

However, that command should be understood as the first backend milestone and an integration/debuggable artifact, not the final developer-facing codegen UX. The product roadmap should converge on language/artifact-oriented commands powered by internal backends.

### Compiler stages

1. **Parse YAML** into the existing registry model.
2. **Validate OpenEvents semantics**: registry version, namespace, event identity, fields, PII, ownership, destinations, packages, and backend-neutral compatibility constraints.
3. **Load/update schema lock**: allocate or read stable field identities, protobuf field numbers, deprecated/reserved entries, and backend compatibility metadata. This is the hardest design problem and should be implemented deliberately, not hidden inside a renderer.
4. **Normalize to schema IR**: deterministic event order, field order, message names, field IDs, enum names, metadata, comments, and backend annotations.
5. **Run backend adapter**:
   - Protobuf adapter emits `.proto`, `buf.yaml`, `buf.gen.yaml`, and a metadata sidecar for MVP.
   - Future Avro adapter emits Avro schemas and registry metadata.
   - Future JSON Schema adapter emits JSON validation schemas.
6. **Verify backend output**: separate verification command/test step runs local `buf lint` / `buf build` for protobuf, and equivalent validators for future backends.
7. **Generate language DTOs**: MVP may document direct `buf generate`; later `openevents generate code --target=...` can invoke Buf internally.
8. **Use runtime helpers**: small OpenEvents libraries adapt DTOs to publishing, consuming, validation, and JSON envelope workflows.

### Proposed generated output shape

For the demo registry, `openevents generate proto ./examples/demo <out>` should produce a deterministic tree similar to:

```text
<out>/
  buf.yaml
  buf.gen.yaml
  proto/
    com/acme/storefront/v1/events.proto
  openevents.metadata.yaml
```

Default to **one `.proto` file per namespace** for MVP. The demo has one namespace and a small event set, so one file is clean. Multi-namespace registries can split by namespace later without over-engineering the initial renderer.

Generated language outputs from Buf should go under caller-selected output directories, not into the source repository by default:

```text
<out>/gen/go
<out>/gen/python
<out>/gen/rust
```

The repository should not check in generated language artifacts unless a later release process explicitly decides otherwise.

## 6. Registry and Versioning Strategy

### MVP: Git-first registry

OpenEvents does not need a hosted schema registry for the first durable implementation. Use Git as the registry:

- Registry YAML is the source of truth.
- A committed lock/snapshot file records stable backend identities and compatibility state.
- Pull requests show changes to both YAML and lock/snapshot output.
- CI runs `openevents validate`, `openevents diff`, and backend compilation checks.
- Git tags or releases identify published taxonomy versions.

This matches the current project shape and keeps infrastructure out of the MVP.

### Later: external schema-registry publishing

Add external registry integration as a separate backend/publish step:

```bash
openevents publish schema-registry ./events --backend=protobuf
openevents publish schema-registry ./events --backend=avro
openevents publish bsr ./events
```

Registry options to consider later:

- **Buf Schema Registry (BSR)** for protobuf modules, linting, and breaking-change workflows.
- **Confluent Schema Registry** for Avro, Protobuf, or JSON Schema in Kafka ecosystems.
- **Apicurio Registry** for multi-format registry support including Avro, Protobuf, JSON Schema, AsyncAPI, and OpenAPI.
- **Plain Git artifacts** for teams that only need PR-based governance and generated code in CI.

The schema registry should be a distribution and compatibility surface, not a requirement for defining or generating event code.

## 7. Stable Backend IDs, Lock File, and Field Numbering

### Decision

Do **not** require normal YAML authors to write protobuf field numbers by hand. That leaks backend detail into the OpenEvents authoring experience.

OpenEvents should generate `proto_number` values itself. Developers should edit only event YAML during normal authoring; the lock file is updated by OpenEvents tooling and reviewed in Git diffs. Seeing `proto_number` in the lock file is acceptable because it is implementation metadata, but requiring authors to choose those numbers in event definitions is not.

Instead, OpenEvents should manage stable backend IDs in a committed lock/snapshot file. This is the highest-risk design area because it affects merge conflict ergonomics, CI validation, compatibility guarantees, and whether the file is human-edited or machine-managed.

### Lock file stance

The lock file should be:

- committed to Git;
- deterministic and reviewable in pull requests;
- generated/updated by OpenEvents commands;
- human-readable but not normally human-authored;
- the source for backend-stable IDs and protobuf field numbers;
- the place removed fields become reserved instead of disappearing.

Possible shape:

```yaml
# openevents.lock.yaml
version: 1
registry: examples/demo
schema:
  events:
    checkout.started@1:
      envelope:
        event_name: { proto_number: 1 }
        event_version: { proto_number: 2 }
        event_id: { proto_number: 3 }
        event_ts: { proto_number: 4 }
        client: { proto_number: 5 }
        context: { proto_number: 6 }
        properties: { proto_number: 7 }
      properties:
        cart_id:
          field_id: fld_01
          proto_number: 1
        currency:
          field_id: fld_02
          proto_number: 2
      reserved: []
  context:
    tenant_id:
      field_id: ctx_01
      proto_number: 1
```

The exact file name and shape can be refined during planning. The important rule is stable identity lives in OpenEvents-managed registry metadata committed to Git, not in sorted map order and not necessarily in the human-authored event YAML.

### Allocation algorithm

For each message scope independently — shared context, client if generated, event envelope, and each event properties message:

1. Load existing lock entries for active and reserved fields.
2. Preserve every existing `proto_number` for fields that still exist.
3. For new fields, allocate the next available protobuf field number greater than all active/reserved numbers in that message.
4. Do not reuse gaps by default. Reuse can be a future explicit/manual operation after compatibility review.
5. Exclude protobuf's reserved implementation range `19000-19999`.
6. When a field is removed, move its number/name into that message's `reserved` list instead of deleting it.
7. Treat a rename as remove + add unless a future explicit rename/alias mechanism proves the change is compatible.
8. Sort output deterministically so repeated runs produce byte-identical locks.

Envelope field numbers should be fixed by convention for every event envelope:

```text
1 event_name
2 event_version
3 event_id
4 event_ts
5 client
6 context
7 properties
```

Properties and context fields use the allocation algorithm above.

### Merge conflict and CI ergonomics

Two branches adding fields to the same message may both allocate the same next number. That is acceptable before publication, but it must be easy to resolve:

- CI should run a lock check and fail on duplicate numbers, missing lock entries, unsorted lock output, or generated output drift.
- The developer resolves by rebasing/merging and running the lock update command again; the newly added field on one side receives the next number.
- Once a lock is published/tagged, existing numbers are immutable.
- The lock check should distinguish unpublished PR allocation conflicts from true breaking changes against a published baseline once snapshot/diff support exists.

### Commands to design

Planning should decide the exact commands, but the intended split is:

```bash
openevents lock update ./events          # writes/updates openevents.lock.yaml
openevents lock check ./events           # CI check: lock is present, sorted, complete, and non-conflicting
openevents generate proto ./events <out> # pure renderer that reads the lock and fails if it is missing/stale
```

A `--update-lock` convenience flag can be added later, but the lock update should be an explicit operation in the first implementation so code generation does not silently mutate registry state.

### Implementation slicing

Do not let the lock file turn T0 into a multi-day foundation with no visible output. Split it deliberately:

- **T0a**: minimal schema IR and deterministic proto renderer for the demo shape, with fixed envelope numbers and an initial lock/check contract.
- **T0b**: lock allocation/update algorithm, merge-conflict validation, and removed-field reservation.

T0a can make the protobuf backend visible quickly. T0b must land before the feature is considered production-safe or before direct generators are deprecated.

### Author escape hatch

Advanced users can eventually be allowed to specify explicit stable IDs or backend numbers, but that should be optional and clearly documented as an advanced interoperability feature.

### Reserved/deprecated fields

The lock/snapshot should preserve removed/deprecated fields so generated protobuf can emit `reserved` numbers/names and future Avro/JSON Schema backends can apply equivalent compatibility rules.

Example eventual YAML or lock shape:

```yaml
reserved:
  fields:
    - name: old_coupon_code
      proto_number: 4
      reason: Replaced by coupon_id.
```

MVP proto generation should preserve `deprecated: true` as comments/metadata. Field removals should become reserved entries once T0b lock/update support exists.

## 8. Envelope vs Payload Model

### Decision

For MVP, generate **per-event envelope messages** that preserve the current OpenEvents shape:

```proto
message CheckoutStartedV1 {
  string event_name = 1;
  int32 event_version = 2;
  string event_id = 3;
  google.protobuf.Timestamp event_ts = 4;
  Client client = 5;
  Context context = 6;
  CheckoutStartedV1Properties properties = 7;
}
```

Also generate shared messages:

```proto
message Client { ... }
message Context { ... }
message CheckoutStartedV1Properties { ... }
```

The event identity fields are redundant with the message type, but they preserve compatibility with the current JSON envelope and analytics/event-router workflows. They also make JSON payloads self-describing when serialized outside protobuf-aware systems.

### Alternatives deferred

- **Payload-only messages** are too narrow for the current OpenEvents envelope and demo integration.
- **One shared envelope with `oneof` payloads** is useful later for transport/routing, but it adds cross-event coupling and oneof evolution complexity. Defer it until the basic per-event envelope path is proven.

## 9. Required and Presence Semantics

Proto3 does not model `required: true` the same way OpenEvents does. OpenEvents requiredness is a validation rule, not just a protobuf wire-format property.

### Decision

Use proto3 `optional` to preserve scalar presence for OpenEvents fields whose presence must be validated.

Rules for the protobuf MVP:

- OpenEvents `required: true` means the field must be present according to OpenEvents validation.
- OpenEvents `required: false` means the field may be absent.
- Do **not** use protobuf `required`.
- Emit proto3 `optional` for scalar context/property fields so runtime validation can distinguish absent from zero/default values.
- Emit proto3 `optional` for enum context/property fields for the same reason.
- Message fields already have presence in generated APIs; required message fields are validated by OpenEvents runtime/helpers.
- Repeated fields do not have scalar-style presence; `required: true` on an array means non-empty in OpenEvents validation.
- Envelope fields can be emitted as normal non-optional fields because they are generated/managed by OpenEvents helpers rather than authored as arbitrary user payload.

Example protobuf direction:

```proto
// Required by OpenEvents metadata; optional in proto3 to preserve presence.
optional string tenant_id = 1;

// Optional by OpenEvents metadata; also optional in proto3 to preserve presence.
optional string user_id = 2;
```

Because custom proto options are deferred for MVP, requiredness should be recorded in comments plus the generated sidecar metadata. Future runtime helpers can enforce requiredness from OpenEvents IR/metadata before publish and after decode.

For future Avro, requiredness maps more naturally to nullable unions/defaults. For JSON Schema, it maps to the schema's `required` arrays. That is another reason the OpenEvents IR should hold requiredness independent of any one backend.

## 10. Enum Strategy

Proto enum generation must account for stricter protobuf naming rules and required zero values.

### Decision for protobuf backend

Generate enum definitions close to their owning message/field, with stable names and prefixed values.

For a `platform` context enum:

```proto
message Context {
  enum Platform {
    PLATFORM_UNSPECIFIED = 0;
    PLATFORM_IOS = 1;
    PLATFORM_ANDROID = 2;
    PLATFORM_WEB = 3;
    PLATFORM_BACKEND = 4;
  }

  Platform platform = 4;
}
```

For event property enums, nest under the property message:

```proto
message CheckoutStartedV1Properties {
  enum Currency {
    CURRENCY_UNSPECIFIED = 0;
    CURRENCY_USD = 1;
    CURRENCY_EUR = 2;
    CURRENCY_GBP = 3;
  }

  Currency currency = 4;
}
```

Rules:

- Always generate a zero `*_UNSPECIFIED = 0` value for protobuf.
- Allocate enum values deterministically from YAML order plus lock metadata where needed.
- Reject YAML enum values that cannot produce valid, collision-free backend constants.
- Preserve original string values in OpenEvents metadata so JSON/event analytics compatibility remains possible.

Future Avro and JSON Schema backends can preserve string enum values more directly.

## 11. Metadata Mapping

OpenEvents metadata should not become payload fields unless it is part of the runtime event envelope.

Payload/envelope fields:

- `event_name`
- `event_version`
- `event_id`
- `event_ts`
- `client`
- `context`
- `properties`

Schema/governance metadata:

- owner/team/email/slack
- PII classification
- retention
- topic/queue
- partition key
- producer
- sources
- Snowflake database/schema/table
- description/status/deprecation

### Decision

Keep schema/governance metadata in OpenEvents IR and emit it through each backend's metadata mechanism:

- Protobuf MVP: deterministic comments plus a generated sidecar metadata file.
- Protobuf later: custom options only if descriptor-attached metadata becomes necessary.
- Avro: schema properties/custom attributes where compatible with target registry tooling.
- JSON Schema: annotations/extensions such as `x-openevents-*` fields.
- Docs: direct Markdown/catalog rendering from OpenEvents IR.

Custom proto options should be **deferred**. They require authoring/distributing an `options.proto` that every consumer must import, complicate Buf module layout, and are not necessary for the first visible backend milestone. Comments plus `openevents.metadata.yaml` are sufficient for MVP.

For protobuf custom options, a future shape could be:

```proto
message CheckoutStartedV1 {
  option (openevents.event).name = "checkout.started";
  option (openevents.event).version = 1;
  option (openevents.event).topic = "storefront-events";
  option (openevents.event).owner = "growth";
  option (openevents.event).snowflake_table = "fact_checkout_started";
}
```

The architecture should keep metadata backend-neutral regardless of whether custom options are added later.

## 12. Buf Integration and Local Installation

Buf is not currently on PATH locally; `protoc` is installed. The pivot should include Buf as a repo-local tool dependency instead of relying only on global installs.

### Decision

Add a local tool installation path, such as:

```text
.tools/bin/buf
```

with a script:

```bash
scripts/install-buf.sh
```

The script should install a pinned Buf version locally, for example via Go tooling:

```bash
GOBIN="$PWD/.tools/bin" go install github.com/bufbuild/buf/cmd/buf@<pinned-version>
```

The exact version should be selected during implementation from the current stable Buf release and committed in one place, such as:

```text
.tools/buf.version
```

or directly in the script.

Validation commands should use the local binary explicitly:

```bash
.tools/bin/buf --version
.tools/bin/buf lint <generated-output>
.tools/bin/buf build <generated-output>
.tools/bin/buf generate <generated-output>
```

`openevents generate proto` should be a **pure renderer** for MVP. It should not shell out to Buf automatically because that would make rendering depend on a local Buf install and network/plugin availability. Verification should be a separate command/test step.

Possible later commands:

```bash
openevents verify proto <generated-output>          # shells out to local/pinned Buf
openevents generate code ./events --target=go       # internally renders proto and runs Buf
openevents generate code ./events --target=python   # internally renders proto and runs Buf
```

For MVP, explicit Buf commands in tests/docs are acceptable as long as the dependency is pinned and local.

### Config strategy

Generated proto output should include enough config to run Buf in the output tree:

```yaml
# buf.yaml
version: v2
modules:
  - path: proto
```

```yaml
# buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/protocolbuffers/python
    out: gen/python
  - remote: buf.build/community/neoeinstein-prost
    out: gen/rust
```

The first MVP can include only Go/Python if Rust plugin setup adds avoidable friction, but the architecture should remain Buf-plugin based.

## 13. Migration from Direct Go/Python Generators

### Decision

Treat direct Go/Python generation as superseded by the backend-driven path, but do not remove or deprecate it until the Buf-generated path proves the same integration value.

Recommended migration path:

1. Stop expanding the direct generators beyond keeping the repository green.
2. Add backend-neutral schema IR and a protobuf + Buf backend as the new durable codegen path.
3. Update docs to describe OpenEvents language codegen as powered by schema backends, with protobuf + Buf first.
4. Add `generate proto` and verify generated proto with Buf.
5. Add Buf-generated Go/Python integration that matches the current direct-generator demo: generated Go constructs/serializes an event and generated Python parses/validates it, or both interoperate through protobuf JSON/binary format.
6. Only after that gate passes, mark direct `generate go` / `generate python` deprecated or remove them.
7. Update or supersede the open direct-codegen arc issue so future work does not continue hardening the wrong path.

Explicit migration gate:

> Direct generators are deprecated only when Buf-generated Go and Python can pass the existing Go↔Python event interop integration test goal.

That means deprecation requires `buf generate` working end-to-end, not merely `generate proto` producing compilable `.proto` files.

## 14. MVP Scope for the Pivot

### In scope for the first backend milestone

- Minimal backend-neutral schema IR sufficient for the demo registry's protobuf output.
- One `.proto` file per namespace, starting with the demo namespace.
- `openevents generate proto <registry-path> <output-dir>` CLI command as a pure backend/debug artifact renderer.
- Deterministic proto package/message/name generation from the existing demo registry.
- Comments plus `openevents.metadata.yaml` sidecar metadata; no custom proto options in MVP.
- Generated `buf.yaml` and `buf.gen.yaml` in the output directory.
- Local Buf install script and documented verification commands.
- Tests that compile/build the generated proto output.
- A focused lock-file slice that prevents silent renumbering. Full merge ergonomics can be T0b/fast-follow, but production safety requires it before deprecation.
- README and demo docs updated to explain the backend-driven path.

### Out of scope for the first backend milestone

- Hosted schema registry.
- External schema-registry publish commands.
- Full runtime publisher/consumer libraries.
- Kafka/SQS/NATS transport integrations.
- Full AsyncAPI export.
- Full Snowflake/JSON Schema replacement.
- Avro backend implementation.
- Custom protobuf options and `options.proto` distribution.
- Multi-registry remote Buf module publishing.
- Removing direct Go/Python generators before Buf-generated interop passes.

## 15. Validation and Test Strategy

Minimum gates:

```bash
go test ./...
go run ./cmd/openevents validate ./examples/demo
go run ./cmd/openevents generate proto ./examples/demo <tmp-out>
.tools/bin/buf lint <tmp-out>
.tools/bin/buf build <tmp-out>
```

Generated proto tests should use both golden and structural checks:

- **Golden-file tests** for the full demo `.proto`, `buf.yaml`, `buf.gen.yaml`, and metadata sidecar. Determinism is part of the product contract, so snapshotting the whole demo output is valuable.
- **Structural unit tests** for focused edge cases: package naming, message naming, enum zero values, optional scalar emission, duplicate/colliding names, missing/stale lock entries, and one-file-per-namespace layout.
- **Integration tests** that generate to a temp directory and run the pinned local Buf binary with `lint` and `build`.

Follow-up gates for migration:

```bash
.tools/bin/buf generate <tmp-out>
go test ./... # including generated Go compile/integration, if wired into temp module
python -m pytest # or a simple generated Python decode/import smoke test
```

The integration test should eventually prove:

1. OpenEvents validates the registry.
2. OpenEvents maintains stable backend IDs/field numbers via Git-visible registry metadata.
3. OpenEvents emits deterministic proto and Buf config.
4. Buf generates Go and Python DTOs.
5. Generated Go can construct/serialize an event.
6. Generated Python can parse/decode the same payload, or both can interoperate via protobuf JSON/binary format.

## 16. Resolved Defaults and Open Questions for Planning

### Resolved defaults

- Proto layout: one `.proto` file per namespace for MVP.
- Custom proto options: defer; use comments plus sidecar metadata.
- `generate proto`: pure renderer; Buf lint/build/generate is a separate verification/generation step.
- Direct generator migration: deprecate only after Buf-generated Go/Python interop passes.
- T0 scope: keep minimal visible output first; split full lock allocation into T0b if necessary.

### Open questions

These should be resolved before implementation tasks are created:

1. What should the lock/snapshot file be named and where should it live for a directory registry?
2. Should lock update be a separate `openevents lock update` command from day one, or an explicit `generate proto --update-lock` flag?
3. How much of T0b lock allocation must land before the first visible `generate proto` merge, and how much can be a follow-up issue that blocks deprecation only?
4. Should the first Buf generation include Rust, or should Rust remain a second batch after Go/Python succeeds?
5. Should the future developer-facing `openevents generate code --target=go` invoke Buf internally, or should OpenEvents only generate backend project files and instruct users to run Buf?

## Parallel Readiness

### T0 Foundation Decision

The implementation should start with a deliberately small sequential foundation, then split the lock file work into a focused follow-up slice if necessary.

T0a should define only what is needed to render and verify the demo registry's protobuf output:

- Minimal schema IR used by the protobuf renderer.
- Proto naming rules for packages, one-file-per-namespace layout, messages, fields, enums, optional scalars, and comments.
- Local Buf install script and pinned version.
- A pure `generate proto` renderer contract.

T0b should focus on the harder lock-file behavior:

- Stable ID/protobuf-number lock representation.
- Lock update/check command behavior.
- New-field allocation without renumbering existing fields.
- Merge conflict/duplicate-number CI behavior.
- Removed-field reservation.

T0a can land visible output quickly. T0b should block production claims, direct-generator deprecation, and any promise of stable protobuf evolution.

Shared contract sketch for the minimal schema IR:

```go
// internal/schemair/model.go
package schemair

type Registry struct {
	Namespace string
	Files     []File
}

type File struct {
	Path     string
	Package  string
	Messages []Message
}

type Message struct {
	Name        string
	Description string
	Fields      []Field
	Enums       []Enum
}

type Field struct {
	Name        string
	Number      int
	Type        TypeRef
	Repeated    bool
	Optional    bool
	Required    bool
	Description string
}

type Enum struct {
	Name   string
	Values []EnumValue
}

type EnumValue struct {
	Name     string
	Original string
	Number   int
}

type TypeRef struct {
	Scalar  string
	Message string
	Enum    string
}
```

Contract assertions should be placed in tests when T0a is implemented:

```go
// internal/schemair/model_test.go
package schemair

// --- Contract assertions ---

var _ string = Registry{}.Namespace
var _ []File = Registry{}.Files
var _ string = File{}.Package
var _ []Message = File{}.Messages
var _ int = Field{}.Number
var _ bool = Field{}.Optional
var _ bool = Field{}.Required
var _ []EnumValue = Enum{}.Values
```

Shared contract sketch for the T0b schema lock:

```go
// internal/schemair/lock.go
package schemair

type Lock struct {
	Version int
	Context map[string]LockedField
	Events  map[string]LockedEvent
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
```

Contract assertions for T0b:

```go
// internal/schemair/lock_test.go
package schemair

// --- Contract assertions ---

var _ int = Lock{}.Version
var _ map[string]LockedEvent = Lock{}.Events
var _ string = LockedField{}.StableID
var _ int = LockedField{}.ProtoNumber
var _ []ReservedField = LockedEvent{}.Reserved
```

### File Ownership Matrix

| Task | Owned files/directories | Notes |
| --- | --- | --- |
| T0a Minimal foundation | `internal/schemair/model.go`, `internal/schemair/model_test.go`, `scripts/install-buf.sh`, `.tools/buf.version` | Must land before renderer/CLI work. |
| T0b Lock workflow | `internal/schemair/lock.go`, `internal/schemair/lock_*.go`, lock CLI plumbing/tests | Blocks production stability and deprecation, but can be planned as a separate focused slice. |
| Proto rendering | `internal/protogen/render.go`, `internal/protogen/render_test.go`, proto templates/helpers | Depends on T0a schema IR. |
| Registry/schema validation | `internal/registry/model.go`, `internal/registry/validate.go`, registry tests, `examples/demo/openevents.yaml` | Adds validation for backend-compatible field names/enums/requiredness. |
| CLI integration | `internal/cli/generate.go`, `internal/cli/root.go`, CLI tests | Adds pure `generate proto`; should be serialized after renderer contract if overlap is high. |
| Integration tests/docs | `internal/integration/*`, `README.md`, `examples/demo/README.md` | Verifies generated proto/Buf output and documents backend-driven codegen. |
| Buf-generated interop | generated-code temp test harness under `internal/integration/*` | Required before direct-generator deprecation. |
| Migration cleanup | `internal/codegen/*`, docs, arc issue updates | Deprecates/removes direct Go/Python path only after Buf interop passes. |

### Parallel Batch Manifest

Batch 0 — sequential minimal foundation:

- T0a Minimal foundation: schema IR + local Buf tooling + renderer contract.
- Must complete before proto rendering and CLI work.

Batch 1 — parallel after T0a:

- Proto rendering implementation.
- Registry/schema validation for backend-compatible names/enums/requiredness.
- CLI integration skeleton for pure `generate proto` if renderer interface is stable; otherwise serialize CLI after renderer.

Batch 2 — focused stability slice:

- T0b Lock workflow: update/check commands, allocation algorithm, merge-conflict validation, reserved removed fields.
- This can run after or alongside Batch 1 only if file ownership stays separate, but it blocks production/deprecation.

Batch 3 — integration and migration:

- Generated proto golden/structural/integration tests and docs.
- Buf-generated Go/Python interop test.
- Migration cleanup/deprecation of direct generators after interop passes.

Independence proof: T0a owns only minimal schema IR/tooling. Renderer, registry validation, CLI, lock workflow, and integration docs own distinct files. Any CLI overlap with lock commands should be serialized in arc-plan.

### Validation Matrix

| Work item | Validation |
| --- | --- |
| T0a Minimal foundation | `go test ./internal/schemair`; `.tools/bin/buf --version` after install script. |
| T0b Lock workflow | tests proving new fields allocate numbers without renumbering existing fields; duplicate allocations fail; removed fields become reserved; lock output is deterministic. |
| Proto rendering | golden tests for demo `.proto`/Buf config/metadata sidecar plus structural tests; `go test ./internal/protogen`. |
| Registry/schema validation | table-driven validation tests for backend-compatible names, enum naming, requiredness/presence behavior. |
| CLI integration | CLI tests for pure `generate proto`; error-path tests for invalid output path and missing/stale lock once T0b lands. |
| Integration tests/docs | `go test ./...`; generated proto compiles with `.tools/bin/buf build <tmp-out>`. |
| Buf-generated interop | `buf generate` succeeds; generated Go/Python can exchange or parse the same event payload. |
| Migration cleanup | `go test ./...`; docs no longer present direct generators as the primary path. |

## 17. Routing Analysis

Work items: 8+ tasks identified.  
Parallel ready: Yes, after T0a establishes minimal shared contracts; T0b lock workflow is a focused stability slice.  
Files touched: ~18-30 files across schema IR, lock/snapshot handling, CLI, registry validation, protogen, scripts/tools, examples, integration tests, and docs.  
Layers crossed: registry model/validation, schema IR, lock/versioning, compiler/generator, CLI, local tooling, integration tests, docs, issue migration.  
Risk areas: lock allocation/merge ergonomics, codegen architecture pivot, stable ID/field-number allocation, Buf dependency, generated-code interop, direct-generator deprecation, possible future schema-registry expectations.  
Scale: Medium to large.

Recommendation: proceed to `/arc-plan` after design approval. The work crosses multiple layers and should be broken into self-contained arc issues with T0a minimal foundation first, then T0b lock stability before any production/deprecation claims.
