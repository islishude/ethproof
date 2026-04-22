#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

generate_binding() {
  fqcn="$1"
  type_name="$2"
  out_name="$3"
  abi_path="$TMPDIR/$type_name.abi.json"
  bin_path="$TMPDIR/$type_name.bin"
  out_path="$ROOT/internal/e2e/bindings/$out_name"

  forge inspect --json "$fqcn" abi > "$abi_path"
  forge inspect "$fqcn" bytecode > "$bin_path"
  go tool github.com/ethereum/go-ethereum/cmd/abigen \
    --abi "$abi_path" \
    --bin "$bin_path" \
    --pkg bindings \
    --type "$type_name" \
    --out "$out_path"
}

cd "$ROOT"
forge build
mkdir -p "$ROOT/internal/e2e/bindings"
generate_binding contracts/ProofDemo.sol:ProofDemo ProofDemo proofdemo.go
generate_binding contracts/ProofComplexDemo.sol:ProofComplexDemo ProofComplexDemo proofcomplexdemo.go
