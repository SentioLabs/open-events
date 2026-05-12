# OpenEvents demo registry

This folder is a small storefront analytics registry that demonstrates current OpenEvents commands.

## Demo workflow

From the repository root, run:

```bash
go run ./cmd/openevents validate ./examples/demo
go run ./cmd/openevents lock update ./examples/demo
go run ./cmd/openevents generate proto ./examples/demo ./build/demo-proto
bash scripts/install-buf.sh
.tools/bin/buf lint ./build/demo-proto
.tools/bin/buf build ./build/demo-proto
.tools/bin/buf generate ./build/demo-proto
```

Expected output:

```text
ok: registry valid (3 events, 4 context fields)
```

The direct Go/Python generators are transitional. They will be deprecated only after Buf-generated Go/Python pass the demo interop integration test.

The demo includes:

- shared context fields with PII classifications
- multiple owners
- event-level producers, sources, and destinations
- required and optional properties
- enum fields
- an array field
- a personal-data field for search query governance

The integration test in `internal/integration/validate_demo_test.go` executes the real CLI validate and generate commands against this folder, runs generated Go code to produce event JSON, and verifies generated Python decoding against that payload.
