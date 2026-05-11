# OpenEvents demo registry

This folder is a small storefront analytics registry that demonstrates current OpenEvents commands:

```bash
go run ../../cmd/openevents validate .
go run ../../cmd/openevents generate go . <output-dir>
go run ../../cmd/openevents generate python . <output-dir>
```

From the repository root, run:

```bash
go run ./cmd/openevents validate ./examples/demo
```

Expected output:

```text
ok: registry valid (3 events, 4 context fields)
```

The demo includes:

- shared context fields with PII classifications
- multiple owners
- event-level producers, sources, and destinations
- required and optional properties
- enum fields
- an array field
- a personal-data field for search query governance

The integration test in `internal/integration/validate_demo_test.go` executes the real CLI validate and generate commands against this folder, runs generated Go code to produce event JSON, and verifies generated Python decoding against that payload.
