# OpenEvents demo: end-to-end workflow

## What this example demonstrates

`examples/demo` is the full OpenEvents workflow: author YAML registry files, validate, check committed lock metadata, render protobuf inputs with `generate proto`, and run Buf to produce language bindings.

## Registry files (source of truth)

- `openevents.yaml`: source-of-truth event registry definition.
- `openevents.lock.yaml`: committed schema lock metadata for deterministic field numbering and compatibility checks.

## Validation and lock checks

From the repository root:

```bash
go run ./cmd/openevents validate ./examples/demo
go run ./cmd/openevents lock check ./examples/demo
```

To intentionally refresh lock metadata after an approved schema change:

```bash
go run ./cmd/openevents lock update ./examples/demo
```

## Protobuf rendering (`generate proto` is a pure renderer)

Render protobuf + Buf config scaffolding to an output directory:

```bash
go run ./cmd/openevents generate proto ./examples/demo ./build/demo-proto
```

`generate proto` does not run Buf or compile language code. It only renders deterministic protobuf backend inputs.

## Buf lint/build/generate

Install local Buf tooling and run checks/generation:

```bash
bash scripts/install-buf.sh
.tools/bin/buf lint ./build/demo-proto
.tools/bin/buf build ./build/demo-proto
PATH="$(pwd)/.tools/bin:$PATH" .tools/bin/buf generate --template ./build/demo-proto/buf.gen.yaml ./build/demo-proto
```

## Expected output tree under `./build/demo-proto`

After `generate proto`:

- `buf.yaml`
- `buf.gen.yaml`
- `openevents.metadata.yaml`
- `proto/com/acme/storefront/v1/events.proto`

After `buf generate`:

- `gen/go/com/acme/storefront/v1/events.pb.go`
- `gen/python/com/acme/storefront/v1/events_pb2.py`

## CI recommendation

In CI, verify the same workflow from repository root:

```bash
go run ./cmd/openevents validate ./examples/demo
go run ./cmd/openevents lock check ./examples/demo
go run ./cmd/openevents generate proto ./examples/demo ./build/demo-proto
bash scripts/install-buf.sh
.tools/bin/buf lint ./build/demo-proto
.tools/bin/buf build ./build/demo-proto
PATH="$(pwd)/.tools/bin:$PATH" .tools/bin/buf generate --template ./build/demo-proto/buf.gen.yaml ./build/demo-proto
```

Keep generated output under ignored build directories (for example `build/`) instead of committing generated language artifacts.

## Deprecated direct generators

Direct `go`/`python` generators are deprecated transitional paths. Prefer `generate proto` + Buf as the durable backend-driven workflow.
