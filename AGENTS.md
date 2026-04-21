# AGENTS.md

## Scope

This repository is a Go-based Ethereum proof generator and verifier with a thin CLI wrapper. The core package is `proof`, and most behavior changes should happen there rather than in the CLI.

Start with [README.md](README.md) for proof semantics and CLI examples. Use [Makefile](Makefile) as the source of truth for common build and test commands.

## Project Layout

- `proof/`: core proof generation, verification, RPC normalization, JSON I/O, and tests.
- `cmd/ethproof/`: CLI entrypoint; keep it thin and delegate logic into `proof`.
- `cmd/mkfixtures/`: generator for deterministic offline fixtures under `proof/testdata/`.
- `contracts/ProofDemo.sol`: minimal contract used by local end-to-end tests.
- `internal/e2e/bindings/`: generated Go bindings for the demo contract. Do not hand-edit generated files.
- `scripts/generate_bindings.sh`: binding generation workflow.

## Core Architecture

The codebase supports exactly three proof flows, defined in [proof/types.go](proof/types.go):

- `StateProofPackage`: account proof plus one storage slot proof against `stateRoot`.
- `ReceiptProofPackage`: receipt inclusion proof plus event claim against `receiptsRoot`.
- `TransactionProofPackage`: transaction inclusion proof against `transactionsRoot`.

`cmd/ethproof` is only a flag parser and dispatcher. If a change affects proof correctness, serialization, or RPC behavior, update `proof/` first and keep CLI changes minimal.

## External Tooling

- Go `1.26.1`.
- `github.com/ethereum/go-ethereum v1.17.2` is the main protocol dependency.
- Foundry (`forge`) is used for the Solidity demo contract.
- Anvil is used for local e2e testing through [docker-compose.yml](docker-compose.yml).
- Contract bindings must be generated with geth `abigen v1` via [scripts/generate_bindings.sh](scripts/generate_bindings.sh), not with `abigen --v2`.

## Working Rules For Agents

- Do not weaken the multi-RPC consensus model. Live proof generation is intentionally strict and fails on any normalized mismatch.
- Preserve the public JSON shape in [proof/types.go](proof/types.go) unless the task explicitly requires a schema change.
- Prefer adding or fixing logic in `proof/` over duplicating behavior in `cmd/ethproof/`.
- Treat [internal/e2e/bindings/proofdemo.go](internal/e2e/bindings/proofdemo.go) as generated output. Regenerate it instead of editing it manually.
- When changing proof encoding, verification, or serialization, check whether offline fixtures in [proof/testdata](proof/testdata) need to be regenerated.
- When changing the Solidity demo contract, regenerate bindings and re-run the Anvil e2e path.

## Build And Test

Use these commands before finishing work:

- `make build`: build both CLI binaries into `bin/`.
- `make lint`: run `golangci-lint` with the configured linters.
- `make fmt-check`: check that `gofmt` formatting is correct.
- `make fmt`: apply `gofmt` and `forge fmt` to the codebase.
- `make test`: run the default Go test suite.
- `make fixtures`: regenerate deterministic fixtures.
- `make bindings`: rebuild Solidity artifacts and Go bindings.
- `make e2e`: regenerate bindings, start Anvil, and run the local end-to-end test.

Additional targeted commands:

- `go test ./proof -run TestAnvilE2E -count=1` with `ETH_PROOF_REQUIRE_E2E=1` for local e2e only.
- `make live-test` only when the required `ETH_PROOF_*` environment variables are intentionally provided.

## Change Guidance

- Proof logic changes: update `proof/` tests first or alongside the implementation.
- CLI flag changes: keep help text and README examples aligned.
- Fixture-affecting changes: run `make fixtures` and review the JSON diffs carefully.
- RPC-related changes: preserve strict normalization and consensus checks across all sources.

## Useful References

- [README.md](README.md): proof model, RPC consistency rules, CLI usage, and e2e overview.
- [Makefile](Makefile): supported build and test entrypoints.
- [foundry.toml](foundry.toml): Foundry project configuration.
- [scripts/generate_bindings.sh](scripts/generate_bindings.sh): exact binding generation steps.
