# AGENTS.md

## Scope

This repository is a Go-based Ethereum proof generator and verifier with a thin CLI wrapper. The core package is `proof`, and most behavior changes should happen there rather than in the CLI.

Start with [README.md](README.md) for proof semantics and CLI examples. Use [Makefile](Makefile) as the source of truth for common build and test commands.

## Project Layout

- `proof/`: core proof generation, verification, RPC normalization, JSON I/O, and tests.
- `proof/source_api.go`: source-injected interfaces and shared collection helpers used by embedders and tests.
- `proof/storage_layout.go` and `proof/storage_resolver.go`: compiler-output loading plus `resolve slot` logic for Solidity storage paths.
- `cmd/ethproof/`: CLI entrypoint; keep it thin and delegate logic into `proof`.
- `cmd/mkfixtures/`: generator for deterministic offline fixtures under `proof/testdata/`.
- `contracts/ProofDemo.sol`: minimal fixed-slot contract used by local end-to-end tests.
- `contracts/ProofComplexDemo.sol`: complex mapping/array/string/bytes contract used by local end-to-end tests.
- `internal/e2e/bindings/`: generated Go bindings for the demo contracts. Do not hand-edit generated files.
- `internal/proofutil/`: low-level trie, encoding, digest, and proof helpers shared by production code and tests.
- `scripts/generate_bindings.sh`: binding generation workflow.

## Core Architecture

The codebase supports exactly three proof flows, defined in [proof/types.go](proof/types.go):

- `StateProofPackage`: account proof plus one or more storage slot proofs against `stateRoot`.
- `ReceiptProofPackage`: receipt inclusion proof plus event claim against `receiptsRoot`.
- `TransactionProofPackage`: transaction inclusion proof against `transactionsRoot`.

The storage-slot resolver in [proof/storage_layout.go](proof/storage_layout.go) and [proof/storage_resolver.go](proof/storage_resolver.go) is an auxiliary path for `ethproof resolve slot`; it is not a fourth proof type.

`cmd/ethproof` is only a flag parser and dispatcher. If a change affects proof correctness, serialization, RPC behavior, or storage-slot resolution semantics, update `proof/` first and keep CLI changes minimal.

## External Tooling

- Go `1.26.2` or the latest compatible version.
- `github.com/ethereum/go-ethereum` is the main protocol dependency.
- Foundry (`forge`) is used for the Solidity demo contract.
- Anvil is used for local e2e testing through [docker-compose.yml](docker-compose.yml).
- Contract bindings must be generated with geth `abigen v1` via [scripts/generate_bindings.sh](scripts/generate_bindings.sh), not with `abigen --v2`.

## Working Rules For Agents

- Do not weaken the multi-RPC consensus model. Live proof generation is intentionally strict and fails on any normalized mismatch.
- Preserve the public JSON shape in [proof/types.go](proof/types.go) unless the task explicitly requires a schema change.
- Prefer adding or fixing logic in `proof/` over duplicating behavior in `cmd/ethproof/`.
- Treat [internal/e2e/bindings/proofdemo.go](internal/e2e/bindings/proofdemo.go) and [internal/e2e/bindings/proofcomplexdemo.go](internal/e2e/bindings/proofcomplexdemo.go) as generated output. Regenerate them instead of editing them manually.
- When changing proof encoding, verification, or serialization, check whether offline fixtures in [proof/testdata](proof/testdata) need to be regenerated.
- When changing the Solidity demo contract, regenerate bindings and re-run the Anvil e2e path.

## Build And Test

Use these commands before finishing work:

- `make build`: build both CLI binaries (`ethproof` and `mkfixtures`) into `bin/`.
- `make lint`: run `golangci-lint` with the configured linters.
- `make fmt-check`: check that `gofmt` formatting is correct.
- `make fmt`: apply `gofmt` and `forge fmt` to the codebase.
- `make fixtures`: regenerate deterministic proof fixtures.
- `make bindings`: rebuild Solidity artifacts and Go bindings.
- `make unit-test`: run the default offline-stable Go test suite with `go test -v -race ./...`.
- `make e2e-test`: start Anvil via `docker compose` and run only the local `TestAnvilE2E` path.
- `make e2e`: regenerate bindings if needed, then run the e2e path.
- `make test`: run the full suite (`make unit-test` + `make e2e-test`).

## Change Guidance

- Proof logic changes: update `proof/` tests first or alongside the implementation.
- CLI flag changes: keep help text and README examples aligned.
- Fixture-affecting changes: run `make fixtures` and review the JSON diffs carefully.
- RPC-related changes: preserve strict normalization and consensus checks across all sources.

## Useful References

- [README.md](README.md): proof model, RPC consistency rules, CLI usage, and e2e overview.
- [design.md](design.md): internal architecture, source-driven APIs, resolver flow, and test layers.
- [Makefile](Makefile): supported build and test entrypoints.
- [config.example.json](config.example.json): example config for `generate` / `verify`.
- [foundry.toml](foundry.toml): Foundry project configuration.
- [docker-compose.yml](docker-compose.yml): local Anvil environment used by `make e2e-test`.
- [scripts/generate_bindings.sh](scripts/generate_bindings.sh): exact binding generation steps.
