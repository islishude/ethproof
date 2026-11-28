#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export POS_FORK=${POS_FORK:-fulu}
project=ethproof-pos-e2e
out=bin/pos-testnet
case "$POS_FORK" in
  fulu) ;;
  gloas)
    project=ethproof-pos-gloas
    out=bin/pos-gloas
    export POS_RETH_TAG=v2.11.0 POS_GENESIS_TAG=6.2.10 POS_PORT_PREFIX=2
    # Lighthouse commit 4cf72c3 serves Gloas light-client updates.
    export POS_LIGHTHOUSE_TAG=latest-unstable@sha256:b215e5264b06a37afb509cb20029494bfde24a8a64dc9f5d30c17ca3288b9225
    ;;
  *) echo "unknown PoS fork: $POS_FORK" >&2; exit 2 ;;
esac
compose() { docker compose -p "$project" -f internal/e2e/pos/compose.yaml "$@"; }
up() {
  mkdir -p "$out/network"
  compose up --quiet-pull -d
  compose cp genesis:/data/metadata/config.yaml "$out/network/config.yaml"
  compose cp genesis:/data/metadata/genesis.ssz "$out/network/genesis.ssz"
}
case "${1:-}" in
  up) up ;;
  down) compose down -v --remove-orphans ;;
  test)
    # Own only this project's disposable nodes/volumes; Anvil is independent.
    compose down -v --remove-orphans
    cleanup() {
      status=$?
      trap - EXIT
      compose logs --no-color > "$out/compose.log" 2>&1 || true
      if [ "${POS_E2E_KEEP:-0}" != 1 ]; then compose down -v --remove-orphans || true; fi
      exit "$status"
    }
    mkdir -p "$out"
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    up
    ETH_PROOF_REQUIRE_POS_E2E=1 go test -v -race -count=1 -timeout=17m ./proof -run '^TestPoSE2E$'
    ;;
  *) echo 'usage: scripts/pos_e2e.sh up|down|test' >&2; exit 2 ;;
esac
