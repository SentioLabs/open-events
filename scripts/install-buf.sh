#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUF_VERSION="$(tr -d '[:space:]' < "$ROOT/.tools/buf.version")"
PROTOC_GEN_GO_VERSION="$(tr -d '[:space:]' < "$ROOT/.tools/protoc-gen-go.version")"
GOBIN="$ROOT/.tools/bin"

if [[ -z "$BUF_VERSION" ]]; then
  echo "missing Buf version in .tools/buf.version" >&2
  exit 1
fi
if [[ -z "$PROTOC_GEN_GO_VERSION" ]]; then
  echo "missing protoc-gen-go version in .tools/protoc-gen-go.version" >&2
  exit 1
fi

mkdir -p "$GOBIN"
echo "installing buf $BUF_VERSION to $GOBIN/buf"
GOBIN="$GOBIN" go install "github.com/bufbuild/buf/cmd/buf@$BUF_VERSION"
"$GOBIN/buf" --version

echo "installing protoc-gen-go $PROTOC_GEN_GO_VERSION to $GOBIN/protoc-gen-go"
GOBIN="$GOBIN" go install "google.golang.org/protobuf/cmd/protoc-gen-go@$PROTOC_GEN_GO_VERSION"
"$GOBIN/protoc-gen-go" --version
