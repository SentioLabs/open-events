# OpenEvents demo registry

This folder is a small storefront analytics registry that demonstrates the current OpenEvents MVP command:

```bash
go run ../../cmd/openevents validate .
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

The integration test in `internal/integration/validate_demo_test.go` executes the real CLI against this folder so the documented example stays working.
