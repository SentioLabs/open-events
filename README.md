# OpenEvents

OpenEvents is a spec-first event taxonomy compiler for analytics, product, and data pipeline events.

Define events once in a YAML directory registry, validate in CI, and treat the YAML as the canonical source of truth. Producers, consumers, schemas, and warehouse exports are all generated from it.

## MVP direction

OpenEvents is intentionally narrow:

- Git-first YAML directory registry (root file + per-domain + per-action)
- Strict validation
- Deterministic normalized model and locked field numbers
- Codegen for protobuf and cross-language constants
- Future snapshot and breaking-change diff
- Future Avro and JSON Schema backends
- Future Snowflake DDL and metadata exports

It does not implement an event broker, hosted governance UI, runtime analytics dashboard, or mobile SDK in the MVP.

## Milestone 1: validate a registry

The first milestone validates the registry, locks field numbers, and generates protobuf + cross-language constants.

```bash
go run ./cmd/openevents validate ./examples/demo/registry
go run ./cmd/openevents lock check ./examples/demo/registry
go run ./cmd/openevents generate ./examples/demo/registry
```

Expected output from `validate`:

```text
ok: registry valid (12 events, 4 context fields)
```

## Demo

A larger end-to-end example — a Go API publishing to SQS, a Python consumer landing events in Parquet — lives in [`examples/demo/`](./examples/demo/). It exercises two domains (user, device) with 12 events total.

See [`examples/demo/GUIDE.md`](./examples/demo/GUIDE.md) for an annotated walkthrough; [`examples/demo/README.md`](./examples/demo/README.md) is the runbook.

Validate the registry from the repository root:

```bash
go run ./cmd/openevents validate ./examples/demo/registry
```

## Registry structure

A registry is a directory with:

- **`openevents.yaml`** (root) — namespace, package paths, owners, codegen language targets
- **`<domain>/domain.yml`** — domain metadata (optional, for multi-service registries)
- **`<domain>/<action>/<action>.yml`** — event definition (type, properties, context requirements, ownership, destination)
- **`openevents.lock.yaml`** (committed) — pinned protobuf field numbers for wire-format stability

Example:

```text
registry/
├── openevents.yaml           # namespace, codegen config
├── openevents.lock.yaml      # field numbers (stable)
├── user/                      # domain 1
│   ├── domain.yml
│   ├── auth/
│   │   ├── signup.yml
│   │   ├── login.yml
│   │   └── logout.yml
│   └── cart/
│       ├── checkout.yml
│       ├── item_added.yml
│       └── purchase.yml
└── device/                    # domain 2
    ├── domain.yml
    ├── info/
    │   ├── hardware.yml
    │   ├── software.yml
    │   └── calibration.yml
    ├── diagnostics/
    │   └── stack_usage.yml
    └── incident/
        ├── drop.yml
        └── temperature.yml
```

## CLI commands

The `openevents` command provides:

| Command | Purpose |
|---------|---------|
| `validate <registry>` | Check YAML well-formedness and schema constraints |
| `lock check <registry>` | Verify protobuf field numbers match the lock file |
| `lock update <registry>` | Allocate new field numbers and update the lock file |
| `generate <registry>` | Emit protobuf + Buf config + codegen bindings (Go, Python) |

The `_build/` directory contains generated artifacts and is ignored by Git.

## Development

Run all tests:

```bash
go test ./...
```

## Roadmap

1. Parse and validate registries.
2. Generate normalized snapshots and detect breaking changes.
3. Lock and manage protobuf field numbers.
4. Codegen protobuf + Buf integration.
5. Codegen cross-language constants.
6. Future Go producer and Python consumer codegen.
7. Export JSON Schema, Avro, and Snowflake DDL.
8. Generate Markdown event catalog docs.
