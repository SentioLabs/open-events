#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="$(tr -d '[:space:]' < "$ROOT/.tools/buf.version")"
GOBIN="$ROOT/.tools/bin"

if [[ -z "$VERSION" ]]; then
  echo "missing Buf version in .tools/buf.version" >&2
  exit 1
fi

mkdir -p "$GOBIN"
echo "installing buf $VERSION to $GOBIN/buf"
GOBIN="$GOBIN" go install "github.com/bufbuild/buf/cmd/buf@$VERSION"
"$GOBIN/buf" --version
