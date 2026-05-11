# OpenEvents

OpenEvents is a spec-first event taxonomy compiler for analytics, product, and data pipeline events.

Define events once in YAML, validate the registry in CI, and use later generator milestones to produce typed producer and consumer artifacts for Go, Python, JSON Schema, Snowflake, and docs.

## MVP direction

OpenEvents is intentionally narrow:

- Git-first YAML registry
- Strict validation
- Deterministic normalized model
- Future snapshot and breaking-change diff
- Future Go producer codegen
- Future Python Pydantic consumer codegen
- Future JSON Schema and Snowflake exports
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
3. Generate Go producer models.
4. Generate Python Pydantic consumer models.
5. Export JSON Schema and Snowflake DDL.
6. Generate Markdown event catalog docs.
