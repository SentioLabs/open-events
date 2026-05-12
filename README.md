# OpenEvents

OpenEvents is a spec-first event taxonomy compiler for analytics, product, and data pipeline events.

Define events once in YAML, validate the registry in CI, and treat the YAML/IR as the canonical source of truth. Schema technologies are backend/export mechanisms, not the source of truth.

## MVP direction

OpenEvents is intentionally narrow:

- Git-first YAML registry
- Strict validation
- Deterministic normalized model
- Future snapshot and breaking-change diff
- Future backend/export codegen for protobuf, Avro, and JSON Schema
- Future Go producer codegen
- Future Python Pydantic consumer codegen
- Future Snowflake exports
- Future Markdown event catalog generation

It does not implement an event broker, hosted governance UI, runtime analytics dashboard, or mobile SDK in the MVP.

## Milestone 1: validate a registry

The first milestone provides the compiler front-end and `validate` command.

```bash
go run ./cmd/openevents validate ./examples/basic
```

Expected output:

```text
ok: registry valid (2 events, 3 context fields)
```

## Demo

A larger documented example lives in `examples/demo/`.

Validate it from the repository root:

```bash
go run ./cmd/openevents validate ./examples/demo
```

Expected output:

```text
ok: registry valid (3 events, 4 context fields)
```

Generate demo code from the repository root:

```bash
go run ./cmd/openevents generate go ./examples/demo <output-dir>
go run ./cmd/openevents generate python ./examples/demo <output-dir>
```

The direct Go/Python generators are deprecated transitional paths. Use `generate proto` plus Buf as the durable backend-driven workflow.

## Backend-driven code generation

OpenEvents YAML remains the source of truth. Protobuf, Avro, and JSON Schema are backend/export formats. The first durable backend is protobuf + Buf.

### Local Buf setup

Install Buf locally through the repo script:

```bash
bash scripts/install-buf.sh
```

Validate the demo registry and committed lock, then generate protobuf output:

```bash
go run ./cmd/openevents validate ./examples/demo
go run ./cmd/openevents lock check ./examples/demo
go run ./cmd/openevents generate proto ./examples/demo ./_build/demo-proto
```

Use `go run ./cmd/openevents lock update ./examples/demo` only after an approved schema change.

Run Buf against the generated output:

```bash
.tools/bin/buf lint ./_build/demo-proto
.tools/bin/buf build ./_build/demo-proto
(cd ./_build/demo-proto && PATH="$(pwd)/../../.tools/bin:$PATH" ../../.tools/bin/buf generate .)
```

The `_build/` directory is ignored by Git and skipped by `go test ./...`.

The demo is also covered by `internal/integration/validate_demo_test.go`, which runs the real CLI validate flow, generates Go and Python code, emits JSON with generated Go types, and decodes it with generated Python types.

## Development

Run all tests:

```bash
go test ./...
```

## Example registry

See `examples/basic/openevents.yaml` for a minimal registry with:

- shared context fields
- owners
- destinations
- `user.signed_up` event
- `search.query_submitted` event
- field-level PII classifications

## Roadmap

1. Parse and validate registries.
2. Generate normalized snapshots and detect breaking changes.
3. Generate protobuf backend artifacts with Buf.
4. Generate Go producer models.
5. Generate Python Pydantic consumer models.
6. Export JSON Schema, Avro, and Snowflake DDL.
7. Generate Markdown event catalog docs.
