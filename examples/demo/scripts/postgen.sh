#!/usr/bin/env bash
# Turn _build/demo-proto/gen/python into an installable Python package
# so the consumer's pyproject.toml path source can build a wheel.
set -euo pipefail

GEN_DIR="${1:-_build/demo-proto/gen/python}"
if [[ ! -d "$GEN_DIR" ]]; then
  echo "postgen: $GEN_DIR not found; run buf generate first" >&2
  exit 1
fi

find "$GEN_DIR" -type d -exec touch {}/__init__.py \;

cat > "$GEN_DIR/pyproject.toml" <<'PYTOML'
[project]
name = "openevents-demo-pb2"
version = "0.0.0"
requires-python = ">=3.11"
dependencies = ["protobuf>=5.0"]

[tool.setuptools.packages.find]
where = ["."]
include = ["com*"]

[build-system]
requires = ["setuptools>=68"]
build-backend = "setuptools.build_meta"
PYTOML

echo "postgen: $GEN_DIR is now an installable package"
