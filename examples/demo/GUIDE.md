# OpenEvents in practice: an annotated demo

This is a guided tour of a small but realistic OpenEvents project: a fictional
storefront emitting checkout and search events from a Go API, into SQS, into a
Python consumer that lands them in Parquet. Every component is referenced by
path so you can read it as you go.

The demo treats itself as a standalone project that happens to live in the
OpenEvents repository. To save you a `go install`, the in-tree Makefile invokes
the CLI via `go run`; you can read every `make gen` invocation as if it ran
`openevents` from your `$PATH`. Outside the repo, you would `go install
github.com/sentiolabs/open-events/cmd/openevents` once and forget the wrapper
exists.

> **Read first:** [`examples/demo/README.md`](./README.md) — prerequisites and
> the one-shot `make demo` runbook. This guide is the *why* and *how*; that's
> the *what to type*.

## Why OpenEvents

Most analytics events end up with three sources of truth: a wiki page where
product first specced them, a tracking-plan tool with PII tags, and code that
emits them. The three drift, and "what does `checkout.completed` mean?"
becomes a 30-minute archaeology project.

OpenEvents collapses that to one file. The registry — `registry/openevents.yaml`
here — is the only thing humans edit. Producers, consumers, warehouse schemas,
and documentation are *generated* from it. Drift between Go and Python isn't
prevented by code review; it's prevented by both sides reading from the same
YAML.

The MVP focuses on:

- Validating the registry on every change (`openevents validate`)
- Pinning field numbers across versions so wire format never breaks
  (`openevents lock check` + `openevents.lock.yaml`)
- Generating protobuf and downstream language bindings (`openevents generate
  proto`)
- Generating cross-language constants so producer and consumer share string
  identifiers (`openevents generate constants`)

Everything else — Avro, JSON Schema, Snowflake DDL, breaking-change diffs — is
roadmap.

## The registry

The registry is a directory tree, not a single file. This separates concerns and makes it easy for multi-team projects to own their own domains.

### Root file

[`registry/openevents.yaml`](./registry/openevents.yaml) defines the namespace, package paths, and code generation targets:

```yaml
openevents: 0.1.0
namespace: com.acme.platform

package:
  go: github.com/sentiolabs/open-events/examples/demo/services/api/eventmap
  python: consumer

owners:
  - { team: growth, email: growth-data@example.com, slack: "#team-growth" }
  - { team: device-platform, email: device-platform@example.com }

codegen:
  languages: [go, python]
  configs:
    go:
      out: ../services/api/eventmap
    python:
      out: ../services/consumer/src/consumer
```

The `namespace` is the proto package and the prefix everything generated derives from. The `package` block tells codegen where to write generated files in each language. The `codegen.languages` list determines which backends are targeted (here: Go and Python).

### Domain metadata

Each domain directory has a `domain.yml`:

```yaml
# registry/user/domain.yml
domain: user
description: User lifecycle and shopping events
owner: growth
```

Domain metadata is optional but helps organize multi-team registries.

### Event definitions

Each event lives in its own file under `<domain>/<action>/<action>.yml`:

```yaml
# registry/user/auth/signup.yml
event: user.auth.signup
version: 1
status: active
owner: growth
producer: storefront-api
destination:
  queue: storefront-events
  snowflake_table: fact_user_signup
context:
  - tenant_id
  - user_id
  - session_id
  - platform
properties:
  method:
    type: enum
    values: [email, sso, phone]
    required: true
  plan:
    type: string
    required: true
```

The file path encodes the domain and action hierarchy: `user/auth/signup.yml` becomes the event `user.auth.signup@1`. An event has a version, owner, producer, destinations (queue, warehouse table), required context fields, and a property bag.

### Context fields

Context fields are shared across all events (defined in the root file or inherited):

```yaml
context:
  tenant_id:
    type: string
    required: true
    pii: none
  user_id:
    type: string
    required: true
    pii: personal
  session_id:
    type: string
    pii: none
  platform:
    type: enum
    values: [ios, android, web, backend]
    required: true
    pii: none
```

