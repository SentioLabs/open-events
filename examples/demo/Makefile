# OpenEvents demo orchestration. Run targets from this directory.
SHELL := /bin/bash

REPO_ROOT       := $(abspath ../..)
DEMO_ROOT       := $(abspath .)
REGISTRY        := ./examples/demo/registry
PROTO_ROOT      := $(DEMO_ROOT)/registry/.openevents/proto
GEN_GO          := $(DEMO_ROOT)/gen/go
GEN_PYTHON      := $(DEMO_ROOT)/gen/python
PARQUET_OUT     := $(REPO_ROOT)/_build/demo-output
QUEUE_NAME      := storefront-events
QUEUE_URL       := http://localhost:4566/000000000000/$(QUEUE_NAME)
ENDPOINT_URL    := http://localhost:4566

export AWS_ENDPOINT_URL    ?= $(ENDPOINT_URL)
export AWS_REGION          ?= us-east-1
export AWS_ACCESS_KEY_ID   ?= test
export AWS_SECRET_ACCESS_KEY ?= test
export OPENEVENTS_QUEUE_URL ?= $(QUEUE_URL)
export OPENEVENTS_OUTPUT_DIR ?= $(PARQUET_OUT)

.PHONY: gen up wait-sqs seed api consumer demo verify down clean test

gen:
	cd $(REPO_ROOT) && go run ./cmd/openevents validate $(REGISTRY)
	cd $(REPO_ROOT) && go run ./cmd/openevents lock check $(REGISTRY)
	rm -rf $(PROTO_ROOT)
	# Clear only the generated proto trees so committed package metadata
	# (gen/go/go.mod, gen/go/go.sum, gen/python/pyproject.toml) survives.
	# Keeps `git clone && make` reproducible — see `examples/demo/CLAUDE.md`.
	rm -rf $(GEN_GO)/com $(GEN_PYTHON)/com
	cd $(REPO_ROOT) && go run ./cmd/openevents generate $(REGISTRY)
	cd $(REPO_ROOT) && bash scripts/install-buf.sh
	# Bootstrap gen/go/go.mod from the template only when missing (e.g.,
	# after `make clean`). On a fresh checkout it's already committed.
	@if [ ! -f $(GEN_GO)/go.mod ]; then \
		mkdir -p $(GEN_GO) && cp $(DEMO_ROOT)/gen-go.mod.template $(GEN_GO)/go.mod; \
	fi
	cd $(DEMO_ROOT) && PATH="$(REPO_ROOT)/.tools/bin:$$PATH" $(REPO_ROOT)/.tools/bin/buf generate
	bash $(DEMO_ROOT)/scripts/postgen.sh $(GEN_PYTHON)

up:
	docker compose up -d localstack
	$(MAKE) wait-sqs

wait-sqs:
	@echo "waiting for LocalStack SQS..."
	@for i in $$(seq 1 60); do \
		if curl -fsS $(ENDPOINT_URL)/_localstack/health 2>/dev/null | grep -q '"sqs": "available"'; then \
			echo "ok"; exit 0; \
		fi; sleep 1; \
	done; echo "timed out" >&2; exit 1

seed:
	docker compose exec -T localstack awslocal sqs create-queue --queue-name $(QUEUE_NAME) >/dev/null
	@echo "queue ready: $(QUEUE_NAME)"

api:
	cd services/api && go run .

consumer:
	cd services/consumer && uv run python -m consumer

demo:
	rm -rf $(PARQUET_OUT)
	bash scripts/demo.sh

verify:
	@uv run --with polars python scripts/verify.py $(PARQUET_OUT)

test:
	cd services/api && go test ./...
	cd services/consumer && uv run pytest

down:
	docker compose down -v

clean:
	rm -rf $(PARQUET_OUT) $(REPO_ROOT)/_build/localstack
	rm -rf $(GEN_GO) $(GEN_PYTHON)
