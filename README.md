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
