#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
compose() { docker compose -f internal/e2e/pos/compose.yaml "$@"; }
up() {
  mkdir -p bin/pos-testnet/network
  compose up --quiet-pull -d
  compose cp genesis:/data/metadata/config.yaml bin/pos-testnet/network/config.yaml
  compose cp genesis:/data/metadata/genesis.ssz bin/pos-testnet/network/genesis.ssz
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
      compose logs --no-color > bin/pos-testnet/compose.log 2>&1 || true
      if [ "${POS_E2E_KEEP:-0}" != 1 ]; then compose down -v --remove-orphans || true; fi
      exit "$status"
    }
    mkdir -p bin/pos-testnet
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    up
    ETH_PROOF_REQUIRE_POS_E2E=1 go test -v -race -count=1 -timeout=17m ./proof -run '^TestPoSE2E$'
    ;;
  *) echo 'usage: scripts/pos_e2e.sh up|down|test' >&2; exit 2 ;;
esac
