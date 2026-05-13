# OpenEvents Demo: End-to-End Walkthrough

## What this demo shows

This is a complete end-to-end workflow from a single YAML event registry through protobuf code generation, a Go HTTP API that publishes to AWS SQS, and a Python consumer that deserializes events and writes them to Parquet. The registry (`openevents.yaml`) defines three events—`checkout.started`, `checkout.completed`, and `search.performed`—along with shared context fields. From one authoritative source, you get validated Go bindings for the API, Python bindings for analytics consumers, and a deterministic descriptor that drives schema derivation at runtime.

## Prerequisites

- **Go 1.24+**
- **Docker + docker compose** (for LocalStack SQS)
- **uv** (Python 3.11+) for the consumer
- **curl** (to send test events)

## One-shot demo

Run the full end-to-end workflow in one command:

```bash
make demo
```

This will:
1. Validate the registry and lock metadata
2. Generate protobuf from the registry
3. Spin up a local SQS (via LocalStack)
4. Start the Go API service
5. Send three sample events (checkout.started, checkout.completed, search.performed)
6. Run the Python consumer to drain the queue
7. Verify the Parquet output
8. Tear down all services

If you want to see what's happening at each step, follow the walkthrough below.

## Step-by-step walkthrough

### 1. Validate the registry and check the lock

The registry and lock are the source of truth for field numbering and backward compatibility.

```bash
go run ../../cmd/openevents validate ./examples/demo
go run ../../cmd/openevents lock check ./examples/demo
```

The validation step checks that event names, types, and owners are well-formed. The lock check verifies that any schema changes are compatible with the locked field numbers.

### 2. Generate protobuf (gen)

Generate protobuf and run codegen post-processing:

```bash
make gen
```

This will:
- Render `openevents.proto` and Buf configuration
- Run Buf to compile protobuf to Go and Python
- Execute `scripts/postgen.sh` to create Python `__init__.py` files and both `pyproject.toml` (Python) and `go.mod` (Go) for the generated modules

Output goes to `_build/demo-proto/` and `_build/demo-proto/gen/{go,python}/`.

### 3. Start LocalStack and create the SQS queue

```bash
make up
make seed
```

This brings up LocalStack with SQS enabled and creates the `storefront-events` queue.

### 4. Start the Go API service (separate terminal)

```bash
make api
```

The API listens on `http://localhost:8080` and exposes POST routes for each event:
- `/v1/events/checkout-started`
- `/v1/events/checkout-completed`
- `/v1/events/search-performed`

### 5. Send events via curl

In another terminal, post events from the sample JSON files:

```bash
curl -X POST http://localhost:8080/v1/events/checkout-started \
  -H 'content-type: application/json' \
  --data-binary @examples/demo/samples/checkout-started.json

curl -X POST http://localhost:8080/v1/events/checkout-completed \
  -H 'content-type: application/json' \
  --data-binary @examples/demo/samples/checkout-completed.json

curl -X POST http://localhost:8080/v1/events/search-performed \
  -H 'content-type: application/json' \
  --data-binary @examples/demo/samples/search-performed.json
```

Each POST to the API parses the JSON, validates it against the schema, and publishes the event to SQS.

### 6. Run the Python consumer

In another terminal, consume events from the queue and write them to Parquet:

```bash
make consumer
```

This runs the consumer with default mode (drains queue, exits when empty). You'll see log output as it deserializes messages and writes batches to Parquet.

### 7. Verify the Parquet output

```bash
make verify
```

This reads all Parquet files from `_build/demo-output/` and prints a summary. You should see three rows, one per event sent.

### 8. Clean up

```bash
make down
```

Shuts down LocalStack and removes the SQS volumes.

## What's happening under the hood

**Wire format:** Events are serialized as base64-encoded protobuf messages in the SQS message body, with the event name (e.g., `checkout.started@1`) in the SQS message attribute `event_name`. A `schema` attribute identifies the registry namespace and version.

**Descriptor-driven schemas:** The Python consumer loads the generated protobuf module and uses its descriptor (`MessageToDict`) to dynamically derive Polars schemas at load time. See `services/consumer/src/consumer/schemas.py` for the implementation. This means the schema evolves with your registry without code changes.

**At-least-once delivery + deduplication:** SQS provides at-least-once delivery. The consumer uses the `event_id` field (automatically added to every event by the registry) as the deduplication key, so replayed messages are idempotent on the Parquet sink.

## Why the Go `replace` directive?

The registry's `package.go` declares the canonical import path (`github.com/acme/storefront/events`). The demo's `services/api/go.mod` uses a `replace` directive to point at the locally generated tree under `_build/demo-proto/gen/go/com/acme/storefront/v1`. In a real project, you'd publish that module to a Go registry or module server instead of using `replace`. Local `replace` is only for development.

## Why the codegen post-step?

`protoc_builtin: python` in Buf does not emit `__init__.py` files or a `pyproject.toml`. The `scripts/postgen.sh` script wraps the generated Python code so it becomes an installable package. It also writes a minimal `go.mod` for the generated Go module so the `replace` directive can find it. In a real project, you'd publish these as separate PyPI and Go module releases instead of generating them locally on every build.

## Graduating to real AWS

To use real AWS SQS instead of LocalStack:

1. Unset `AWS_ENDPOINT_URL`:
   ```bash
   unset AWS_ENDPOINT_URL
   ```

2. Set real AWS credentials:
   ```bash
   export AWS_ACCESS_KEY_ID="your-key"
   export AWS_SECRET_ACCESS_KEY="your-secret"
   export AWS_REGION="us-east-1"
   ```

3. Point `OPENEVENTS_QUEUE_URL` to your real queue:
   ```bash
   export OPENEVENTS_QUEUE_URL="https://sqs.us-east-1.amazonaws.com/123456789012/your-queue"
   ```

That's all. The API and consumer use the AWS SDK, which respects these environment variables.

## CI recommendation

In CI, run the validation and codegen steps to ensure the registry is in sync:

```bash
make gen && make test
```

This validates the registry, locks, and generates protobuf, then runs unit tests on both services. Omit `make demo` in CI since it requires Docker; gate it behind a flag or run it only in integration test workflows.