Defining context once means every event gets the right tenant/user/session/platform fields for free, and a warehouse query like "events from the iOS app this week" doesn't need a per-event `JOIN`. PII tags are first-class: `personal`, `pseudonymous`, `sensitive`, or `none`. Downstream tooling can use them for masking, retention policies, or access control.

## The lock file

[`registry/openevents.lock.yaml`](./registry/openevents.lock.yaml) pins the
machine-significant identifiers that the YAML deliberately doesn't:

```yaml
events:
  checkout.completed@1:
    envelope:
      event_name:
        stable_id: event_name
        proto_number: 1
      event_version:
        proto_number: 2
      # ...
    properties:
      cart_id:
        proto_number: 1
      # ...
```

Protobuf field numbers are part of the wire format — once shipped, you cannot
change them without breaking every existing producer or consumer. The lock file
is the contract that says "field 5 means `total_cents` forever, even if the
YAML rearranges itself."

The workflow:

| Command | When | What it does |
|---------|------|--------------|
| `openevents lock check ./registry` | every CI run | Fails if the YAML and lock disagree |
| `openevents lock update ./registry` | after an approved schema change | Adds new fields with new numbers; fails if you try to renumber an existing field |

If a deletion is intentional, mark the field deprecated in the YAML and remove
it later in a separate, reviewed step.

## Code generation

`make gen` runs three steps in a single `openevents generate` invocation:

1. `openevents validate ./registry` — registry well-formedness
2. `openevents lock check ./registry` — wire-format compatibility  
3. `openevents generate ./registry` — emits protobuf files, Buf config, and cross-language constants

The `generate` command:
- Emits a `.proto` file + `buf.yaml` + `buf.gen.yaml` into `registry/.openevents/proto/`
- Uses `codegen.languages` from the root YAML to decide which language bindings to generate
- Writes Go constants to the path in `codegen.configs.go.out` (here: `services/api/eventmap/`)
- Writes Python constants to the path in `codegen.configs.python.out` (here: `services/consumer/src/consumer/`)

After `openevents generate`, the Makefile runs:

```bash
cd (.openevents/proto) && buf generate .
```

This produces `events.pb.go` and `events_pb2.py` via Buf's standard protoc plugins.

The Go bindings land in
[`_build/demo-proto/gen/go/`](./_build/demo-proto/) (gitignored) and the Python
bindings in `_build/demo-proto/gen/python/`. The Go side is consumed via a
`replace` directive (see `services/api/go.mod`); the Python side is consumed as
an editable path source declared in
[`services/consumer/pyproject.toml`](./services/consumer/pyproject.toml).

The generated **constants** are the cross-language glue. Event names are organized per domain. For the user domain:

```go
// services/api/eventmap/user/event_names.go (generated)
package user

const (
    AuthLoginV1 = "user.auth.login@1"
    AuthLogoutV1 = "user.auth.logout@1"
    AuthSignupV1 = "user.auth.signup@1"
    CartCheckoutV1 = "user.cart.checkout@1"
    CartItemAddedV1 = "user.cart.item_added@1"
    CartPurchaseV1 = "user.cart.purchase@1"
)
```

```python
# services/consumer/src/consumer/event_names/user.py (generated)
AUTH_LOGIN_V1 = "user.auth.login@1"
AUTH_LOGOUT_V1 = "user.auth.logout@1"
AUTH_SIGNUP_V1 = "user.auth.signup@1"
CART_CHECKOUT_V1 = "user.cart.checkout@1"
CART_ITEM_ADDED_V1 = "user.cart.item_added@1"
CART_PURCHASE_V1 = "user.cart.purchase@1"
```

A registry change → all files regenerate. There is no possible state where Go
and Python disagree about the canonical name of an event.

## The producer

The Go API is a small Echo service that accepts JSON, validates it against the
registry's shape, builds the proto envelope, and publishes to SQS.

Per-domain packages organize the route tables. [`services/api/eventmap/user/routes.go`](./services/api/eventmap/user/routes.go) defines routes for the user domain:

