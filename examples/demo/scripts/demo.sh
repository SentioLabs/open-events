#!/usr/bin/env bash
# End-to-end demo: gen, up, seed, send 3 events, drain consumer, verify, teardown.
set -euo pipefail

cd "$(dirname "$0")/.."   # examples/demo

API_PID=""
cleanup() {
  [[ -n "$API_PID" ]] && kill "$API_PID" 2>/dev/null || true
  docker compose down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

make gen
make up
make seed

(cd services/api && go run .) &
API_PID=$!

# Wait for /healthz to come up (max 30s).
for _ in $(seq 1 60); do
  if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then break; fi
  sleep 0.5
done

for sample in samples/*.json; do
  route="$(basename "$sample" .json)"
  echo "POST /v1/events/$route"
  curl -fsS -X POST "http://localhost:8080/v1/events/$route" \
    -H 'content-type: application/json' \
    --data-binary "@$sample"
  echo
done

echo "draining consumer..."
(cd services/consumer && uv run python -m consumer --until-empty)

make verify
