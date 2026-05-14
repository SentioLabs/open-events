#!/usr/bin/env bash
# End-to-end demo: gen, up, seed, send 3 events, drain consumer, verify, teardown.
set -euo pipefail

cd "$(dirname "$0")/.."   # examples/demo

API_BIN=""
API_PID=""
cleanup() {
  if [[ -n "$API_PID" ]]; then
    # API_PID is the compiled binary itself (we built then exec'd it), so a
    # single SIGTERM is enough. Falling back to SIGKILL covers the case where
    # the binary ignores SIGTERM.
    kill "$API_PID" 2>/dev/null || true
    for _ in $(seq 1 20); do
      kill -0 "$API_PID" 2>/dev/null || break
      sleep 0.1
    done
    kill -KILL "$API_PID" 2>/dev/null || true
  fi
  [[ -n "$API_BIN" ]] && rm -f "$API_BIN" || true
  docker compose down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

make gen
make up
make seed

# Build the API binary first so we control the actual PID. `go run .` would
# spawn the compiled binary as a child and leak it on cleanup.
API_BIN="$(mktemp -t demo-api.XXXXXX)"
(cd services/api && go build -o "$API_BIN" .)
"$API_BIN" &
API_PID=$!

# Wait for /healthz to come up (max 30s).
for _ in $(seq 1 60); do
  if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then break; fi
  sleep 0.5
done

while IFS= read -r sample; do
  # samples/ mirrors the registry tree, so the path inside samples/
  # (minus the .json extension) IS the URL route.
  route="${sample#samples/}"; route="${route%.json}"
  echo "POST /v1/events/$route"
  curl -fsS -X POST "http://localhost:8080/v1/events/$route" \
    -H 'content-type: application/json' \
    --data-binary "@$sample"
  echo
done < <(find samples -name '*.json' | sort)

echo "draining consumer..."
(cd services/consumer && uv run python -m consumer --until-empty)

make verify