```go
func Routes() []eventmap.Route {
    return []eventmap.Route{
        {Path: "/v1/events/user/auth/signup",      EventName: AuthSignupV1,     Build: buildAuthSignup},
        {Path: "/v1/events/user/auth/login",       EventName: AuthLoginV1,      Build: buildAuthLogin},
        {Path: "/v1/events/user/auth/logout",      EventName: AuthLogoutV1,     Build: buildAuthLogout},
        {Path: "/v1/events/user/cart/checkout",    EventName: CartCheckoutV1,   Build: buildCartCheckout},
        {Path: "/v1/events/user/cart/item_added",  EventName: CartItemAddedV1,  Build: buildCartItemAdded},
        {Path: "/v1/events/user/cart/purchase",    EventName: CartPurchaseV1,   Build: buildCartPurchase},
    }
}
```

Each event file (e.g., [`services/api/eventmap/user/auth_signup.go`](./services/api/eventmap/user/auth_signup.go)) defines a request struct and build function:

```go
type AuthSignupRequest struct {
    Context Context `json:"context"`
    Method  string  `json:"method"`
    Plan    string  `json:"plan"`
}

func buildAuthSignup(c echo.Context) (*eventspb.UserAuthSignupV1, []FieldError, error) {
    var req AuthSignupRequest
    if err := c.Bind(&req); err != nil {
        return nil, nil, err
    }
    if errs := req.Validate(); len(errs) > 0 {
        return nil, errs, nil
    }
    return &eventspb.UserAuthSignupV1{
        EventName:    AuthSignupV1,
        EventVersion: 1,
        EventId:      newEventID(),
        EventTs:      newTimestamp(),
        Client:       newClient(),
        Context:      req.Context.toProto(),
        Properties: &eventspb.UserAuthSignupV1Properties{
            Method: proto.String(req.Method),
            Plan:   proto.String(req.Plan),
        },
    }, nil, nil
}
```

The server in [`services/api/server/server.go`](./services/api/server/server.go) collects routes from all domains and wires them:

```go
routes := append(user.Routes(), device.Routes()...)
for _, r := range routes {
    e.POST(r.Path, handle(pub, queueURL, r.EventName, r.Build))
}
```

Each route binds JSON → typed request, validates, builds the proto, then hands
off to `handle()` which marshals, base64-encodes, attaches SQS attributes, and
publishes via the
[`Publisher`](./services/api/publisher/publisher.go) interface. A `FakePublisher`
satisfies the same interface for tests.

## The wire format

The demo encodes each event as a base64 protobuf message in the SQS message
body, with two string attributes:

| Attribute | Value | Purpose |
|-----------|-------|---------|
| `event_name` | `checkout.started@1` | Routes the consumer to the right decoder *without* having to parse the body |
| `schema` | `com.acme.storefront/0.1.0` | Identifies the registry namespace and version that produced this message |

Encoding the routing key as a SQS attribute means a consumer can drop unknown
events, route to per-event-type DLQs, or count messages-by-type from CloudWatch
without ever touching a protobuf library. The body is opaque until you decide
you want it.

## The consumer

The Python consumer is intentionally small. Its job is to drain SQS, decode
each message, and write rows to per-event-type Parquet files.

### Dispatch

[`services/consumer/src/consumer/dispatch.py`](./services/consumer/src/consumer/dispatch.py)
maps the wire string to the generated protobuf class:

```python
EVENT_REGISTRY: dict[str, type] = {
    CHECKOUT_STARTED_V1: events_pb2.CheckoutStartedV1,
    CHECKOUT_COMPLETED_V1: events_pb2.CheckoutCompletedV1,
    SEARCH_PERFORMED_V1: events_pb2.SearchPerformedV1,
}

def decode(event_name: str, body_b64: str) -> dict[str, Any]:
    cls = EVENT_REGISTRY.get(event_name)
    if cls is None:
        raise ValueError(f"unknown event_name: {event_name!r}")
    wire = base64.b64decode(body_b64)
    msg = cls()
    msg.ParseFromString(wire)
    return MessageToDict(msg, preserving_proto_field_name=True)
```

