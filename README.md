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
go run ./cmd/openevents validate ./examples/demo/registry
```

Expected output:

```text
ok: registry valid (12 events, 4 context fields)
```

## Demo

A larger end-to-end example — a Go API publishing to SQS into a Python consumer
that lands events in Parquet — lives in [`examples/demo/`](./examples/demo/).
See [`examples/demo/GUIDE.md`](./examples/demo/GUIDE.md) for an annotated
walkthrough; [`examples/demo/README.md`](./examples/demo/README.md) is the
runbook.

Validate it from the repository root:

```bash
go run ./cmd/openevents validate ./examples/demo/registry
```

Expected output:

```text
ok: registry valid (3 events, 4 context fields)
```

Generate protobuf output and run Buf against it (see the Backend-driven code generation section below for the full workflow).

## Backend-driven code generation

OpenEvents YAML remains the source of truth. Protobuf, Avro, and JSON Schema are backend/export formats. The first durable backend is protobuf + Buf.

### Local Buf setup

Install Buf locally through the repo script:

```bash
bash scripts/install-buf.sh
```

Validate the demo registry and committed lock, then generate protobuf output:

```bash
go run ./cmd/openevents validate ./examples/demo/registry
go run ./cmd/openevents lock check ./examples/demo/registry
go run ./cmd/openevents generate proto ./examples/demo/registry ./_build/demo-proto
```

Use `go run ./cmd/openevents lock update ./examples/demo/registry` only after an approved schema change.

Run Buf against the generated output:

```bash
.tools/bin/buf lint ./_build/demo-proto
.tools/bin/buf build ./_build/demo-proto
(cd ./_build/demo-proto && PATH="$(pwd)/../../.tools/bin:$PATH" ../../.tools/bin/buf generate .)
```

The `_build/` directory is ignored by Git and skipped by `go test ./...`.

The demo is also covered by `internal/integration/validate_demo_test.go`, which runs the real CLI validate flow, generates protobuf output, builds it with Buf, and verifies end-to-end Go/Python interop using `protoc-gen-go` and `protoc-gen-python` against the pinned protobuf runtime.

## Development

Run all tests:

```bash
go test ./...
```

## Roadmap

1. Parse and validate registries.
2. Generate normalized snapshots and detect breaking changes.
3. Generate protobuf backend artifacts with Buf.
4. Generate Go producer models.
5. Generate Python Pydantic consumer models.
6. Export JSON Schema, Avro, and Snowflake DDL.
7. Generate Markdown event catalog docs.
