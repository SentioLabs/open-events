# demo-consumer

SQS-to-Parquet sink for the OpenEvents demo. See the parent
[`README.md`](../../README.md) for the end-to-end runbook and
[`GUIDE.md`](../../GUIDE.md) for the annotated walkthrough of how the consumer
fits into the pipeline.

## Quick reference

The consumer depends on the generated protobuf bindings, which the demo's
top-level `make gen` produces. From this directory, the easy path is:

```bash
(cd ../.. && make gen)   # produce protobuf bindings
uv sync --dev            # install deps
uv run pytest            # tests
uv run python -m consumer --until-empty   # drain queue and exit
```

## Configuration

Settings come from the environment:

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `OPENEVENTS_QUEUE_URL` | yes | — | SQS queue to poll |
| `OPENEVENTS_OUTPUT_DIR` | yes | — | Directory for Parquet output |
| `AWS_REGION` | no | `us-east-1` | |
| `AWS_ENDPOINT_URL` | no | — | Set for LocalStack; unset for real AWS |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | yes | — | Credentials (LocalStack ignores their value) |
| `OPENEVENTS_BATCH_SIZE` | no | `10` | Rows per flush trigger |
| `OPENEVENTS_FLUSH_INTERVAL_S` | no | `5.0` | Time-based flush trigger |

The demo's Makefile sets all of these for you. Reading `consumer/config.py`
shows the contract.

## CLI

```
demo-consumer [--until-empty]
```

`--until-empty` causes the consumer to exit cleanly after two consecutive empty
polls — useful for tests and the one-shot `make demo`. Without the flag the
consumer runs until it receives `SIGINT`/`SIGTERM`.

## Behavior

- **Long-poll** SQS for up to 10 messages at a time (`WaitTimeSeconds=20`).
- **Dispatch** on the `event_name` SQS attribute to pick the right protobuf
  decoder. Messages missing the attribute or failing to decode are dropped
  (logged with full traceback via `log.exception`); add a SQS DLQ in production.
- **Buffer** rows in memory per event type. Flush when either `batch_size` is
  reached or `flush_interval_s` has elapsed since the last flush.
- **Write** atomically: `.parquet.tmp` → `os.replace` → `.parquet`. Filenames
  are `<UTC-timestamp>-<monotonic-seq>.parquet` so flushes within the same
  second never collide.
- **Schemas** are derived from the generated proto descriptors at import time
  — see `consumer/schemas.py`. A new field in the registry shows up in Parquet
  on the next consumer restart, with no code change.