The keys (e.g., `AUTH_SIGNUP_V1`) are imported from the generated
[`event_names/user.py`](./services/consumer/src/consumer/event_names/user.py) and [`event_names/device.py`](./services/consumer/src/consumer/event_names/device.py)
modules (one per domain). Renaming an event in the registry forces a regenerate, which forces a re-resolve here.

### Descriptor-driven schemas (the centerpiece)

[`services/consumer/src/consumer/schemas/user.py`](./services/consumer/src/consumer/schemas/user.py) and [`services/consumer/src/consumer/schemas/device.py`](./services/consumer/src/consumer/schemas/device.py)
derive Polars dataframe schemas *from the generated proto descriptors* at
import time:

```python
_SCALAR_DTYPES: dict[int, type[pl.DataType]] = {
    FieldDescriptor.TYPE_STRING: pl.Utf8,
    FieldDescriptor.TYPE_INT64: pl.Int64,
    FieldDescriptor.TYPE_BOOL: pl.Boolean,
    FieldDescriptor.TYPE_ENUM: pl.Utf8,  # MessageToDict emits enum names
    # ...
}

def _field_dtype(field: FieldDescriptor) -> pl.DataType:
    if field.type == FieldDescriptor.TYPE_MESSAGE:
        if field.message_type.full_name == "google.protobuf.Timestamp":
            dtype = pl.Utf8()  # MessageToDict emits RFC3339 strings
        else:
            dtype = _struct_from_descriptor(field.message_type)
    else:
        dtype = _SCALAR_DTYPES[field.type]()
    if field.is_repeated:
        return pl.List(dtype)
    return dtype

EVENT_SCHEMAS: dict[str, pl.Schema] = {
    CHECKOUT_STARTED_V1: _schema_for(events_pb2.CheckoutStartedV1),
    # ...
}
```

This is the payoff. **Add a field to the registry → it appears in the Parquet
output with the right dtype on the next consumer restart, no code change.**
The schema isn't transcribed by hand; it's read from the proto descriptor that
the registry produced. Nested messages map to `pl.Struct`, repeated fields to
`pl.List`, enums to `pl.Utf8` (because `MessageToDict` emits the enum *name*,
not its integer value).

### Poll loop and delivery semantics

[`services/consumer/src/consumer/sqs.py`](./services/consumer/src/consumer/sqs.py)
long-polls SQS, dispatches via the attribute, and appends to the sink. The
delivery posture:

- **At-least-once**: every successfully-processed message is `delete_message`-d
  immediately after the row is appended.
- **Poison messages are dropped** with a `log.exception` carrying the full
  traceback — a bad payload should not block the queue. In production you'd
  pair this with a SQS redrive policy so dropped messages land in a DLQ for
  inspection.
- **Long-poll cadence** is `WaitTimeSeconds=20`. The sink's flush interval is
  effectively bounded below by this, since `maybe_flush` runs once per receive.
  Higher-throughput workloads would move flushing onto a separate timer.

### Sink

[`services/consumer/src/consumer/sink.py`](./services/consumer/src/consumer/sink.py)
buffers rows in memory and flushes either when a batch fills or when
`flush_interval_s` elapses. Writes are atomic — the dataframe is written to
`<file>.parquet.tmp` and renamed via `os.replace`, so a crash mid-write cannot
corrupt a reader. Filenames carry a UTC timestamp and a monotonic counter, so
two flushes in the same second don't collide.

## Cross-language contracts in practice

Producer and consumer never call each other. They agree because they both read
from the registry:

```text
            registry/openevents.yaml
                       │
       ┌───────────────┼────────────────┐
       │               │                │
       ▼               ▼                ▼
 events.proto    event_names.go    event_names.py
       │               │                │
       ▼               ▼                ▼
 events.pb.go    Go API (Echo)    Python consumer
                       │                │
                       └──── SQS ───────┘
                            (event_name@vN
                             attribute routes)
```

The only string the producer hard-codes is the wire format choice
(`"event_name"`, `"schema"` as attribute names). The actual event names — the
strings that matter for routing, observability, and joining tables — come from
a generated file.

## End-to-end data flow

