# OpenEvents demo

A realistic storefront pipeline that exercises every moving part of OpenEvents:
a directory registry with 12 events across 2 domains (user, device) → protobuf codegen → Go API publishing to SQS → Python consumer landing events in Parquet.

> **Want to understand how it works?** Read
> [`GUIDE.md`](./GUIDE.md) — an annotated walkthrough of every component.
> This README is the runbook.

## Prerequisites

- **Go 1.25+**
- **Docker + Docker Compose** (for LocalStack SQS)
- **uv** (Python 3.11+) — installs the consumer
- **curl** (to send test events)

No global install of `openevents` is needed; the Makefile runs the CLI in place.

## One-shot demo

```bash
make demo
```

That target:

1. Validates the registry and lock
2. Generates protobuf and cross-language constants via `openevents generate`
3. Brings up LocalStack and creates the SQS queue
4. Builds and starts the Go API
5. POSTs the 12 sample events under [`samples/`](./samples/) across both domains
6. Drains the queue with the Python consumer (`--until-empty`)
7. Prints a summary of the Parquet output
8. Tears LocalStack and the API down

The whole thing takes about a minute from a cold start. Repeat runs are
faster — most of the time is LocalStack image download on the first run.

## Targets

| Target | What it does |
|--------|--------------|
| `make gen` | Validate + lock-check + generate (proto + constants). Idempotent; re-run after any registry change. |
| `make up` | `docker compose up -d localstack` and wait for SQS to be ready. |
| `make seed` | Create the `storefront-events` SQS queue inside LocalStack. |
| `make api` | Run the Go API in the foreground on `:8080`. Use in a second terminal during a step-by-step walkthrough. |
| `make consumer` | Run the Python consumer in the foreground (drains, exits when the queue is empty for two polls). |
| `make demo` | The one-shot pipeline described above. |
| `make verify` | Read each per-event-type directory under `_build/demo-output/` and print its rows. |
| `make test` | Run Go tests in `services/api/` and pytest in `services/consumer/`. |
| `make down` | `docker compose down -v` — stops LocalStack and removes its volumes. |
| `make clean` | Remove the Parquet output and LocalStack scratch dirs. |

## Sending events by hand

With `make up && make seed && make api` running, send an event:

```bash
curl -X POST http://localhost:8080/v1/events/user/auth/signup \
  -H 'content-type: application/json' \
  --data-binary @samples/user-auth-signup.json
```

Routes (all POST):

**User domain:**
- `/v1/events/user/auth/signup`
- `/v1/events/user/auth/login`
- `/v1/events/user/auth/logout`
- `/v1/events/user/cart/checkout`
- `/v1/events/user/cart/item_added`
- `/v1/events/user/cart/purchase`

**Device domain:**
- `/v1/events/device/info/hardware`
- `/v1/events/device/info/software`
- `/v1/events/device/info/calibration`
- `/v1/events/device/diagnostics/stack_usage`
- `/v1/events/device/incident/drop`
- `/v1/events/device/incident/temperature`

**Health:**
- `GET /healthz`

Each successful POST returns `202 Accepted` with `event_id`, `queue_url`, and
`message_id`.

## Graduating to real AWS

The Go API and Python consumer use the standard AWS SDK environment variables,
so swapping LocalStack for real SQS is a matter of unsetting one variable and
setting three more:

```bash
unset AWS_ENDPOINT_URL
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_REGION="us-east-1"
export OPENEVENTS_QUEUE_URL="https://sqs.us-east-1.amazonaws.com/<account>/<queue>"
```

Then `make api` and `make consumer` are unchanged.

## Layout

```text
examples/demo/
├── GUIDE.md                # annotated walkthrough (start here for "how")
├── README.md               # this file (runbook)
├── Makefile                # make targets
├── docker-compose.yaml     # LocalStack config
├── registry/               # the single source of truth
│   ├── openevents.yaml         # root: namespace, packages, codegen config
│   ├── openevents.lock.yaml    # pinned protobuf field numbers
│   ├── user/                   # domain 1
│   │   ├── domain.yml
│   │   ├── auth/
│   │   │   ├── signup.yml
│   │   │   ├── login.yml
│   │   │   └── logout.yml
│   │   └── cart/
│   │       ├── checkout.yml
│   │       ├── item_added.yml
│   │       └── purchase.yml
│   └── device/                 # domain 2
│       ├── domain.yml
│       ├── info/
│       │   ├── hardware.yml
│       │   ├── software.yml
│       │   └── calibration.yml
│       ├── diagnostics/
│       │   └── stack_usage.yml
│       └── incident/
│           ├── drop.yml
│           └── temperature.yml
├── samples/                # JSON event payloads (one per event type)
├── scripts/
│   ├── demo.sh             # orchestrates the one-shot demo
│   ├── postgen.sh          # post-codegen package wrappers
│   └── verify.py           # reads Parquet output for `make verify`
└── services/
    ├── api/                # Go: Echo + SQS publisher
    └── consumer/           # Python: boto3 + Polars sink
```

## Troubleshooting

- **Port 8080 already in use.** `lsof -nP -iTCP:8080 -sTCP:LISTEN` and kill the
  offender; `make demo` builds the API into a temp file and cleans up on exit,
  but a leftover instance from a manual `make api` will block it.
- **LocalStack image download is slow.** First-run only; subsequent runs reuse
  the cached image.
- **`make verify` says "no parquet output"** before the consumer has run.
  Run `make consumer` first (or `make demo` which does both).
