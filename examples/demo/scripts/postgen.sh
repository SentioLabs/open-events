#!/usr/bin/env bash
# Post-process buf-generated Python output: turn gen/python into an
# installable Python package by adding __init__.py files and a minimal
# pyproject.toml. The Go side needs no post-processing now that
# gen/go is its own Go module with services/api importing it via a
# replace directive — buf writes files directly where they're imported.
set -euo pipefail

PY_GEN_DIR="${1:?usage: postgen.sh <python-gen-dir>}"

if [[ ! -d "$PY_GEN_DIR" ]]; then
  echo "postgen: $PY_GEN_DIR not found; run buf generate first" >&2
  exit 1
fi

find "$PY_GEN_DIR" -type d -exec touch {}/__init__.py \;

if [[ -f "$PY_GEN_DIR/pyproject.toml" ]]; then
  echo "postgen: $PY_GEN_DIR/pyproject.toml already exists (preserved)"
else
  cat > "$PY_GEN_DIR/pyproject.toml" <<'PYTOML'
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
  echo "postgen: $PY_GEN_DIR is now an installable package"
fi