For one POST to `/v1/events/user/auth/signup`:

1. **Client** — `samples/user-auth-signup.json` POSTed via curl.
2. **API binds and validates** —
   [`user/auth_signup.go::buildAuthSignup`](./services/api/eventmap/user/auth_signup.go) decodes the JSON
   into `AuthSignupRequest` and calls `Validate()`.
3. **API builds the proto** — `buildAuthSignup` constructs `UserAuthSignupV1` with a
   generated `event_id` (UUIDv4), a server-side `event_ts`, the validated
   request fields, and the shared `Context`.
4. **API publishes** —
   [`publisher.go::SQSPublisher.Publish`](./services/api/publisher/publisher.go)
   marshals the proto, base64-encodes it into the SQS body, and attaches
   `event_name=user.auth.signup@1` and `schema=...` as SQS attributes.
5. **API responds** — `202 Accepted` with `{event_id, queue_url, message_id}`.
6. **Consumer long-polls** — `sqs.py::poll` receives the batch from LocalStack
   (or real SQS), reads the `event_name` attribute.
7. **Consumer dispatches and decodes** — `dispatch.py::decode` picks
   `events_pb2.UserAuthSignupV1`, parses the body, calls `MessageToDict`.
8. **Consumer appends to the sink** — `sink.py::Sink.append` buffers; on
   threshold or interval, `_write` materializes a Polars DataFrame using
   `EVENT_SCHEMAS[event_name]` and writes
   `_build/demo-output/user_auth_signup_v1/<ts>-<seq>.parquet`.
9. **Verify** — `make verify` reads back each per-event-type directory and
   prints the row count + the dataframe.

## What's intentionally out of scope

The demo is a teaching artifact, not a production starter. The following are
left as exercises:

- **Deduplication**: SQS is at-least-once. Every event carries an `event_id` so
  a real consumer can dedup; the demo intentionally doesn't.
- **DLQs**: poison messages are logged and dropped. Add a redrive policy in
  production.
- **Observability**: structured logs only. No metrics, no traces, no
  per-event-type counters.
- **Schema evolution**: the lock file pins compatibility, but the demo doesn't
  walk through a versioned migration (`checkout.started@1` → `@2`).
- **Real warehouse loading**: rows land in Parquet on local disk. A production
  pipeline would COPY into Snowflake using the `destination.snowflake_table`
  declared in the registry.

## Reading order

If you want to read the code itself rather than this guide, here's the order
that produces the fewest open questions:

1. [`registry/openevents.yaml`](./registry/openevents.yaml) — the root file (namespace, packages, codegen config)
2. [`registry/user/domain.yml`](./registry/user/domain.yml) and [`registry/device/domain.yml`](./registry/device/domain.yml) — domain metadata
3. [`registry/user/auth/signup.yml`](./registry/user/auth/signup.yml) — an example event definition
4. [`registry/openevents.lock.yaml`](./registry/openevents.lock.yaml) — the wire-compatibility pin
5. After `make gen`: peek at `registry/.openevents/proto/proto/com/acme/platform/v1/events.proto`
6. [`services/api/eventmap/user/routes.go`](./services/api/eventmap/user/routes.go) and [`services/api/eventmap/device/routes.go`](./services/api/eventmap/device/routes.go) — route tables per domain
7. [`services/api/eventmap/user/auth_signup.go`](./services/api/eventmap/user/auth_signup.go) — an example event handler
8. [`services/api/server/server.go`](./services/api/server/server.go) — HTTP wiring and domain route collection
9. [`services/api/publisher/publisher.go`](./services/api/publisher/publisher.go) — SQS publish
10. [`services/consumer/src/consumer/schemas.py`](./services/consumer/src/consumer/schemas.py) — descriptor → Polars schema (the most interesting file)
11. [`services/consumer/src/consumer/sqs.py`](./services/consumer/src/consumer/sqs.py) — poll, dispatch, delete
12. [`services/consumer/src/consumer/sink.py`](./services/consumer/src/consumer/sink.py) — batching and atomic writes

Then run `make demo` and tail the logs.
