# AGENTS.md

Start with [README.md](README.md) for usage and proof semantics, [design.md](design.md) for architecture, and [Makefile](Makefile) for build/test commands. Tool versions and compiler settings are defined in [go.mod](go.mod) and [foundry.toml](foundry.toml).

## Code boundaries

- Keep proof correctness, serialization, RPC behavior, and storage resolution in `proof/`. `cmd/ethproof/` is a thin parser and dispatcher.
- The three proof types are state, receipt/event, and transaction; storage resolution is auxiliary. Public models live in [proof/types.go](proof/types.go), source interfaces in [proof/source.go](proof/source.go).
- Share low-level encoding and trie helpers through `internal/proofutil/`. `cmd/mkfixtures/` owns deterministic fixture generation.

## Invariants

- Each live operation uses one RPC. Pass the source or URL separately from proof inputs; preserve response consistency checks and local cryptographic verification.
- Online verification checks the complete block context against its separately configured RPC. Never implicitly reuse generation configuration.
- Preserve the distinction between embedded-root consistency and caller-trusted header verification. Neither establishes consensus finality.
- Preserve cancellation, credential redaction, and bounded receipt-scan concurrency.
- Preserve the public proof JSON shape unless the task explicitly requires a schema change.
- Never hand-edit `internal/e2e/bindings/`. Regenerate through [scripts/generate_bindings.sh](scripts/generate_bindings.sh), using geth `abigen v1`, not `--v2`.

## Validation

- Update relevant tests alongside proof or RPC changes. Run `make build`, `make lint`, `make fmt-check`, and `make unit-test` for code changes; use `make fmt` to apply formatting and Go fixes.
- Run `make e2e-test` for changes affecting live generation, verification, or CLI flows. It manages Anvil through Docker Compose.
- For encoding, verification, or serialization changes, check fixture impact. Run `make fixtures` when needed and review JSON diffs.
- After Solidity contract changes, run `make bindings` and `make e2e-test` (or `make e2e`).
- Keep CLI help, README examples, and [config.example.json](config.example.json) aligned with interface changes.
- For documentation-only changes, check links, examples, and `git diff --check`.
