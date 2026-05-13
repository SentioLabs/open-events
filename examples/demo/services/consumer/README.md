# demo-consumer

SQS to Parquet sink for the OpenEvents demo. See `examples/demo/README.md`
for the full end-to-end walkthrough.

## Quick reference

Requires `make gen` to have run at the repo root so the generated
Python pb2 package exists at `../../../../_build/demo-proto/gen/python`.

Install and run:

```bash
uv sync --dev
uv run pytest
uv run python -m consumer --until-empty   # poll, flush, exit
```

Env vars: `OPENEVENTS_QUEUE_URL`, `AWS_ENDPOINT_URL`, `AWS_REGION`,
`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, optional
`OPENEVENTS_OUTPUT_DIR`, `OPENEVENTS_BATCH_SIZE`,
`OPENEVENTS_FLUSH_INTERVAL_S`.
