#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

ABI_PATH="$TMPDIR/ProofDemo.abi.json"
BIN_PATH="$TMPDIR/ProofDemo.bin"
OUT_PATH="$ROOT/internal/e2e/bindings/proofdemo.go"

cd "$ROOT"
forge build
forge inspect --json contracts/ProofDemo.sol:ProofDemo abi > "$ABI_PATH"
forge inspect contracts/ProofDemo.sol:ProofDemo bytecode > "$BIN_PATH"
mkdir -p "$(dirname "$OUT_PATH")"
go tool github.com/ethereum/go-ethereum/cmd/abigen \
  --abi "$ABI_PATH" \
  --bin "$BIN_PATH" \
  --pkg bindings \
  --type ProofDemo \
  --out "$OUT_PATH"
